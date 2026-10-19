package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// rv4-FR-F194: the owner a project is given to has to be a member of its
// workspace. An admin could give a project to a user outside the workspace (a
// 200, and a project owned by someone who cannot even open it) and to a user
// id that does not exist (the foreign key failed: a 500).

func TestRvHandoverANewOwnerMustBeAWorkspaceMember(t *testing.T) {
	for name, pick := range map[string]func(w *routeWorld) string{
		"an existing user outside the workspace": func(w *routeWorld) string { return w.outsider.ID.String() },
		"a user id that does not exist":          func(*routeWorld) string { return uuid.NewString() },
	} {
		t.Run(name, func(t *testing.T) {
			w := newCoverageWorld(t) // a world of its own: the red run lets the first case change the project
			status, body := w.call(w.owner, "PATCH", w.proj(), rvHandoverPatch("Away", "active", rvHandoverOwner(pick(w))))
			if status != http.StatusBadRequest || !strings.Contains(body, "member of this workspace") {
				t.Fatalf("giving the project to %s answered %d %s, want 400 naming the member requirement", name, status, body)
			}
			if owner, title := rvHandoverOwnerOf(t, w); owner == nil || *owner != w.owner.ID.String() || title != "Route tests" {
				t.Fatalf("a refused edit left owner %v, title %q; want the project as it was", owner, title)
			}
		})
	}
}

// Who may change the owner is judged before whom it is given to: a caller who
// may not give the project away is told so (403) whatever they name, and learns
// nothing about who is in the workspace. These pass without the member check.
func TestRvHandoverAnUnpermittedCallerIsRefusedBeforeTheNewOwnerIsJudged(t *testing.T) {
	w := newCoverageWorld(t)
	other := w.newMember(t, "other")
	rvHandoverGiveTo(t, w, w.member)
	named := rvHandoverPatch("Sneaked", "active", rvHandoverOwner(w.outsider.ID.String()))

	w.must(t, w.member, "PATCH", w.proj(), named, http.StatusForbidden) // its owner, not an admin
	w.must(t, other, "PATCH", w.proj(), named, http.StatusForbidden)    // neither its owner nor an admin
	if owner, title := rvHandoverOwnerOf(t, w); owner == nil || *owner != w.member.ID.String() || title != "Handed down" {
		t.Fatalf("a refused edit left owner %v, title %q; want the project as it was", owner, title)
	}
}

// Only a change of owner is judged. A project whose owner has since left the
// workspace stays editable by an admin with the owner the client was shown (an
// edit sends it back unchanged). This passes without the member check.
func TestRvHandoverAnAdminCanStillEditAProjectWhoseOwnerLeftTheWorkspace(t *testing.T) {
	w := newCoverageWorld(t)
	gone := w.newMember(t, "gone")
	rvHandoverGiveTo(t, w, gone)
	if err := w.st.RemoveMember(context.Background(), w.owner.ID, w.project.WorkspaceID, gone.ID); err != nil {
		t.Fatalf("the member leaves the workspace: %v", err)
	}

	got := decodeAs[projectView](t, w.must(t, w.owner, "PATCH", w.proj(), rvHandoverPatch("Renamed", "paused", rvHandoverOwner(gone.ID.String())), http.StatusOK))
	if got.Title != "Renamed" || got.Status != "paused" || got.OwnerID == nil || *got.OwnerID != gone.ID.String() {
		t.Fatalf("the admin's edit = %+v, want it applied with the owner unchanged", got)
	}
}
