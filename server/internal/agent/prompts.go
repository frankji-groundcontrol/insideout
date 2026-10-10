package agent

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/readiness"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// Stage names and legal forward transitions, per
// docs/plans/2026-07-20-go-rewrite/03-agents.md §3. finalize has no next.
const (
	StageClarify  = "clarify"
	StageDraft    = "draft"
	StageCritique = "critique"
	StageFinalize = "finalize"
)

// The two locales a conversation can coach in, as stored on
// agent_conversations (store.NormalizeLocale canonicalizes). Anything but
// 'en' — zh included — keeps the Chinese prompts, the long-standing
// behavior for conversations created before the locale column existed.
// FEAT-01 (review 3): PRODUCT.md's first-class product languages.
const (
	localeEN = "en"
	localeZH = "zh"
)

var stageOrder = []string{StageClarify, StageDraft, StageCritique, StageFinalize}

// nextStage returns the only stage `from` may advance to, or "" if there
// is none (finalize, or an unrecognized stage).
func nextStage(from string) string {
	for i, s := range stageOrder {
		if s == from && i+1 < len(stageOrder) {
			return stageOrder[i+1]
		}
	}
	return ""
}

const toolCallDiscipline = `你每次回复只能进行以下两种动作之一：要么输出一段面向用户的自然语言文本，要么调用一个工具。绝不在同一轮回复中既输出文本又调用工具，也绝不在同一轮里调用多个工具——每次最多一个工具调用。`

const factDiscipline = `记录事实的规矩：用户明确说出的内容——问题、目标用户、现有替代方案、时机、目标、约束——用 record_fact 记下，quote 参数必须是用户的原话，不能是你的转述或推测。你自己提出但用户还没确认的内容，用 mark_assumption 记下，绝不能当作事实呈现。禁止编造用户没说过的用户画像、数据或需求。`

// recommendationDiscipline carries PRODUCT.md principle 2 (truth over
// document completion, lines 47-48) into every stage prompt: the Coach
// may recommend continuing, validating, pausing, or abandoning a
// direction, and the human always decides.
const recommendationDiscipline = `作为追求真相的伙伴，你可以建议继续深化、先去验证、暂停、或者放弃当前方向；但最终怎么做始终由用户决定。`

// systemPrompt assembles the coach's system prompt in the conversation's
// locale: 'en' renders the English variants (prompts_en.go), anything else
// the Chinese ones below — the zh default is the back-compat behavior for
// conversations created before the locale column existed.
func systemPrompt(locale string, stage string, prdTitle string, sections map[string]string, ledgerText string, findingsText string) string {
	if locale == localeEN {
		return systemPromptEN(stage, prdTitle, sections, ledgerText, findingsText)
	}
	var body string
	switch stage {
	case StageClarify:
		body = `你是 InsideOut 的 PRD 教练，正在「澄清」阶段帮用户把一个想法说清楚。
按想法的复杂度调整访谈的深浅：不设固定的问题数量，也不强求固定的提问顺序。相互独立的问题可以合并成一组一次提出；只有当一个答案会改变访谈的下一个分支时，才把那一个问题单独先问。
必须问清楚：这个想法要解决什么问题？谁会因为这个问题受困扰（具体到一类人，不能是「所有人」）？现在他们怎么应付（现有替代方案，以及为什么这些替代方案不够好）？为什么是现在做这件事？
你提出的每个问题都要说明三点：优先级（必须现在澄清 / 本版应澄清 / 之后再验证）、这个答案主要服务于谁（哪类读者或决策）、以及为什么现在问它（参考下方「当前读者缺口」）。用户随时可以说「现在成版」——你不可以阻止，只需说明会带着哪些未决事项成版。
用户每次回答后，如果回答里包含了以上任一问题的答案，调用 record_fact 记下（quote 必须是用户原话）。
在问题、目标用户、现有替代方案、时机都有对应的已记录事实之前，不要起草任何 PRD 章节，也不要调用 update_prd_section。关键问题都已有事实、剩下的可以带假设时，主动提议现在成版，把剩余缺口记为假设或待验证问题。
当你认为已经问清楚了，调用 advance_stage 工具，参数 next="draft"。`
	case StageDraft:
		body = `你是 InsideOut 的 PRD 教练，正在「起草」阶段。基于已记录的事实（见下方清单），通过多次调用 update_prd_section 工具依次写完全部 8 个章节（background, users, goals, nonGoals, stories, requirements, constraints, risks）。
每个章节里的每一句话都必须能追溯到一个已记录的事实，或者明确标注为 [ASSUMPTION]（先用 mark_assumption 记录）或 OPEN QUESTION:。禁止写没有依据的、听起来合理但用户没说过的内容。
requirements 章节要写清楚要解决的问题，而不是预设的技术方案——参考 Cagan 的"what vs how"原则。
写 update_prd_section 时，把这句话依据的事实 id 通过 section_facts 参数传回去。
每次工具调用只写一个章节，内容为简洁的 Markdown。写完全部 8 个章节后，调用 advance_stage 工具，参数 next="critique"。
在此之前，用简短的文字告诉用户你正在写哪个章节。`
	case StageCritique:
		body = `你是 InsideOut 的 PRD 教练，正在「批判」阶段。下方"独立评审员发现的问题"是另一个模型在没看过起草过程、只看 PRD 本身的情况下给出的意见。
	进入本阶段的第一轮回复必须是一份按严重程度从高到低排列的编号列表，一条一行，格式固定为：
	1. [blocking] <章节键> — <问题>。建议：<具体建议>
	方括号里只能是 blocking、major、minor 之一；章节键用 background、users、goals、nonGoals、stories、requirements、constraints、risks。没有具体建议时省略「建议：」及其后的内容。不要在列表前后加别的话。
	用户随后逐条处理 blocking 和 major：要么用 update_prd_section 按建议修改（新增依据先 record_fact），要么选择"保持不变"；无论哪种，处理完这一条后都要调用 resolve_finding（已修改的用 status="resolved"，保持不变的用 status="overridden"）。minor 可以成批处理。
	所有 blocking 问题处理完后，调用 advance_stage 工具，参数 next="finalize"。`
	case StageFinalize:
		body = `你是 InsideOut 的 PRD 教练，正在「定稿」阶段。给出一份固定格式的完整度报告：逐节标注空/单薄/扎实，统计 [ASSUMPTION] 数量，把所有未解决的问题逐条列出。报告最后必须以一行结束：如果没有未解决的问题，写"NO OPEN QUESTIONS"；否则把问题逐条列出。这是本次对话的最后阶段，不需要再调用 advance_stage。`
	default:
		body = `你是 InsideOut 的 PRD 教练。回答简洁、结构化，不编造信息。`
	}

	findings := findingsText
	if findings == "" {
		findings = "(无)"
	}
	// The title and sections are human-written and the ledger and findings are
	// derived from them, so all four ride in data fences; the gaps come from
	// static readiness text and stay outside.
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n当前 PRD 标题：%s\n当前各章节内容：\n%s\n当前读者缺口（按读者分组，含优先级与原因；「之后再验证」的缺口不阻塞成版）：\n%s\n已记录的事实清单：\n%s\n独立评审员发现的问题：\n%s",
		body, toolCallDiscipline, factDiscipline, recommendationDiscipline, dataFenceClause, fenceData(prdTitle), fenceData(formatSectionsForPrompt(sections)), formatGapsForPrompt(sections), fenceData(ledgerText), fenceData(findings))
}

// formatFindingsForPrompt renders open critic findings for injection —
// only open ones matter to the relay stage; resolved/overridden findings
// are already handled.
func formatFindingsForPrompt(findings []CriticFinding) string {
	var open []CriticFinding
	for _, f := range findings {
		if f.Status == FindingOpen {
			open = append(open, f)
		}
	}
	if len(open) == 0 {
		return ""
	}
	var b strings.Builder
	for _, f := range open {
		fmt.Fprintf(&b, "- [%s/%s] id=%s 章节=%s：%s", f.Severity, f.Kind, f.ID, f.Section, f.Issue)
		if f.Quote != "" {
			fmt.Fprintf(&b, "（原文：%s）", f.Quote)
		}
		if f.Suggestion != "" {
			fmt.Fprintf(&b, " 建议：%s", f.Suggestion)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// formatGapsForPrompt renders the per-audience readiness gaps with
// priorities and reasons (PRODUCT.md principle 7: explain the
// question) so the Coach can weave them into conversation.
func formatGapsForPrompt(sections map[string]string) string {
	all := readiness.Assess(sections)
	var b strings.Builder
	for _, audience := range []string{"decision", "management", "delivery", "validation"} {
		ar := all[audience]
		blocking := 0
		for _, g := range ar.Gaps {
			if g.Priority != readiness.ValidateLate {
				blocking++
			}
		}
		fmt.Fprintf(&b, "- 读者 %s：", audience)
		if blocking == 0 {
			b.WriteString("无阻塞性缺口（其余为之后再验证）\n")
			continue
		}
		b.WriteString("\n")
		for _, g := range ar.Gaps {
			fmt.Fprintf(&b, "  - [%s] %s：%s\n", g.Priority, g.Section, g.Reason)
		}
	}
	return strings.TrimRight(b.String(), "\n")
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

func formatSectionsForPrompt(sections map[string]string) string {
	var out string
	for _, key := range store.PrdSectionKeys {
		content := sections[key]
		if content == "" {
			content = "(空)"
		}
		out += fmt.Sprintf("- %s: %s\n", key, content)
	}
	return out
}
