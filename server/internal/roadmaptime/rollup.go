package roadmaptime

import "time"

// Rollup is what lies below one node of the tree: its leaves (a leaf counts
// itself), how many are finished (done or validated, as in Progress), how
// many are validated, and the earliest deadline among the unfinished ones,
// the work still racing a date (nil when none has one).
type Rollup struct {
	Leaves, Done, Validated int
	EarliestDeadline        *time.Time
}

// TreeNode is the reduced node shape Rollups needs.
type TreeNode struct {
	ID, ParentID string // ParentID "" for a root
	Status       string
	Deadline     *time.Time
}

// Rollups returns the Rollup of every node in nodes, which must hold each
// node it is asked about together with everything below it. A leaf is a node
// no other node in the list names as its parent, the rule Progress uses.
func Rollups(nodes []TreeNode) map[string]Rollup {
	parent := make(map[string]string, len(nodes))
	hasChild := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		parent[n.ID] = n.ParentID
		if n.ParentID != "" {
			hasChild[n.ParentID] = true
		}
	}
	out := make(map[string]Rollup, len(nodes))
	for _, n := range nodes {
		if hasChild[n.ID] {
			continue
		}
		// Walk from the leaf up through the listed ancestors; seen stops a
		// malformed cycle from looping.
		seen := map[string]bool{}
		for id := n.ID; !seen[id]; id = parent[id] {
			if _, listed := parent[id]; !listed {
				break
			}
			seen[id] = true
			r := out[id]
			r.Leaves++
			if Finished(n.Status) {
				r.Done++
			} else if n.Deadline != nil && (r.EarliestDeadline == nil || n.Deadline.Before(*r.EarliestDeadline)) {
				r.EarliestDeadline = n.Deadline
			}
			if n.Status == "validated" {
				r.Validated++
			}
			out[id] = r
		}
	}
	return out
}
