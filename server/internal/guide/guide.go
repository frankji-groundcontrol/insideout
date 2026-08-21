// Package guide generates the repo-side matching guide (insideout.yaml)
// that maps GitHub activity onto roadmap nodes. One format, produced by
// the API (GET /projects/{id}/guide), the CLI (`insideout guide`), and
// the MCP tool of the same name — the parity rule.
package guide

import (
	"fmt"
	"sort"
	"strings"
)

// Node is the generator's input: a roadmap node reduced to what the
// guide records. Leaf is precomputed (has no children) because evidence
// only ever attaches to leaves (PRODUCT.md).
type Node struct {
	ID    string
	Title string
	Leaf  bool
}

// Generate renders insideout.yaml for a project's roadmap. Output is
// deterministic: nodes in the given order (the API's list order), and
// only leaves carry an editable matchers block — non-leaves are emitted
// commented-out so the file still documents the tree shape.
func Generate(projectTitle string, nodes []Node) string {
	var b strings.Builder
	b.WriteString(`# InsideOut matching guide — maps GitHub activity in this repo onto
# roadmap nodes of ` + quote(projectTitle) + `.
#
# An event (push, pull request, deployment) attaches evidence to every
# node it matches that is a LEAF. Matchers:
#   branches: exact branch name or prefix ending in /*  (event's branch)
#   labels:   exact pull-request label (e.g. roadmap/mvp)
#   paths:    prefix of any file the event touched   (e.g. server/)
# Unmatched events stay visible in InsideOut but attach to nothing.
# Re-generate any time with ` + "`insideout guide <project-id>`" + `.
version: 1
nodes:
`)
	if len(nodes) == 0 {
		b.WriteString("  {} # no roadmap nodes yet — build one first\n")
		return b.String()
	}
	for _, n := range nodes {
		id := indent(n.ID)
		if n.Leaf {
			fmt.Fprintf(&b, "  %s:\n    title: %s\n    branches: [] # e.g. [feature/x, main]\n    labels:   [] # e.g. [roadmap/mvp]\n    paths:    [] # e.g. [\"server/\"]\n", id, quote(n.Title))
			continue
		}
		fmt.Fprintf(&b, "  # %s is a branch node — matchers apply to leaves only:\n  # %s:\n  #   title: %s\n", id, id, quote(n.Title))
	}
	return b.String()
}

// indent keeps ids (already plain uuids today) future-proof if ids ever
// contain YAML-hostile characters.
func indent(s string) string {
	if needsQuote(s) {
		return `"` + escape(s) + `"`
	}
	return s
}

func quote(s string) string {
	return `"` + escape(s) + `"`
}

func escape(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return r.Replace(s)
}

func needsQuote(s string) bool {
	if s == "" {
		return true
	}
	for _, c := range `:#{}[],&*!|>'"%@` + "\t\n" {
		if strings.ContainsRune(s, c) {
			return true
		}
	}
	return false
}

// SortByTitle is a convenience for callers that want a stable,
// human-browsable guide regardless of tree order.
func SortByTitle(nodes []Node) {
	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].Title < nodes[j].Title })
}
