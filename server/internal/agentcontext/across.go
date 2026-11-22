package agentcontext

import (
	"maps"
	"slices"

	"github.com/frankji-groundcontrol/insideout/server/internal/roadmaptime"
)

// People-in-charge briefing (2026-11-22 plan, S1): the focus says what its
// whole subtree waits on outside it, what outside waits on it, and its own
// steps in order; with no focus, a human in charge of several branches is
// briefed on those branches.

// putAcross adds waitsOn (the unfinished nodes outside the focus that
// unfinished work inside waits on), blocks (the unfinished nodes outside that
// wait on unfinished work inside), as titles, and steps, the focus's steps in
// order, when it has any below it.
func putAcross(ctx map[string]any, n *NodeIn, in Inputs) {
	below := subtreeIDs(in.Nodes, n.ID)
	byID := make(map[string]*NodeIn, len(in.Nodes))
	for i := range in.Nodes {
		byID[in.Nodes[i].ID] = &in.Nodes[i]
	}
	var waits, blocks []string
	for _, x := range in.Nodes {
		if roadmaptime.Finished(x.Status) {
			continue
		}
		for _, id := range x.DependsOn {
			d := byID[id]
			if d == nil || roadmaptime.Finished(d.Status) || below[x.ID] == below[id] {
				continue
			}
			if below[x.ID] && !slices.Contains(waits, d.Title) {
				waits = append(waits, d.Title)
			} else if !below[x.ID] && !slices.Contains(blocks, x.Title) {
				blocks = append(blocks, x.Title)
			}
		}
	}
	if waits != nil {
		ctx["waitsOn"] = waits
	}
	if blocks != nil {
		ctx["blocks"] = blocks
	}
	nodes := make([]roadmaptime.StepNode, len(in.Nodes))
	for i, x := range in.Nodes {
		nodes[i] = roadmaptime.StepNode{ID: x.ID, ParentID: x.ParentID, Position: x.Position, DependsOn: x.DependsOn}
	}
	if _, ids := roadmaptime.StepIDs(nodes, n.ID); ids[0] != n.ID {
		var steps []map[string]any
		for _, s := range roadmaptime.Steps(nodes, ids) {
			steps = append(steps, map[string]any{"id": s.ID, "title": byID[s.ID].Title, "stage": s.Stage})
		}
		ctx["steps"] = steps
	}
}

// inChargeContext names the roots in in.InCharge and keeps the leaves below
// them.
func inChargeContext(in Inputs, leaves []NodeIn) ([]map[string]string, []NodeIn) {
	under := map[string]bool{}
	var roots []map[string]string
	for _, id := range in.InCharge {
		if n := findNode(in.Nodes, id); n != nil {
			roots = append(roots, map[string]string{"id": id, "title": n.Title})
			maps.Copy(under, subtreeIDs(in.Nodes, id))
		}
	}
	return roots, slices.DeleteFunc(leaves, func(l NodeIn) bool { return !under[l.ID] })
}

// acrossFields names the waitsOn, blocks and steps lists the output carries,
// for the untrusted block, which lists only fields that hold text.
func acrossFields(out map[string]any) []string {
	var fields []string
	if f, ok := out["focus"].(map[string]any); ok {
		for key, field := range map[string]string{"waitsOn": "focus.waitsOn[]", "blocks": "focus.blocks[]", "steps": "focus.steps[].title"} {
			if f[key] != nil {
				fields = append(fields, field)
			}
		}
	}
	leaves, _ := out["leaves"].([]map[string]any)
	if slices.ContainsFunc(leaves, func(l map[string]any) bool { return l["waitsOn"] != nil }) {
		fields = append(fields, "leaves[].waitsOn[]")
	}
	if out["inCharge"] != nil {
		fields = append(fields, "inCharge[].title")
	}
	slices.Sort(fields)
	return fields
}
