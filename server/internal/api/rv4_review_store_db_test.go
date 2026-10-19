package api

import (
	"net/http"
	"strings"
	"testing"
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
