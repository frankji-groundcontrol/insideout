package agent

import (
	"context"
	"sort"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// templateRoadmapPlanner is the deterministic planner, used only offline
// (no provider configured).
type templateRoadmapPlanner struct{}

func NewTemplateRoadmapPlanner() RoadmapPlanner { return &templateRoadmapPlanner{} }

// The template planners ignore the locale: they are the offline fallback
// whose fixed shapes never consult a model, so they answer the same in
// both languages (the LLM prompts are what the locale selects).
func (templateRoadmapPlanner) PlanMVP(_ context.Context, _ string, prdTitle string, sections map[string]string) (*store.RoadmapPlanNode, error) {
	root := &store.RoadmapPlanNode{Title: prdTitle, Description: "交付第一个可用版本"}
	seen := make(map[string]bool, len(sections))
	// Each branch is titled "key — first clause of the body" so an offline
	// plan still carries the PRD's own words, not just its section keys;
	// the full body rides the description. Empty sections are skipped —
	// gaps are readiness's job, not the plan's.
	add := func(key, body string) {
		body = strings.TrimSpace(body)
		if body == "" || seen[key] {
			return
		}
		seen[key] = true
		firstLine := body
		if i := strings.IndexAny(body, "\n。；;"); i > 0 {
			firstLine = body[:i]
		}
		if r := []rune(firstLine); len(r) > 60 {
			firstLine = string(r[:60]) + "…"
		}
		root.Children = append(root.Children, store.RoadmapPlanNode{
			Title: key + " — " + firstLine, Description: body,
		})
	}
	for _, key := range store.PrdSectionKeys {
		add(key, sections[key])
	}
	extra := make([]string, 0, len(sections))
	for key := range sections {
		if !seen[key] {
			extra = append(extra, key)
		}
	}
	sort.Strings(extra)
	for _, key := range extra {
		add(key, sections[key])
	}
	return root, nil
}

func (templateRoadmapPlanner) ExpandNode(_ context.Context, _, _, _, _ string) ([]store.RoadmapPlanNode, error) {
	return []store.RoadmapPlanNode{
		{Title: "明确验收标准", Description: "写清楚怎样算完成"},
		{Title: "拆出第一步并实现", Description: "最小可验证的一步"},
		{Title: "自测并修正", Description: "跑一遍，修掉问题"},
		{Title: "标记完成并同步进展", Description: "更新状态，记录进展"},
	}, nil
}
