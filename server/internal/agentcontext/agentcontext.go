// Package agentcontext assembles the compact, focus-scoped context an
// agent receives (PRODUCT.md: never the entire graph). Pure: callers
// feed plain inputs from the store; the output shape is the contract
// shared by API, CLI, and MCP.
package agentcontext

// Inputs are the facts an assembly needs, already reduced to plain
// values so this package stays free of store types.
type Inputs struct {
	ProjectTitle string
	ProjectID    string
	Mode         string // brainstorming | implementation | review
	FocusNodeID  string // optional; scopes the assembly
	PrdTitle     string
	PrdSections  map[string]string
	LatestCommit *CommitIn
	Nodes        []NodeIn
	// EvidenceCounts maps node id → evidence row count.
	EvidenceCounts map[string]int
}

// CommitIn is the latest human Commit, for version context.
type CommitIn struct {
	Revision int
	Name     string
	Audience string
}

// NodeIn is a roadmap node reduced to what context needs.
type NodeIn struct {
	ID       string
	ParentID string // empty = root
	Title    string
	Status   string
}

// Assemble returns the mode-shaped context map. Modes select fields:
// brainstorming emphasizes the product argument, implementation the
// focus node/leaves/evidence, review the version baseline.
func Assemble(in Inputs) map[string]any {
	out := map[string]any{
		"projectId":    in.ProjectID,
		"projectTitle": in.ProjectTitle,
		"mode":         in.Mode,
	}
	if in.LatestCommit != nil {
		out["version"] = map[string]any{
			"revision": in.LatestCommit.Revision, "name": in.LatestCommit.Name, "audience": in.LatestCommit.Audience,
		}
	}

	leaves := leafNodes(in.Nodes)
	switch in.Mode {
	case "brainstorming":
		out["productArgument"] = pickSections(in.PrdSections, "background", "users", "nonGoals")
		out["openQuestions"] = blankSections(in.PrdSections)
	case "review":
		out["prdCore"] = pickSections(in.PrdSections, "background", "goals", "nonGoals", "risks", "requirements")
	default: // implementation
		if in.FocusNodeID != "" {
			if n := findNode(in.Nodes, in.FocusNodeID); n != nil {
				out["focus"] = nodeContext(n, in)
			}
		}
		out["leaves"] = leavesContext(leaves, in)
		if in.FocusNodeID == "" && in.PrdTitle != "" {
			out["prdTitle"] = in.PrdTitle
		}
	}
	out["vocabulary"] = map[string]string{
		"checkpoint": "POST /api/v1/agent/checkpoint — record work done; never applies changes",
		"propose":    "POST /api/v1/agent/propose — propose structure/scope/priority; a human decides",
		"version":    "commits are human-only (POST /api/v1/prds/{id}/commit)",
	}
	return out
}

func nodeContext(n *NodeIn, in Inputs) map[string]any {
	ctx := map[string]any{
		"id": n.ID, "title": n.Title, "status": n.Status,
		"evidence": in.EvidenceCounts[n.ID],
	}
	var siblings []string
	for _, other := range in.Nodes {
		if other.ParentID == n.ParentID && other.ID != n.ID {
			siblings = append(siblings, other.Title)
		}
	}
	if siblings != nil {
		ctx["siblings"] = siblings
	}
	return ctx
}

func leavesContext(leaves []NodeIn, in Inputs) []map[string]any {
	var out []map[string]any
	for _, n := range leaves {
		out = append(out, map[string]any{
			"id": n.ID, "title": n.Title, "status": n.Status, "evidence": in.EvidenceCounts[n.ID],
		})
	}
	return out
}

func leafNodes(nodes []NodeIn) []NodeIn {
	parents := map[string]bool{}
	for _, n := range nodes {
		if n.ParentID != "" {
			parents[n.ParentID] = true
		}
	}
	var leaves []NodeIn
	for _, n := range nodes {
		if !parents[n.ID] {
			leaves = append(leaves, n)
		}
	}
	return leaves
}

func findNode(nodes []NodeIn, id string) *NodeIn {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}

func pickSections(sections map[string]string, keys ...string) map[string]string {
	out := map[string]string{}
	for _, k := range keys {
		if v, ok := sections[k]; ok && v != "" {
			out[k] = v
		}
	}
	return out
}

func blankSections(sections map[string]string) []string {
	var out []string
	for k, v := range sections {
		if v == "" {
			out = append(out, k)
		}
	}
	return out
}
