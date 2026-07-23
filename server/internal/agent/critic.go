package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// maybeRunCritic implements the fresh-context critic pass (plan §4.4):
// it runs at critique-stage entry and after one applied revision round,
// bounded at maxCriticRounds, and never sees the chat history — only the
// PRD and the fact ledger. Under queue pressure (waited) it degrades
// instead of running, recording a critic-skipped marker so the finalize
// gate isn't hostage to load (plan §5.1). The critic inherits this
// turn's already-held concurrency permit; it never acquires its own.
func (c *Coach) maybeRunCritic(ctx context.Context, actorID, prdID, conversationID, runID uuid.UUID, stage string, sectionUpdated, waited bool, sse *sseWriter) {
	if stage != StageCritique {
		return
	}
	conv, err := c.store.GetConversationForOwner(ctx, conversationID, actorID)
	if err != nil {
		return
	}
	lm, extra := loadLedger(conv.Meta)
	cs := loadCriticState(extra)
	if cs.RoundCount >= maxCriticRounds {
		return
	}
	if !(cs.RoundCount == 0 || (cs.RoundCount == 1 && sectionUpdated)) {
		return
	}

	markSkipped := func(reason string) {
		if cs.Skipped != "" {
			return // already recorded this round's skip — don't keep re-writing
		}
		cs.Skipped = reason
		saveCriticState(extra, cs)
		if metaJSON, err := saveLedger(lm, extra); err == nil {
			_ = c.store.UpdateConversationMeta(ctx, actorID, conversationID, metaJSON)
		}
	}

	if waited {
		markSkipped("contention")
		return
	}

	prd, err := c.store.GetPrdForMember(ctx, prdID, actorID)
	if err != nil {
		return
	}
	var focus []string
	if cs.RoundCount == 1 {
		for _, f := range loadCriticFindings(extra) {
			if f.Status == FindingOpen {
				focus = append(focus, f.Section)
			}
		}
	}
	started := time.Now()
	findings, usage, ok := runCritic(ctx, c.streamer, prd.Title, prd.Sections, lm, focus)
	var callErr error
	if !ok {
		callErr = fmt.Errorf("critic output failure")
	}
	c.recordTelemetry(ctx, runID, "critic", started, usage, callErr)
	if !ok {
		markSkipped("critic-output-failure")
		return
	}

	merged := append(loadCriticFindings(extra), findings...)
	saveCriticFindings(extra, merged)
	cs.RoundCount++
	saveCriticState(extra, cs)
	metaJSON, err := saveLedger(lm, extra)
	if err != nil {
		c.log.Error("save critic findings", "error", err)
		return
	}
	if err := c.store.UpdateConversationMeta(ctx, actorID, conversationID, metaJSON); err != nil {
		c.log.Error("persist critic findings", "error", err)
		return
	}
	sse.criticFindings(findings)
}

// CriticFinding is one item from a critic pass, persisted in
// agent_conversations.meta.critic_findings — the critique→finalize gate
// reads this list, never the model's say-so (plan §4.4).
type CriticFinding struct {
	ID         string `json:"id"`
	Section    string `json:"section"`
	Severity   string `json:"severity"` // blocking | major | minor
	Kind       string `json:"kind"`     // defect (quotes the PRD) | omission (names what's missing)
	Quote      string `json:"quote,omitempty"`
	Issue      string `json:"issue"`
	Suggestion string `json:"suggestion,omitempty"`
	Status     string `json:"status"` // open | resolved | overridden
}

const (
	FindingOpen       = "open"
	FindingResolved   = "resolved"
	FindingOverridden = "overridden"

	maxCriticRounds = 2
)

func loadCriticFindings(extra map[string]json.RawMessage) []CriticFinding {
	var out []CriticFinding
	if v, ok := extra["critic_findings"]; ok {
		_ = json.Unmarshal(v, &out)
	}
	return out
}

func saveCriticFindings(extra map[string]json.RawMessage, findings []CriticFinding) {
	b, err := json.Marshal(findings)
	if err != nil {
		return
	}
	extra["critic_findings"] = b
}

// criticState is the small bit of round-tracking the finalize gate and
// the round cap need, stored alongside the findings under the same
// "critic_round_count" / "critic_skipped" extra keys.
type criticState struct {
	RoundCount int    `json:"roundCount"`
	Skipped    string `json:"skipped,omitempty"` // reason, e.g. "contention" — empty means not skipped
}

func loadCriticState(extra map[string]json.RawMessage) criticState {
	var cs criticState
	if v, ok := extra["critic_state"]; ok {
		_ = json.Unmarshal(v, &cs)
	}
	return cs
}

func saveCriticState(extra map[string]json.RawMessage, cs criticState) {
	b, err := json.Marshal(cs)
	if err != nil {
		return
	}
	extra["critic_state"] = b
}

func hasOpenBlockingFindings(findings []CriticFinding) bool {
	for _, f := range findings {
		if f.Status == FindingOpen && f.Severity == "blocking" {
			return true
		}
	}
	return false
}

func reportFindingsTool() Tool {
	return Tool{
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

当前 PRD 标题：%s
当前各章节内容：
%s
已记录的事实清单：
%s`

// runCritic makes one fresh-context critic call (no chat history — only
// the PRD and the ledger, per plan §4.4) and returns verified findings.
// Output-shape failures (unparseable/empty) get one re-prompt retry with
// the error appended; a second failure returns ok=false so the caller
// can degrade to single-context critique rather than fail the turn.
func runCritic(ctx context.Context, streamer ChatStreamer, prdTitle string, sections map[string]string, lm ledgerMeta, focusSections []string) (findings []CriticFinding, usage Usage, ok bool) {
	focus := ""
	if len(focusSections) > 0 {
		focus = "\n\n本轮只需重新评审以下此前被标记过的章节：" + strings.Join(focusSections, ", ")
	}
	system := fmt.Sprintf(criticSystemPromptTemplate, focus, prdTitle, formatSectionsForPrompt(sections), formatLedgerForPrompt(lm))
	tool := reportFindingsTool()

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		sys := system
		if lastErr != nil {
			sys += fmt.Sprintf("\n\n上一次调用失败：%s。请重新输出，严格符合 report_findings 的参数格式。", lastErr.Error())
		}
		turn, err := streamer.StreamChatForcingTool(ctx, sys, nil, tool, nil)
		usage = turn.Usage
		if err != nil {
			lastErr = err
			continue
		}
		if turn.ToolCall == nil || turn.ToolCall.Name != tool.Name {
			lastErr = fmt.Errorf("model did not call report_findings")
			continue
		}
		parsed, perr := parseCriticFindings(turn.ToolCall.Arguments, sections)
		if perr != nil {
			lastErr = perr
			continue
		}
		return parsed, usage, true
	}
	return nil, usage, false
}

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
