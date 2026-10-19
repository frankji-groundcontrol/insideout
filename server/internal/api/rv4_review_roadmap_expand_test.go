package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// Whole-change review 4, roadmap expand and build confirmation, through the
// whole handler chain on real PostgreSQL (rv4-FR-F198, F122, F120). The cast is
// routeWorld's: the owner (workspace admin and project owner), a plain member,
// and an outsider.

// rvExpandCount is how many roadmap nodes the project holds, read as the owner.
func rvExpandCount(t *testing.T, w *routeWorld) int {
	t.Helper()
	code, body := w.call(w.owner, http.MethodGet, "/api/v1/projects/"+w.project.ID.String()+"/roadmap", "")
	expect(t, "list roadmap", code, body, http.StatusOK)
	return len(decodeAs[[]roadmapNodeView](t, body))
}

// POST /roadmap/{nid}/expand/confirm writes nodes into the plan, so it takes the
// rule every other roadmap write takes (BUG-020): the project owner or a
// workspace admin. A plain member is refused with that rule in the message, and
// nothing is written.
func TestRvExpandConfirmRefusesAPlainMember(t *testing.T) {
	w := newRouteWorld(t)
	parent := w.roadmapNode(t, "Parent")
	confirm := "/api/v1/roadmap/" + parent + "/expand/confirm"
	body := `{"children":[{"title":"a reader's subtask"}]}`

	// The sibling write is the yardstick: a plain member may not add a node.
	code, out := w.call(w.member, http.MethodPost, "/api/v1/projects/"+w.project.ID.String()+"/roadmap", `{"title":"a reader's node","parentId":"`+parent+`"}`)
	expect(t, "member adds a node", code, out, http.StatusForbidden)

	code, out = w.call(w.member, http.MethodPost, confirm, body)
	expect(t, "member confirms an expansion", code, out, http.StatusForbidden)
	if !strings.Contains(out, "project owner or a workspace admin") {
		t.Errorf("the refusal reads %s, want the plan-writer rule", out)
	}
	if got := rvExpandCount(t, w); got != 1 {
		t.Fatalf("the roadmap holds %d nodes after the refused confirm, want only the parent", got)
	}

	code, out = w.call(w.outsider, http.MethodPost, confirm, body)
	expect(t, "outsider confirms an expansion", code, out, http.StatusNotFound)
	code, out = w.call(w.owner, http.MethodPost, confirm, body)
	expect(t, "owner confirms an expansion", code, out, http.StatusCreated)
}

// rv4-FR-F122: the confirm handler was the one node-writing handler without the
// caps COMP-01 put on the others: a title over 200 characters, a description
// over 2000, and a batch over the 40 subtasks the planner itself returns at
// most. The limits count characters, not bytes, so 200 three-byte characters
// still pass. A refused request writes nothing.
func TestRvExpandConfirmCapsWhatItWrites(t *testing.T) {
	w := newRouteWorld(t)
	parent := w.roadmapNode(t, "Parent")
	confirm := "/api/v1/roadmap/" + parent + "/expand/confirm"
	one := func(title, description string) string {
		return jsonBody(t, map[string]any{"children": []map[string]any{{"title": title, "description": description}}})
	}
	many := func(n int) string {
		kids := make([]map[string]any, n)
		for i := range kids {
			kids[i] = map[string]any{"title": fmt.Sprintf("subtask %d", i)}
		}
		return jsonBody(t, map[string]any{"children": kids})
	}

	for _, tc := range []struct{ name, body, msg string }{
		{"a title of 201 characters", one(cjk(201), ""), "max 200 characters"},
		{"a description of 2001 characters", one("subtask", cjk(2001)), "max 2000 characters"},
		{"41 subtasks", many(41), "max 40"},
	} {
		if code, out := w.call(w.owner, http.MethodPost, confirm, tc.body); code != http.StatusBadRequest || !strings.Contains(out, tc.msg) {
			t.Errorf("%s: status %d body %.120s, want 400 naming %q", tc.name, code, out, tc.msg)
		}
	}
	if got := rvExpandCount(t, w); got != 1 {
		t.Fatalf("the refused confirms left %d nodes in the roadmap, want only the parent", got)
	}

	code, out := w.call(w.owner, http.MethodPost, confirm, one(cjk(200), cjk(2000)))
	expect(t, "a title of 200 and a description of 2000 characters", code, out, http.StatusCreated)
	code, out = w.call(w.owner, http.MethodPost, confirm, many(40))
	expect(t, "40 subtasks", code, out, http.StatusCreated)
}
