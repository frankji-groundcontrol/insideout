package roadmaptime

import "sort"

// The Canvas's one level (docs/architecture/roadmap-tree.md, "Steps"): which
// nodes are the steps under a root, and their order by what they wait on. It
// gives the answer client/lib/features/project/roadmap_step_flow.dart gives.

// StepNode is the reduced node shape Steps needs. Nodes come in list order:
// siblings by position, ties by creation.
type StepNode struct {
	ID, ParentID string // ParentID "" for a root
	Position     int
	DependsOn    []string
}

// Step is one step of a level in order. Loop is the loop group of steps that
// wait on each other through their work (nil: none). After and Links are
// the steps of other groups it waits on, in step order, with their link
// counts; WaitsOutside the ids its work waits on under no shown step.
type Step struct {
	ID           string
	Stage        int
	Loop         *int
	After        []string
	Links        map[string]int
	ItemsInside  int
	WaitsOutside []string
}

// StepIDs picks the level under root: its children by position, or itself
// when it has none. With root "" a project with exactly one root opens that
// root; otherwise the roots are the steps. It answers the root it opened.
func StepIDs(nodes []StepNode, root string) (string, []string) {
	if root == "" {
		roots := childrenOf(nodes, "")
		if len(roots) != 1 {
			return "", roots
		}
		root = roots[0]
	}
	if kids := childrenOf(nodes, root); len(kids) > 0 {
		return root, kids
	}
	return root, []string{root}
}

func childrenOf(nodes []StepNode, parent string) []string {
	var kids []StepNode
	for _, n := range nodes {
		if n.ParentID == parent {
			kids = append(kids, n)
		}
	}
	sort.SliceStable(kids, func(i, j int) bool { return kids[i].Position < kids[j].Position })
	var out []string
	for _, k := range kids {
		out = append(out, k.ID)
	}
	return out
}

// Steps orders stepIDs. Every dependency in nodes is lifted to the steps
// holding its two ends; a link inside one step, or up or down one branch,
// drops. Steps that reach each other form a loop group; a group's stage is
// the longest path to it over the groups; the order is stage, then group (by
// its first position), then position.
func Steps(nodes []StepNode, stepIDs []string) []Step {
	lv := lift(nodes, stepIDs)
	groups := strongGroups(stepIDs, lv.edges)
	groupOf := map[string]int{}
	for g, members := range groups {
		for _, id := range members {
			groupOf[id] = g
		}
	}
	stage := stages(len(groups), lv.edges, groupOf)
	first := map[string]int{} // a step's place by position
	for i, id := range stepIDs {
		first[id] = i
	}
	order := make([]string, len(stepIDs))
	copy(order, stepIDs)
	sort.SliceStable(order, func(i, j int) bool {
		gi, gj := groupOf[order[i]], groupOf[order[j]]
		if stage[gi] != stage[gj] {
			return stage[gi] < stage[gj]
		}
		return first[groups[gi][0]] < first[groups[gj][0]]
	})
	return assembleSteps(order, groups, groupOf, stage, lv)
}

// assembleSteps builds each step in order, numbering loop groups as they
// first appear.
func assembleSteps(order []string, groups [][]string, groupOf map[string]int, stage []int, lv lifted) []Step {
	rank := map[string]int{}
	for i, id := range order {
		rank[id] = i
	}
	loopIDs := map[int]int{}
	out := make([]Step, 0, len(order))
	for _, id := range order {
		g := groupOf[id]
		s := Step{ID: id, Stage: stage[g], After: []string{}, Links: map[string]int{}, ItemsInside: lv.inside[id], WaitsOutside: lv.waits[id]}
		if s.WaitsOutside == nil {
			s.WaitsOutside = []string{}
		}
		if len(groups[g]) > 1 {
			if _, ok := loopIDs[g]; !ok {
				loopIDs[g] = len(loopIDs)
			}
			loop := loopIDs[g]
			s.Loop = &loop
		}
		for _, e := range lv.edges {
			if e.to == id && groupOf[e.from] != g {
				s.After = append(s.After, e.from)
				s.Links[e.from] = lv.counts[e]
			}
		}
		sort.SliceStable(s.After, func(i, j int) bool { return rank[s.After[i]] < rank[s.After[j]] })
		out = append(out, s)
	}
	return out
}

type edge struct{ from, to string }

// lifted is the level's lifted links (first seen first), their counts, each
// step's descendant count and the outside ids its work waits on.
type lifted struct {
	edges  []edge
	counts map[edge]int
	inside map[string]int
	waits  map[string][]string
}

func lift(nodes []StepNode, stepIDs []string) lifted {
	parent := map[string]string{}
	for _, n := range nodes {
		parent[n.ID] = n.ParentID
	}
	isStep := map[string]bool{}
	for _, id := range stepIDs {
		isStep[id] = true
	}
	// up walks id and its ancestors; seen stops a malformed cycle.
	up := func(id string, visit func(string) bool) {
		for seen := map[string]bool{}; id != "" && !seen[id]; id = parent[id] {
			seen[id] = true
			if visit(id) {
				return
			}
		}
	}
	stepOf := func(id string) (step string) {
		up(id, func(a string) bool {
			if isStep[a] {
				step = a
			}
			return step != ""
		})
		return step
	}
	above := func(id, anc string) (found bool) {
		up(id, func(a string) bool { found = a == anc; return found })
		return found
	}
	lv := lifted{counts: map[edge]int{}, inside: map[string]int{}, waits: map[string][]string{}}
	for _, n := range nodes {
		to := stepOf(n.ID)
		if to == "" {
			continue
		}
		if to != n.ID {
			lv.inside[to]++
		}
		for _, d := range n.DependsOn {
			if above(n.ID, d) || above(d, n.ID) {
				continue // up or down one branch: the tree already orders the two
			}
			from := stepOf(d)
			switch {
			case from == "":
				if !contains(lv.waits[to], d) {
					lv.waits[to] = append(lv.waits[to], d)
				}
			case from != to:
				e := edge{from, to}
				if lv.counts[e] == 0 {
					lv.edges = append(lv.edges, e)
				}
				lv.counts[e]++
			}
		}
	}
	return lv
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// strongGroups is Tarjan's strongly connected components of ids under edges:
// steps that reach each other share a group, members by position.
func strongGroups(ids []string, edges []edge) [][]string {
	next := map[string][]string{}
	for _, e := range edges {
		next[e.from] = append(next[e.from], e.to)
	}
	index, low, onStack := map[string]int{}, map[string]int{}, map[string]bool{}
	var stack []string
	var groups [][]string
	var visit func(v string)
	visit = func(v string) {
		index[v], low[v] = len(index), len(index)
		stack, onStack[v] = append(stack, v), true
		for _, w := range next[v] {
			if _, seen := index[w]; !seen {
				visit(w)
				low[v] = min(low[v], low[w])
			} else if onStack[w] {
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] != index[v] {
			return
		}
		var group []string
		for w := ""; w != v; {
			w, stack = stack[len(stack)-1], stack[:len(stack)-1]
			onStack[w] = false
			group = append(group, w)
		}
		groups = append(groups, group)
	}
	for _, id := range ids {
		if _, seen := index[id]; !seen {
			visit(id)
		}
	}
	rank := map[string]int{}
	for i, id := range ids {
		rank[id] = i
	}
	for _, g := range groups {
		sort.Slice(g, func(i, j int) bool { return rank[g[i]] < rank[g[j]] })
	}
	return groups
}

// stages is each group's stage: one past the furthest group it waits on. The
// links between groups are acyclic, so the walk ends.
func stages(count int, edges []edge, groupOf map[string]int) []int {
	into := map[int][]int{}
	for _, e := range edges {
		if f, t := groupOf[e.from], groupOf[e.to]; f != t {
			into[t] = append(into[t], f)
		}
	}
	stage := make([]int, count)
	done := make([]bool, count)
	var place func(g int) int
	place = func(g int) int {
		if !done[g] {
			for _, f := range into[g] {
				stage[g] = max(stage[g], place(f)+1)
			}
			done[g] = true
		}
		return stage[g]
	}
	for g := range stage {
		place(g)
	}
	return stage
}
