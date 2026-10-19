package api

import (
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
