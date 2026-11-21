package agent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/llm"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

func reportFindingsTool() llm.Tool {
	return llm.Tool{
		Name:        "report_findings",
		Description: "Report your critique findings against the PRD.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"findings": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"section":    map[string]any{"type": "string"},
							"severity":   map[string]any{"type": "string", "enum": []any{"blocking", "major", "minor"}},
							"kind":       map[string]any{"type": "string", "enum": []any{"defect", "omission"}},
							"quote":      map[string]any{"type": "string", "description": "required for kind=defect: the exact PRD text motivating this finding, verbatim. Omit for kind=omission."},
							"issue":      map[string]any{"type": "string"},
							"suggestion": map[string]any{"type": "string"},
						},
						"required": []any{"section", "severity", "kind", "issue"},
					},
				},
			},
			"required": []any{"findings"},
		},
	}
}

const criticSystemPromptTemplate = `你是一位严格的 PRD 评审员，正在独立评审一份 PRD——你没有看过起草过程中的对话，只看到 PRD 本身和已记录的事实清单。你的评审基于以下标准：

1. 目标用户是否具体（不能是"所有人"）？
2. 问题是否从用户视角清晰陈述？
3. 方案是否真的解决了这个问题？
4. 是否说明了现有替代方案，以及为什么不够好？
5. 目标是否可衡量？
6. requirements 章节是否描述的是问题而非预设方案（Cagan 的 "what vs how"）？
7. 一个工程师能否根据这份 PRD 构建，一个 QA 能否据此写测试计划？

对每一节都要评估，不能跳过。每条 finding 分两种：kind="defect" 表示章节里有具体问题，quote 参数必须是该章节的原文片段；kind="omission" 表示章节缺少某项标准要求的内容，不需要 quote。severity 分 blocking/major/minor。必须通过 report_findings 工具报告，不要用自然语言回复。
%s

%s

当前 PRD 标题：%s
当前各章节内容：
%s
已记录的事实清单：
%s`

// parseCriticFindings applies the verification gate: a defect finding
// whose quote isn't actually grounded in the PRD text is dropped (the
// hallucination it would otherwise let through), not the whole batch.
func parseCriticFindings(argsJSON string, sections map[string]string) ([]CriticFinding, error) {
	var args struct {
		Findings []CriticFinding `json:"findings"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("invalid report_findings arguments: %w", err)
	}

	var allSections strings.Builder
	for _, k := range store.PrdSectionKeys {
		allSections.WriteString(sections[k])
		allSections.WriteString("\n")
	}
	full := allSections.String()

	var out []CriticFinding
	for _, f := range args.Findings {
		if f.Issue == "" || (f.Severity != "blocking" && f.Severity != "major" && f.Severity != "minor") {
			continue
		}
		if !isPrdSection(f.Section) {
			continue // section key outside the canonical eight — drop, don't fail the batch
		}
		if f.Kind != "defect" && f.Kind != "omission" {
			continue
		}
		if f.Kind == "defect" && (f.Quote == "" || !quoteIsGrounded(f.Quote, full)) {
			continue // unverifiable — the anti-hallucination gate this whole mechanism exists for
		}
		f.ID = "c" + uuid.NewString()[:8]
		f.Status = FindingOpen
		out = append(out, f)
	}
	return out, nil
}

// rankedCritiqueLine matches one item of the critique-stage overview:
// "1. [major] goals — 问题。建议：建议" — the suggestion is optional.
var rankedCritiqueLine = regexp.MustCompile(
	`^\s*\d+\.\s*\[(blocking|major|minor)\]\s*([A-Za-z]+)\s*[—–-]\s*(.+?)\s*$`,
)

// isPrdSection reports whether key is one of the 8 canonical PRD section
// keys — a finding against anything else points the round-2 critic and
// the panel at a section that doesn't exist, so both parsers drop it
// (the same drop-don't-fail policy used for bad severity/kind).
func isPrdSection(key string) bool {
	return slices.Contains(store.PrdSectionKeys, key)
}

// parseRankedCritique turns the coach's severity-ranked overview into the
// findings the panel renders. Items stay in the order written (most severe
// first); a line that doesn't match the ranked format, or names a section
// outside the canonical eight, is skipped.
func parseRankedCritique(reply string) []CriticFinding {
	var out []CriticFinding
	for _, line := range strings.Split(reply, "\n") {
		m := rankedCritiqueLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		if !isPrdSection(m[2]) {
			continue // unknown section key — drop, don't fail the batch
		}
		issue, suggestion := m[3], ""
		// Both markers are accepted: the zh prompt's 「建议：」 and the en
		// prompt's "Suggestion:" (prompts_en.go) — the parser is locale-blind.
		for _, marker := range []string{"建议：", "Suggestion:"} {
			if i := strings.Index(issue, marker); i >= 0 {
				suggestion = strings.TrimSpace(issue[i+len(marker):])
				issue = strings.TrimSpace(issue[:i])
				break
			}
		}
		out = append(out, CriticFinding{
			ID:         "c" + uuid.NewString()[:8],
			Section:    m[2],
			Severity:   m[1],
			Kind:       "omission",
			Issue:      issue,
			Suggestion: suggestion,
			Status:     FindingOpen,
		})
	}
	return out
}
