package agent

import (
	"context"

	"github.com/frankji-groundcontrol/insideout/server/internal/llm"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// RoadmapPlanner turns a PRD (or a single roadmap node) into a branched
// roadmap tree — the "build the MVP" bridge from idea to reality. Two
// implementations: Anthropic (real LLM via a forced tool call, the same
// schema-validated trick the critic uses) and a deterministic template used
// only when no provider is configured. A configured planner returns the
// model's error instead of substituting that template — the template
// discards the PRD sections and would look like a real plan.
type RoadmapPlanner interface {
	// The locale ('en'/'zh') selects the prompt language — the coaching
	// conversation's stored choice (PRODUCT.md's first-class languages).
	PlanMVP(ctx context.Context, locale, prdTitle string, sections map[string]string) (*store.RoadmapPlanNode, error)
	ExpandNode(ctx context.Context, locale, projectTitle, nodeTitle, nodeDesc string) ([]store.RoadmapPlanNode, error)
}

// --- shared payload shapes (flat list is far easier to validate than a
// recursive schema, and the model handles "outline with parent ids" well). ---

type flatNode struct {
	ID          string  `json:"id"`
	Parent      *string `json:"parent"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
}

var submitRoadmapTool = llm.Tool{
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

var submitSubtasksTool = llm.Tool{
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

// Output bounds for the model's roadmap data. maxSubtasks mirrors
// assembleTree's maxNodes so a node expansion can't outgrow a full plan;
// maxTitleRunes trims overlong titles by rune (F18).
const (
	maxSubtasks   = 40
	maxTitleRunes = 200
)
