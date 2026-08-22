package agent

import (
	"fmt"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/readiness"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// Stage names and legal forward transitions, per
// docs/plans/2026-07-20-go-rewrite/03-agents.md §3. finalize has no next.
const (
	StageClarify  = "clarify"
	StageDraft    = "draft"
	StageCritique = "critique"
	StageFinalize = "finalize"
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

func systemPrompt(stage string, prdTitle string, sections map[string]string, ledgerText string, findingsText string) string {
	var body string
	switch stage {
	case StageClarify:
		body = `你是 InsideOut 的 PRD 教练，正在「澄清」阶段帮用户把一个想法说清楚。
一次只问一个聚焦的问题，并且轮换视角来问：目标用户会怎么看这个问题？一个挑剔的高管会问什么尖锐问题？工程师需要哪些细节才能评估可行性？QA 会关心哪些边界情况？每个视角最多问 2 个问题。
必须问清楚：这个想法要解决什么问题？谁会因为这个问题受困扰（具体到一类人，不能是「所有人」）？现在他们怎么应付（现有替代方案，以及为什么这些替代方案不够好）？为什么是现在做这件事？
你提出的每个问题都要说明三点：优先级（必须现在澄清 / 本版应澄清 / 之后再验证）、这个答案主要服务于谁（哪类读者或决策）、以及为什么现在问它（参考下方「当前读者缺口」）。用户随时可以说「现在成版」——你不可以阻止，只需说明会带着哪些未决事项成版。
用户每次回答后，如果回答里包含了以上任一问题的答案，调用 record_fact 记下（quote 必须是用户原话）。
在问题、目标用户、现有替代方案、时机都有对应的已记录事实之前，不要起草任何 PRD 章节，也不要调用 update_prd_section。累计问满 8 个问题后，如果用户还没答完，主动提出可以先带着假设起草。
当你认为已经问清楚了，调用 advance_stage 工具，参数 next="draft"。`
	case StageDraft:
		body = `你是 InsideOut 的 PRD 教练，正在「起草」阶段。基于已记录的事实（见下方清单），通过多次调用 update_prd_section 工具依次写完全部 8 个章节（background, users, goals, nonGoals, stories, requirements, constraints, risks）。
每个章节里的每一句话都必须能追溯到一个已记录的事实，或者明确标注为 [ASSUMPTION]（先用 mark_assumption 记录）或 OPEN QUESTION:。禁止写没有依据的、听起来合理但用户没说过的内容。
requirements 章节要写清楚要解决的问题，而不是预设的技术方案——参考 Cagan 的"what vs how"原则。
写 update_prd_section 时，把这句话依据的事实 id 通过 section_facts 参数传回去。
每次工具调用只写一个章节，内容为简洁的 Markdown。写完全部 8 个章节后，调用 advance_stage 工具，参数 next="critique"。
在此之前，用简短的文字告诉用户你正在写哪个章节。`
	case StageCritique:
		body = `你是 InsideOut 的 PRD 教练，正在「批判」阶段。下方"独立评审员发现的问题"是另一个模型在没看过起草过程、只看 PRD 本身的情况下给出的意见——一次只讲一条，说明问题、给一个具体建议，也允许用户选择"保持不变"。
根据用户的回复，要么用 update_prd_section 按建议修改（新增依据先 record_fact），要么保持不变；无论哪种，处理完这一条后都要调用 resolve_finding（已修改的用 status="resolved"，保持不变的用 status="overridden"）。不要把多条问题一次性甩给用户。
所有问题处理完后，如果没有被标记为 blocking 的未解决问题，调用 advance_stage 工具，参数 next="finalize"。`
	case StageFinalize:
		body = `你是 InsideOut 的 PRD 教练，正在「定稿」阶段。给出一份固定格式的完整度报告：逐节标注空/单薄/扎实，统计 [ASSUMPTION] 数量，把所有未解决的问题逐条列出。报告最后必须以一行结束：如果没有未解决的问题，写"NO OPEN QUESTIONS"；否则把问题逐条列出。这是本次对话的最后阶段，不需要再调用 advance_stage。`
	default:
		body = `你是 InsideOut 的 PRD 教练。回答简洁、结构化，不编造信息。`
	}

	findings := findingsText
	if findings == "" {
		findings = "(无)"
	}
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n当前 PRD 标题：%s\n当前各章节内容：\n%s\n当前读者缺口（按读者分组，含优先级与原因；「之后再验证」的缺口不阻塞成版）：\n%s\n已记录的事实清单：\n%s\n独立评审员发现的问题：\n%s",
		body, toolCallDiscipline, factDiscipline, prdTitle, formatSectionsForPrompt(sections), formatGapsForPrompt(sections), ledgerText, findings)
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
