package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

func parentPtr(s string) *string { return &s }

// countNodes recursively counts a plan tree (root inclusive).
func countNodes(n *store.RoadmapPlanNode) int {
	if n == nil {
		return 0
	}
	c := 1
	for i := range n.Children {
		c += countNodes(&n.Children[i])
	}
	return c
}

// findNode returns the first node with the given title, depth-first.
func findNode(n *store.RoadmapPlanNode, title string) *store.RoadmapPlanNode {
	if n == nil {
		return nil
	}
	if n.Title == title {
		return n
	}
	for i := range n.Children {
		if got := findNode(&n.Children[i], title); got != nil {
			return got
		}
	}
	return nil
}

// F2: assembleTree must preserve EVERY node's subtree. The old implementation
// copied each child by value while ranging over the node map in random order; a
// copy taken before the child's own children were attached froze a partial
// Children slice and silently dropped whole subtrees (~97% of nodes on a
// branched outline). Because map iteration order is randomized per-run, we
// assemble the same outline many times and demand the full node count every
// time — with the old bug this fails on nearly every iteration.
func TestAssembleTree_PreservesSubtrees(t *testing.T) {
	nodes := []flatNode{
		{ID: "r", Parent: nil, Title: "MVP"},
		{ID: "a", Parent: parentPtr("r"), Title: "Core"},
		{ID: "b", Parent: parentPtr("r"), Title: "Launch"},
		{ID: "a1", Parent: parentPtr("a"), Title: "Auth"},
		{ID: "a2", Parent: parentPtr("a"), Title: "Storage"},
		{ID: "b1", Parent: parentPtr("b"), Title: "Landing page"},
	}
	const want = 6

	for i := 0; i < 100; i++ {
		root, err := assembleTree(nodes, "MVP")
		if err != nil {
			t.Fatalf("iter %d: assembleTree: %v", i, err)
		}
		if got := countNodes(root); got != want {
			t.Fatalf("iter %d: subtree lost — want %d nodes, got %d", i, want, got)
		}
		// The deepest leaf must survive, attached under its real parent.
		a := findNode(root, "Core")
		if a == nil {
			t.Fatalf("iter %d: branch 'Core' missing", i)
		}
		if len(a.Children) != 2 {
			t.Fatalf("iter %d: 'Core' lost children — want 2, got %d", i, len(a.Children))
		}
		if findNode(root, "Storage") == nil || findNode(root, "Landing page") == nil {
			t.Fatalf("iter %d: a grandchild leaf was dropped", i)
		}
	}
}

// A malformed cycle (a node listing itself, or a two-node loop) must not hang
// or duplicate nodes — the built-set breaks re-entry. Such nodes, if not
// reachable from a real root, are simply excluded rather than looping forever.
func TestAssembleTree_CycleSafe(t *testing.T) {
	nodes := []flatNode{
		{ID: "r", Parent: nil, Title: "Root"},
		{ID: "x", Parent: parentPtr("y"), Title: "X"},
		{ID: "y", Parent: parentPtr("x"), Title: "Y"}, // x↔y loop, no root
	}
	root, err := assembleTree(nodes, "Root")
	if err != nil {
		t.Fatalf("assembleTree: %v", err)
	}
	if root.Title != "Root" {
		t.Fatalf("want the real root, got %q", root.Title)
	}
	// The unreachable cycle must not appear (and must not have looped).
	if findNode(root, "X") != nil || findNode(root, "Y") != nil {
		t.Fatalf("unreachable cycle leaked into the tree")
	}
}

// garbageRoadmapStreamer answers the forced tool call with arguments that
// are not JSON, so PlanMVP cannot assemble a tree from them.
type garbageRoadmapStreamer struct{}

func (garbageRoadmapStreamer) StreamChat(context.Context, string, []Message, []Tool, func(string)) (Turn, error) {
	return Turn{}, nil
}

func (garbageRoadmapStreamer) StreamChatForcingTool(context.Context, string, []Message, Tool, func(string)) (Turn, error) {
	return Turn{ToolCall: &ToolCallRequest{Name: submitRoadmapTool.Name, Arguments: "not json {"}}, nil
}

func (garbageRoadmapStreamer) Model() string { return "stub" }

// An unparseable model response must surface as an error. Falling back to
// the template planner would discard the PRD sections and hand back the
// generic three-branch tree (核心功能 / 打磨与验证 / 发布).
func TestPlanMVP_UnparseableModelResponseErrors(t *testing.T) {
	p := NewAnthropicRoadmapPlanner(garbageRoadmapStreamer{})
	sections := map[string]string{"goals": "a specific section the template must not swallow"}
	root, err := p.PlanMVP(context.Background(), localeZH, "Specific PRD", sections)
	if err == nil {
		t.Fatalf("unparseable model response returned a plan, want an error; got %+v", root)
	}
	if root != nil {
		titles := []string{root.Title}
		for _, c := range root.Children {
			titles = append(titles, c.Title)
		}
		joined := strings.Join(titles, ",")
		if strings.Contains(joined, "核心功能") && strings.Contains(joined, "打磨与验证") && strings.Contains(joined, "发布") {
			t.Fatalf("unparseable response fell back to the generic three-branch template: %s", joined)
		}
		t.Fatalf("unparseable response returned a tree alongside the error: %+v", root)
	}
}

// The offline planner must build its branches from the PRD section titles
// it was given, not the fixed 核心功能 / 打磨与验证 / 发布 tree.
func TestTemplatePlanMVP_BranchesFromSectionTitles(t *testing.T) {
	p := NewTemplateRoadmapPlanner()
	sections := map[string]string{
		"goals":        "ship a measurable first version",
		"requirements": "one saved path from idea to result",
	}
	root, err := p.PlanMVP(context.Background(), localeZH, "Workshop tracker", sections)
	if err != nil {
		t.Fatalf("PlanMVP: %v", err)
	}
	if root == nil || root.Title != "Workshop tracker" {
		t.Fatalf("root = %+v, want title Workshop tracker", root)
	}
	for _, key := range []string{"goals", "requirements"} {
		found := false
		for i := range root.Children {
			if strings.HasPrefix(root.Children[i].Title, key+" — ") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("section %q missing from template tree %+v", key, root)
		}
	}
}
