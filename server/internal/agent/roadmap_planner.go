package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// RoadmapPlanner turns a PRD (or a single roadmap node) into a branched
// roadmap tree — the "build the MVP" bridge from idea to reality. Two
// implementations: Anthropic (real LLM via a forced tool call, the same
// schema-validated trick the critic uses) and a deterministic template used
// when no provider is configured — and as a fallback whenever the model's
// output can't be parsed, so the feature never hard-fails on LLM flakiness.
type RoadmapPlanner interface {
	PlanMVP(ctx context.Context, prdTitle string, sections map[string]string) (*store.RoadmapPlanNode, error)
	ExpandNode(ctx context.Context, projectTitle, nodeTitle, nodeDesc string) ([]store.RoadmapPlanNode, error)
}

// --- shared payload shapes (flat list is far easier to validate than a
// recursive schema, and the model handles "outline with parent ids" well). ---

type flatNode struct {
	ID          string  `json:"id"`
	Parent      *string `json:"parent"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
}

var submitRoadmapTool = Tool{
	Name:        "submit_roadmap",
	Description: "Return the MVP roadmap as a flat list of nodes: one root, branches under it (parallel workstreams), tasks under each branch.",
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"nodes": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":          map[string]any{"type": "string"},
						"parent":      map[string]any{"type": []string{"string", "null"}},
						"title":       map[string]any{"type": "string"},
						"description": map[string]any{"type": "string"},
					},
					"required": []string{"id", "parent", "title"},
				},
			},
		},
		"required": []string{"nodes"},
	},
}

var submitSubtasksTool = Tool{
	Name:        "submit_subtasks",
	Description: "Return the subtasks that break the given roadmap node down into executable steps.",
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"subtasks": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"title":       map[string]any{"type": "string"},
						"description": map[string]any{"type": "string"},
					},
					"required": []string{"title"},
				},
			},
		},
		"required": []string{"subtasks"},
	},
}

// assembleTree builds a single-rooted tree from the model's flat list. It
// tolerates multiple "roots" by folding them under one synthesized root and
// guards against parent cycles. Caps the size so a runaway outline can't
// flood the roadmap.
func assembleTree(nodes []flatNode, rootTitle string) (*store.RoadmapPlanNode, error) {
	const maxNodes = 40
	if len(nodes) == 0 {
		return nil, fmt.Errorf("agent: empty roadmap")
	}
	if len(nodes) > maxNodes {
		nodes = nodes[:maxNodes]
	}

	byID := make(map[string]*store.RoadmapPlanNode, len(nodes))
	parent := make(map[string]*string, len(nodes))
	for _, n := range nodes {
		if n.ID == "" || n.Title == "" {
			continue
		}
		byID[n.ID] = &store.RoadmapPlanNode{Title: n.Title, Description: n.Description}
		parent[n.ID] = n.Parent
	}
	if len(byID) == 0 {
		return nil, fmt.Errorf("agent: no usable nodes")
	}

	var roots []string
	for id := range byID {
		p := parent[id]
		if p == nil || *p == "" || byID[*p] == nil {
			roots = append(roots, id)
			continue
		}
		byID[*p].Children = append(byID[*p].Children, *byID[id])
	}

	if len(roots) == 1 {
		return byID[roots[0]], nil
	}
	// Multiple roots: fold them under one synthesized root so the project has
	// a single MVP goal at the top.
	root := &store.RoadmapPlanNode{Title: rootTitle}
	for _, id := range roots {
		root.Children = append(root.Children, *byID[id])
	}
	return root, nil
}

// --- Anthropic implementation ---

type anthropicRoadmapPlanner struct {
	streamer ChatStreamer
	fallback RoadmapPlanner
}

func NewAnthropicRoadmapPlanner(streamer ChatStreamer) RoadmapPlanner {
	return &anthropicRoadmapPlanner{streamer: streamer, fallback: NewTemplateRoadmapPlanner()}
}

func (p *anthropicRoadmapPlanner) PlanMVP(ctx context.Context, prdTitle string, sections map[string]string) (*store.RoadmapPlanNode, error) {
	system := `你是 InsideOut 的产品落地教练。根据用户给出的 PRD（标题和各章节），把这个产品拆成一棵可执行的 MVP 路线图树。
规则：一个根节点（交付 MVP 的总目标）；根节点下 3-5 个可以并行推进的工作流（例如：核心功能、打磨与验证、发布）；每个工作流下 2-4 个具体任务。每个节点 title 简洁（12 字以内），description 一句话。不要编造 PRD 里没有的重大方向。只通过 submit_roadmap 工具返回。`
	user := "PRD 标题：" + prdTitle + "\n\n各章节内容：\n" + formatSectionsForPrompt(sections)

	turn, err := p.streamer.StreamChatForcingTool(ctx, system, []Message{{Role: RoleUser, Content: user}}, submitRoadmapTool, nil)
	if err != nil || turn.ToolCall == nil {
		return p.fallback.PlanMVP(ctx, prdTitle, sections)
	}
	var payload struct {
		Nodes []flatNode `json:"nodes"`
	}
	if json.Unmarshal([]byte(turn.ToolCall.Arguments), &payload) != nil {
		return p.fallback.PlanMVP(ctx, prdTitle, sections)
	}
	root, err := assembleTree(payload.Nodes, prdTitle)
	if err != nil {
		return p.fallback.PlanMVP(ctx, prdTitle, sections)
	}
	return root, nil
}

func (p *anthropicRoadmapPlanner) ExpandNode(ctx context.Context, projectTitle, nodeTitle, nodeDesc string) ([]store.RoadmapPlanNode, error) {
	system := `你是 InsideOut 的产品落地教练。用户要把路线图里的一个节点进一步拆解成可以直接执行的子任务。
给出 3-6 个子任务：title 简洁（12 字以内）、动词开头、可独立执行；description 一句话说明产出。只通过 submit_subtasks 工具返回。`
	user := fmt.Sprintf("项目：%s\n要拆解的节点：%s\n节点说明：%s", projectTitle, nodeTitle, nodeDesc)

	turn, err := p.streamer.StreamChatForcingTool(ctx, system, []Message{{Role: RoleUser, Content: user}}, submitSubtasksTool, nil)
	if err != nil || turn.ToolCall == nil {
		return p.fallback.ExpandNode(ctx, projectTitle, nodeTitle, nodeDesc)
	}
	var payload struct {
		Subtasks []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"subtasks"`
	}
	if json.Unmarshal([]byte(turn.ToolCall.Arguments), &payload) != nil || len(payload.Subtasks) == 0 {
		return p.fallback.ExpandNode(ctx, projectTitle, nodeTitle, nodeDesc)
	}
	out := make([]store.RoadmapPlanNode, 0, len(payload.Subtasks))
	for _, s := range payload.Subtasks {
		if s.Title != "" {
			out = append(out, store.RoadmapPlanNode{Title: s.Title, Description: s.Description})
		}
	}
	if len(out) == 0 {
		return p.fallback.ExpandNode(ctx, projectTitle, nodeTitle, nodeDesc)
	}
	return out, nil
}

// --- deterministic template (offline / fallback) ---

type templateRoadmapPlanner struct{}

func NewTemplateRoadmapPlanner() RoadmapPlanner { return &templateRoadmapPlanner{} }

func (templateRoadmapPlanner) PlanMVP(_ context.Context, prdTitle string, _ map[string]string) (*store.RoadmapPlanNode, error) {
	return &store.RoadmapPlanNode{
		Title:       prdTitle,
		Description: "交付第一个可用版本",
		Children: []store.RoadmapPlanNode{
			{Title: "核心功能", Children: []store.RoadmapPlanNode{
				{Title: "搭建最小可用版本", Description: "只做最核心的一条用户路径"},
				{Title: "打通关键流程", Description: "从开始到得到结果的完整闭环"},
			}},
			{Title: "打磨与验证", Children: []store.RoadmapPlanNode{
				{Title: "找目标用户试用", Description: "3-5 个真实目标用户"},
				{Title: "收集反馈并迭代", Description: "记录问题，快速修正"},
			}},
			{Title: "发布", Children: []store.RoadmapPlanNode{
				{Title: "准备落地页", Description: "一句话说清价值"},
				{Title: "发布并收集首批用户", Description: "小范围放出，观察数据"},
			}},
		},
	}, nil
}

func (templateRoadmapPlanner) ExpandNode(_ context.Context, _, _, _ string) ([]store.RoadmapPlanNode, error) {
	return []store.RoadmapPlanNode{
		{Title: "明确验收标准", Description: "写清楚怎样算完成"},
		{Title: "拆出第一步并实现", Description: "最小可验证的一步"},
		{Title: "自测并修正", Description: "跑一遍，修掉问题"},
		{Title: "标记完成并同步进展", Description: "更新状态，记录进展"},
	}, nil
}
