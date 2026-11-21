package api

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestMemberListIsForMembersAndNamesEveryRole(t *testing.T) {
	w := newCoverageWorld(t)
	for _, as := range []routeUser{w.owner, w.member} {
		list := decodeAs[[]memberView](t, w.must(t, as, "GET", w.ws()+"/members", "", http.StatusOK))
		roles := map[string]string{}
		for _, m := range list {
			roles[m.UserID] = m.Role
		}
		if len(list) != 2 || roles[w.owner.ID.String()] != "admin" || roles[w.member.ID.String()] != "member" {
			t.Fatalf("members = %+v, want the owner as admin and the member as member", list)
		}
	}
	w.must(t, w.outsider, "GET", w.ws()+"/members", "", http.StatusNotFound)
	w.must(t, w.member, "GET", "/api/v1/workspaces/"+uuid.NewString()+"/members", "", http.StatusNotFound)
	w.must(t, w.member, "GET", "/api/v1/workspaces/not-a-uuid/members", "", http.StatusBadRequest)
}

// Roles are changed by an admin alone, and the change takes effect at once.
func TestMemberRolesAreChangedByAnAdminAlone(t *testing.T) {
	w := newCoverageWorld(t)
	other := w.newMember(t, "other")
	path := func(u routeUser) string { return w.ws() + "/members/" + u.ID.String() }
	role := func(as routeUser, of routeUser) string {
		t.Helper()
		for _, m := range decodeAs[[]memberView](t, w.must(t, as, "GET", w.ws()+"/members", "", http.StatusOK)) {
			if m.UserID == of.ID.String() {
				return m.Role
			}
		}
		return ""
	}
	set := func(as routeUser, of routeUser, payload string, want int) {
		t.Helper()
		w.must(t, as, "PATCH", path(of), payload, want)
	}

	set(w.member, w.member, `{"role":"admin"}`, http.StatusForbidden) // no promoting yourself
	set(other, w.member, `{"role":"admin"}`, http.StatusForbidden)
	set(w.outsider, w.member, `{"role":"admin"}`, http.StatusForbidden)
	for name, payload := range map[string]string{
		"a role that does not exist": `{"role":"owner"}`,
		"no role":                    `{}`,
		"not json":                   `role=admin`,
		"unknown field":              `{"role":"admin","note":"x"}`,
	} {
		t.Run(name, func(t *testing.T) { set(w.owner, w.member, payload, http.StatusBadRequest) })
	}
	w.must(t, w.owner, "PATCH", w.ws()+"/members/not-a-uuid", `{"role":"admin"}`, http.StatusBadRequest)
	w.must(t, w.owner, "PATCH", "/api/v1/workspaces/not-a-uuid/members/"+w.member.ID.String(), `{"role":"admin"}`, http.StatusBadRequest)
	set(w.owner, w.outsider, `{"role":"member"}`, http.StatusNotFound) // not in this workspace
	if got := role(w.owner, w.member); got != "member" {
		t.Fatalf("refused changes left the member as %q, want member", got)
	}

	// A promoted member holds the admin's powers at once: deleting a project is
	// an admin's alone.
	w.must(t, w.member, "DELETE", w.proj(), "", http.StatusForbidden)
	set(w.owner, w.member, `{"role":"admin"}`, http.StatusOK)
	if got := role(w.owner, w.member); got != "admin" {
		t.Fatalf("member's role = %q after the promotion, want admin", got)
	}
	set(w.member, other, `{"role":"admin"}`, http.StatusOK) // and may promote in turn
	set(w.owner, w.member, `{"role":"member"}`, http.StatusOK)
	w.must(t, w.member, "DELETE", w.proj(), "", http.StatusForbidden)
	w.must(t, w.owner, "GET", w.proj(), "", http.StatusOK)
}

func TestMembersAreRemovedByAnAdminOrLeaveOnTheirOwn(t *testing.T) {
	w := newCoverageWorld(t)
	other := w.newMember(t, "other")
	path := func(u routeUser) string { return w.ws() + "/members/" + u.ID.String() }
	count := func() int {
		t.Helper()
		return len(decodeAs[[]memberView](t, w.must(t, w.owner, "GET", w.ws()+"/members", "", http.StatusOK)))
	}

	w.must(t, w.member, "DELETE", path(other), "", http.StatusForbidden) // not another member's to remove
	w.must(t, w.outsider, "DELETE", path(other), "", http.StatusForbidden)
	w.must(t, w.outsider, "DELETE", path(w.outsider), "", http.StatusNotFound) // nothing of theirs to leave
	w.must(t, w.owner, "DELETE", path(w.outsider), "", http.StatusNotFound)
	w.must(t, w.owner, "DELETE", w.ws()+"/members/not-a-uuid", "", http.StatusBadRequest)
	w.must(t, w.owner, "DELETE", "/api/v1/workspaces/not-a-uuid/members/"+other.ID.String(), "", http.StatusBadRequest)
	if n := count(); n != 3 {
		t.Fatalf("%d members after the refused removals, want 3", n)
	}

	// Leaving takes the workspace and everything in it out of sight.
	w.must(t, w.member, "GET", w.proj(), "", http.StatusOK)
	w.must(t, w.member, "DELETE", path(w.member), "", http.StatusOK)
	w.must(t, w.member, "DELETE", path(w.member), "", http.StatusNotFound)
	for _, hidden := range []string{w.ws(), w.ws() + "/members", w.ws() + "/projects", w.proj()} {
		w.must(t, w.member, "GET", hidden, "", http.StatusNotFound)
	}
	if listed := decodeAs[[]workspaceView](t, w.must(t, w.member, "GET", "/api/v1/workspaces", "", http.StatusOK)); len(listed) != 0 {
		t.Fatalf("workspaces listed for a member who left = %+v, want none", listed)
	}

	w.must(t, w.owner, "DELETE", path(other), "", http.StatusOK) // an admin removes anyone
	w.must(t, other, "GET", w.ws(), "", http.StatusNotFound)
	if n := count(); n != 1 {
		t.Fatalf("%d members at the end, want only the owner", n)
	}
}
