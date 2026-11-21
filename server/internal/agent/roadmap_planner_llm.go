package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/frankji-groundcontrol/insideout/server/internal/llm"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// The zh planner prompts (the en mirrors live in prompts_en.go).
const (
	plannerPlanMVPPromptZH = `你是 InsideOut 的产品落地教练。根据用户给出的 PRD（标题和各章节），把这个产品拆成一棵可执行的 MVP 路线图树。
规则：一个根节点（交付 MVP 的总目标）；根节点下 3-5 个可以并行推进的工作流（例如：核心功能、打磨与验证、发布）；每个工作流下 2-4 个具体任务。每个节点 title 简洁（12 字以内），description 一句话。不要编造 PRD 里没有的重大方向。只通过 submit_roadmap 工具返回。`

	plannerExpandPromptZH = `你是 InsideOut 的产品落地教练。用户要把路线图里的一个节点进一步拆解成可以直接执行的子任务。
给出 3-6 个子任务：title 简洁（12 字以内）、动词开头、可独立执行；description 一句话说明产出。只通过 submit_subtasks 工具返回。`
)

// anthropicRoadmapPlanner plans through the configured model with a forced
// tool call.
type anthropicRoadmapPlanner struct {
	streamer llm.ChatStreamer
	fallback RoadmapPlanner
}

func NewAnthropicRoadmapPlanner(streamer llm.ChatStreamer) RoadmapPlanner {
	return &anthropicRoadmapPlanner{streamer: streamer, fallback: NewTemplateRoadmapPlanner()}
}

func (p *anthropicRoadmapPlanner) PlanMVP(ctx context.Context, locale, prdTitle string, sections map[string]string) (*store.RoadmapPlanNode, error) {
	system := plannerPlanMVPPromptZH + "\n\n" + dataFenceClause
	user := "PRD 标题：" + fenceData(prdTitle) + "\n\n各章节内容：\n" + fenceData(formatSectionsForPrompt(sections))
	if locale == localeEN {
		system = plannerPlanMVPPromptEN + "\n\n" + dataFenceClause
		user = "PRD title: " + fenceData(prdTitle) + "\n\nSection contents:\n" + fenceData(formatSectionsForPrompt(sections))
	}

	turn, err := p.streamer.StreamChatForcingTool(ctx, system, []llm.Message{{Role: llm.RoleUser, Content: user}}, submitRoadmapTool, nil)
	if err != nil {
		return nil, err
	}
	if turn.ToolCall == nil {
		return nil, fmt.Errorf("agent: roadmap planner returned no tool call")
	}
	var payload struct {
		Nodes []flatNode `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(turn.ToolCall.Arguments), &payload); err != nil {
		return nil, fmt.Errorf("agent: unparseable roadmap: %w", err)
	}
	return assembleTree(payload.Nodes, prdTitle)
}

func (p *anthropicRoadmapPlanner) ExpandNode(ctx context.Context, locale, projectTitle, nodeTitle, nodeDesc string) ([]store.RoadmapPlanNode, error) {
	system := plannerExpandPromptZH + "\n\n" + dataFenceClause
	user := fmt.Sprintf("项目：%s\n要拆解的节点：%s\n节点说明：%s", fenceData(projectTitle), fenceData(nodeTitle), fenceData(nodeDesc))
	if locale == localeEN {
		system = plannerExpandPromptEN + "\n\n" + dataFenceClause
		user = fmt.Sprintf("Project: %s\nNode to break down: %s\nNode description: %s", fenceData(projectTitle), fenceData(nodeTitle), fenceData(nodeDesc))
	}

	turn, err := p.streamer.StreamChatForcingTool(ctx, system, []llm.Message{{Role: llm.RoleUser, Content: user}}, submitSubtasksTool, nil)
	if err != nil {
		// Serve the template, but carry the provider error back: a fallback
		// that reports success would keep the caller's circuit breaker
		// healthy through a provider outage.
		draft, _ := p.fallback.ExpandNode(ctx, locale, projectTitle, nodeTitle, nodeDesc)
		return draft, &templateExpandError{err: err, draft: draft}
	}
	if turn.ToolCall == nil {
		return p.fallback.ExpandNode(ctx, locale, projectTitle, nodeTitle, nodeDesc)
	}
	var payload struct {
		Subtasks []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"subtasks"`
	}
	if json.Unmarshal([]byte(turn.ToolCall.Arguments), &payload) != nil || len(payload.Subtasks) == 0 {
		return p.fallback.ExpandNode(ctx, locale, projectTitle, nodeTitle, nodeDesc)
	}
	// Cap the model's output the same way PlanMVP caps a full outline
	// (assembleTree's maxNodes): a runaway expansion can't flood a single node
	// with dozens of subtasks, and an overlong title is trimmed by rune (not
	// byte) so multibyte CJK titles don't split a character — same inline
	// pattern github/commits.go uses for commit subjects (F18).
	if len(payload.Subtasks) > maxSubtasks {
		payload.Subtasks = payload.Subtasks[:maxSubtasks]
	}
	out := make([]store.RoadmapPlanNode, 0, len(payload.Subtasks))
	for _, s := range payload.Subtasks {
		if s.Title == "" {
			continue
		}
		if r := []rune(s.Title); len(r) > maxTitleRunes {
			s.Title = string(r[:maxTitleRunes])
		}
		out = append(out, store.RoadmapPlanNode{Title: s.Title, Description: s.Description})
	}
	if len(out) == 0 {
		return p.fallback.ExpandNode(ctx, locale, projectTitle, nodeTitle, nodeDesc)
	}
	return out, nil
}

// templateExpandError reports that an expand was answered from the built-in
// template because the provider call failed. The draft is still served, but
// the error must reach the caller's gate (Admit's done) so the breaker
// counts the outage instead of a success. The API layer finds it
// structurally, the way it finds AIGate on a Coach.
type templateExpandError struct {
	err   error
	draft []store.RoadmapPlanNode
}

func (e *templateExpandError) Error() string {
	return "agent: expand served from the template: " + e.err.Error()
}

func (e *templateExpandError) Unwrap() error { return e.err }

func (e *templateExpandError) TemplateDraft() []store.RoadmapPlanNode { return e.draft }
