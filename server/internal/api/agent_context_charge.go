package api

import (
	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/agentcontext"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// briefInCharge points a briefing with no focus at what its human is in
// charge of (2026-11-22, people-in-charge mode): one root becomes the focus,
// several are named in InCharge, none leaves the whole project.
func briefInCharge(in *agentcontext.Inputs, nodes []store.RoadmapNode, actor uuid.UUID, planWriter bool) {
	roots := inChargeRoots(nodes, actor, planWriter)
	if len(roots) == 1 {
		in.FocusNodeID = roots[0]
		return
	}
	in.InCharge = roots
}

// inChargeRoots are the topmost of a project's nodes the actor is in charge
// of, in list order: the nodes they own, and an unowned root when they are a
// plan writer (the project owner or a workspace admin, owner decision
// 2026-11-22).
//
// shortcut: computed in Go over the listed nodes, apart from the store's
// `mine` read (ListMyRoadmapRoots); unify the two if the rule changes again.
func inChargeRoots(nodes []store.RoadmapNode, actor uuid.UUID, planWriter bool) []string {
	parent := make(map[uuid.UUID]*uuid.UUID, len(nodes))
	charged := map[uuid.UUID]bool{}
	for _, n := range nodes {
		parent[n.ID] = n.ParentID
		owns := n.OwnerID != nil && *n.OwnerID == actor
		if owns || (planWriter && n.ParentID == nil && n.OwnerID == nil) {
			charged[n.ID] = true
		}
	}
	var out []string
	for _, n := range nodes {
		if !charged[n.ID] {
			continue
		}
		top := true
		for p := n.ParentID; p != nil && top; p = parent[*p] {
			top = !charged[*p]
		}
		if top {
			out = append(out, n.ID.String())
		}
	}
	return out
}
