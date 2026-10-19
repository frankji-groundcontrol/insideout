package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// rv4 whole-change review, cluster w4-rv-store-db, through the real router on
// real PostgreSQL.

// rvStoreDbAdminCount counts the admins in the workspace roster as caller sees it.
func rvStoreDbAdminCount(t *testing.T, w *routeWorld, caller routeUser) int {
	t.Helper()
	n := 0
	for _, m := range decodeAs[[]memberView](t, w.must(t, caller, "GET", w.ws()+"/members", "", http.StatusOK)) {
		if m.Role == "admin" {
			n++
		}
	}
	return n
}

// rv4-FR-F134, F141, F146: the last admin can neither step down nor leave. The
// store refuses with a conflict; the routes answer 409 and say why instead of
// 200 followed by a workspace nobody can administer.
func TestRvStoreDbLastAdmin_RoutesAnswer409(t *testing.T) {
	w := newCoverageWorld(t)
	ownerPath := w.ws() + "/members/" + w.owner.ID.String()
	memberPath := w.ws() + "/members/" + w.member.ID.String()
	const why = "a workspace needs at least one admin"

	for _, c := range []struct{ name, method, body string }{
		{"stepping down", "PATCH", `{"role":"member"}`},
		{"leaving", "DELETE", ""},
	} {
		got := w.must(t, w.owner, c.method, ownerPath, c.body, http.StatusConflict)
		if !strings.Contains(got, why) {
			t.Errorf("the only admin %s: body %s, want it to say %q", c.name, got, why)
		}
	}
	if n := rvStoreDbAdminCount(t, w, w.owner); n != 1 {
		t.Fatalf("%d admins after the refusals, want 1", n)
	}

	// With a second admin the owner may step down, and the rule then holds for
	// the member who took over.
	w.must(t, w.owner, "PATCH", memberPath, `{"role":"admin"}`, http.StatusOK)
	w.must(t, w.owner, "PATCH", ownerPath, `{"role":"member"}`, http.StatusOK)
	w.must(t, w.member, "PATCH", memberPath, `{"role":"member"}`, http.StatusConflict)
	w.must(t, w.member, "DELETE", memberPath, "", http.StatusConflict)
	w.must(t, w.owner, "DELETE", ownerPath, "", http.StatusOK) // a plain member leaves as before
	if n := rvStoreDbAdminCount(t, w, w.member); n != 1 {
		t.Fatalf("%d admins at the end, want 1", n)
	}
}

// rv4-FR-F02, F08, F15 (server half): a double tap on Accept sends the same
// decision twice. The nodes were already built once; the history note was
// appended every time. The route answers both copies 200 and the timeline
// records one decision, while the opposite decision stays a decision of its own.
func TestRvStoreDbDecision_ADoubleTapAddsOneNote(t *testing.T) {
	w := newRouteWorld(t)
	id := w.proposeItems(t, "Alpha "+uuid.NewString()[:8])
	notes := func() []string {
		t.Helper()
		updates, err := w.st.ListProjectUpdates(context.Background(), w.owner.ID, w.project.ID, 100, nil)
		if err != nil {
			t.Fatalf("list updates: %v", err)
		}
		var out []string
		for _, u := range updates {
			if u.Kind == "note" && strings.HasPrefix(u.Content, "[proposal ") {
				out = append(out, u.Content)
			}
		}
		return out
	}

	for tap := 1; tap <= 2; tap++ {
		status, body := w.decide(w.owner, id, acceptAndApply)
		expect(t, "accept", status, body, http.StatusOK)
	}
	if got := notes(); len(got) != 1 {
		t.Errorf("two taps on Accept left %d decision notes, want 1: %q", len(got), got)
	}
	status, body := w.decide(w.owner, id, `{"decision":"rejected"}`)
	expect(t, "reject", status, body, http.StatusOK)
	if got := notes(); len(got) != 2 {
		t.Errorf("accepting then rejecting left %d decision notes, want 2: %q", len(got), got)
	}
	if d := w.decisionOf(t, id); d != "rejected" {
		t.Errorf("the record says %q after the reversal, want rejected (a decision may be reversed)", d)
	}
}
