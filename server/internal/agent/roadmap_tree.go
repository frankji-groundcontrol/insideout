package agent

import (
	"fmt"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

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

	// Register every node first (keeping the model's order), then link and
	// build. The old path copied each child *by value* while ranging over the
	// node map in random order; a copy taken before the child's own children
	// were attached froze a partial Children slice and silently dropped whole
	// subtrees. Building top-down from the registered nodes by id guarantees
	// every subtree is complete before it's attached, and iterating `order`
	// (not the map) keeps sibling order deterministic.
	byID := make(map[string]*store.RoadmapPlanNode, len(nodes))
	parent := make(map[string]*string, len(nodes))
	order := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if n.ID == "" || n.Title == "" {
			continue
		}
		if _, dup := byID[n.ID]; dup {
			continue // keep the first occurrence of a repeated id
		}
		byID[n.ID] = &store.RoadmapPlanNode{Title: n.Title, Description: n.Description}
		parent[n.ID] = n.Parent
		order = append(order, n.ID)
	}
	if len(byID) == 0 {
		return nil, fmt.Errorf("agent: no usable nodes")
	}

	children := make(map[string][]string, len(byID))
	var roots []string
	for _, id := range order {
		p := parent[id]
		if p == nil || *p == "" || byID[*p] == nil {
			roots = append(roots, id)
			continue
		}
		children[*p] = append(children[*p], id)
	}

	// build reconstructs a node and its whole subtree by id. The built-set
	// breaks parent cycles (a cycle never re-enters an in-progress node) and
	// keeps a node reachable from two parents attached only once.
	built := make(map[string]bool, len(byID))
	var build func(id string) store.RoadmapPlanNode
	build = func(id string) store.RoadmapPlanNode {
		built[id] = true
		n := *byID[id]
		for _, cid := range children[id] {
			if built[cid] {
				continue
			}
			n.Children = append(n.Children, build(cid))
		}
		return n
	}

	if len(roots) == 1 {
		r := build(roots[0])
		return &r, nil
	}
	// Multiple roots: fold them under one synthesized root so the project has
	// a single MVP goal at the top.
	root := &store.RoadmapPlanNode{Title: rootTitle}
	for _, id := range roots {
		if built[id] {
			continue
		}
		root.Children = append(root.Children, build(id))
	}
	return root, nil
}
