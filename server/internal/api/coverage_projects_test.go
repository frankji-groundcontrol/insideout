package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// projectDetail is GET /projects/{pid}: the project with its newest page of
// updates.
type projectDetail struct {
	projectView
	Updates    []projectUpdateView `json:"updates"`
	NextCursor *string             `json:"nextCursor"`
}

func TestProjectReadsAreForWorkspaceMembersOnly(t *testing.T) {
	w := newCoverageWorld(t)
	for _, as := range []routeUser{w.owner, w.member} {
		board := decodeAs[[]projectView](t, w.must(t, as, "GET", w.ws()+"/projects", "", http.StatusOK))
		if len(board) != 1 || board[0].ID != w.project.ID.String() {
			t.Fatalf("workspace board = %+v, want exactly the world's project", board)
		}
		got := decodeAs[projectDetail](t, w.must(t, as, "GET", w.proj(), "", http.StatusOK))
		if got.ID != w.project.ID.String() || got.Title != "Route tests" || got.Updates == nil || len(got.Updates) != 0 || got.NextCursor != nil {
			t.Fatalf("project detail = %+v, want the project with an empty (not null) update feed", got)
		}
	}

	w.must(t, w.outsider, "GET", w.ws()+"/projects", "", http.StatusNotFound)
	w.must(t, w.outsider, "GET", w.proj(), "", http.StatusNotFound)
	w.must(t, w.owner, "GET", "/api/v1/projects/"+uuid.NewString(), "", http.StatusNotFound)
	w.must(t, w.owner, "GET", "/api/v1/projects/not-a-uuid", "", http.StatusBadRequest)
	w.must(t, w.owner, "GET", "/api/v1/workspaces/not-a-uuid/projects", "", http.StatusBadRequest)
}

// The detail embeds one page of the timeline. A full page is the only hint
// that older history exists, so the cursor appears exactly when the page is
// full and points at the oldest update shown.
func TestProjectDetailEmbedsOnePageOfUpdatesAndPointsAtTheRest(t *testing.T) {
	w := newCoverageWorld(t)
	add := func() {
		t.Helper()
		// Through the store: fifty requests would test the router fifty times
		// and the cursor once.
		if _, err := w.st.AddProjectUpdate(context.Background(), w.owner.ID, w.project.ID, "note", "history"); err != nil {
			t.Fatalf("add an update: %v", err)
		}
	}
	for i := 0; i < store.ProjectUpdatesPageSize-1; i++ {
		add()
	}
	got := decodeAs[projectDetail](t, w.must(t, w.member, "GET", w.proj(), "", http.StatusOK))
	if len(got.Updates) != store.ProjectUpdatesPageSize-1 || got.NextCursor != nil {
		t.Fatalf("%d updates, cursor %v: a page that is not full must not claim there is more", len(got.Updates), got.NextCursor)
	}

	add()
	got = decodeAs[projectDetail](t, w.must(t, w.member, "GET", w.proj(), "", http.StatusOK))
	last := got.Updates[len(got.Updates)-1].ID
	if len(got.Updates) != store.ProjectUpdatesPageSize || got.NextCursor == nil || *got.NextCursor != last {
		t.Fatalf("%d updates, cursor %v: a full page must point at its oldest update %s", len(got.Updates), got.NextCursor, last)
	}
}

func TestProjectsAreCreatedByMembersWithATitle(t *testing.T) {
	w := newCoverageWorld(t)
	made := decodeAs[projectView](t, w.must(t, w.member, "POST", w.ws()+"/projects", `{"title":"  Member's project  ","description":"d"}`, http.StatusCreated))
	if made.Title != "Member's project" || made.OwnerID == nil || *made.OwnerID != w.member.ID.String() || made.WorkspaceID != w.project.WorkspaceID.String() {
		t.Fatalf("created project = %+v, want a trimmed title, the creator as owner, in the workspace", made)
	}
	if n := len(decodeAs[[]projectView](t, w.must(t, w.owner, "GET", w.ws()+"/projects", "", http.StatusOK))); n != 2 {
		t.Fatalf("board has %d projects, want the world's and the new one", n)
	}

	w.must(t, w.outsider, "POST", w.ws()+"/projects", `{"title":"Sneaked in"}`, http.StatusForbidden)
	for name, payload := range map[string]string{
		"blank title":    `{"title":"   "}`,
		"title too long": `{"title":"` + strings.Repeat("x", 201) + `"}`,
		"not json":       `title=T`,
	} {
		t.Run(name, func(t *testing.T) { w.must(t, w.member, "POST", w.ws()+"/projects", payload, http.StatusBadRequest) })
	}
	if n := len(decodeAs[[]projectView](t, w.must(t, w.owner, "GET", w.ws()+"/projects", "", http.StatusOK))); n != 2 {
		t.Fatalf("board has %d projects after the refused creates, want 2", n)
	}
}

// Editing is for the project's owner or a workspace admin; owning a project
// is not administering the workspace, so it never reaches deleting. The edit
// replaces the whole record, owner included: the client always sends ownerId,
// and leaving it out (or sending "") leaves the project unowned.
func TestProjectEditsFollowTheirRoles(t *testing.T) {
	w := newCoverageWorld(t)
	patch := func(as routeUser, payload string, want int) string {
		t.Helper()
		return w.must(t, as, "PATCH", w.proj(), payload, want)
	}
	edit := func(title, status string, owner routeUser) string {
		return fmt.Sprintf(`{"title":%s,"description":"d","status":%q,"ownerId":%q}`, js(title), status, owner.ID.String())
	}
	title := func() string {
		t.Helper()
		return decodeAs[projectDetail](t, w.must(t, w.owner, "GET", w.proj(), "", http.StatusOK)).Title
	}

	patch(w.member, edit("Renamed", "active", w.owner), http.StatusForbidden) // neither its owner nor an admin
	patch(w.outsider, edit("Renamed", "active", w.owner), http.StatusNotFound)
	if got := title(); got != "Route tests" {
		t.Fatalf("a refused edit renamed the project to %q", got)
	}

	for name, payload := range map[string]string{
		"blank title":    `{"title":"  ","status":"active"}`,
		"title too long": `{"title":"` + strings.Repeat("x", 201) + `","status":"active"}`,
		"unknown status": `{"title":"T","status":"finished"}`,
		"bad owner id":   `{"title":"T","status":"active","ownerId":"nobody"}`,
		"not json":       `title=T`,
		"unknown field":  `{"title":"T","status":"active","color":"red"}`,
	} {
		t.Run(name, func(t *testing.T) { patch(w.owner, payload, http.StatusBadRequest) })
	}
	if got := title(); got != "Route tests" {
		t.Fatalf("a rejected edit renamed the project to %q", got)
	}

	got := decodeAs[projectView](t, patch(w.owner, edit("Renamed", "paused", w.owner), http.StatusOK))
	if got.Title != "Renamed" || got.Status != "paused" || got.OwnerID == nil || *got.OwnerID != w.owner.ID.String() {
		t.Fatalf("edited project = %+v, want the new title and status, still the owner's", got)
	}

	// Handed over, the member edits as the owner but is still not an admin; the
	// former owner is an admin, so their edit stands on that alone.
	got = decodeAs[projectView](t, patch(w.owner, edit("Renamed", "active", w.member), http.StatusOK))
	if got.OwnerID == nil || *got.OwnerID != w.member.ID.String() {
		t.Fatalf("owner after the hand-over = %v, want the member", got.OwnerID)
	}
	patch(w.member, edit("By the new owner", "active", w.member), http.StatusOK)
	patch(w.owner, edit("By the admin", "active", w.member), http.StatusOK)
	w.must(t, w.member, "DELETE", w.proj(), "", http.StatusForbidden)
	if got := title(); got != "By the admin" {
		t.Fatalf("title = %q, want the admin's edit, and the project still standing", got)
	}

	// Leaving ownerId out unowns the project; from then on only an admin edits.
	got = decodeAs[projectView](t, patch(w.owner, `{"title":"Ownerless","status":"active"}`, http.StatusOK))
	if got.OwnerID != nil {
		t.Fatalf("owner after an edit that omitted ownerId = %v, want none", *got.OwnerID)
	}
	patch(w.member, edit("Taken back", "active", w.member), http.StatusForbidden)
	got = decodeAs[projectView](t, patch(w.owner, `{"title":"Still ownerless","status":"active","ownerId":""}`, http.StatusOK))
	if got.OwnerID != nil || got.Title != "Still ownerless" {
		t.Fatalf("after an admin's edit with ownerId \"\" = %+v, want it ownerless", got)
	}
}

// KNOWN DEFECT, found by rv4-TEST-03 and not fixed here (the fix is a WITH CHECK
// on the projects_update policy or a refusal in store.UpdateProject, neither in
// this cluster's files). The policy has a USING clause and no WITH CHECK, so
// Postgres judges the NEW row by it too: an owner who is not a workspace admin
// cannot give the project away or leave it ownerless. The store lets the edit
// through, the database refuses the new row (SQLSTATE 42501) and the handler
// answers 500, where its own rule says the owner may edit. This pins today's
// behaviour so that whoever fixes it (the call succeeding, or a 403) has to
// revisit this test: the answer is 500 and the project is as it was.
func TestProjectOwnerWhoIsNotAnAdminCannotGiveTheProjectAway_KnownDefect(t *testing.T) {
	w := newCoverageWorld(t)
	other := w.newMember(t, "other")
	owned := func(by routeUser) string {
		return fmt.Sprintf(`{"title":"Handed down","status":"active","ownerId":%q}`, by.ID.String())
	}
	w.must(t, w.owner, "PATCH", w.proj(), owned(w.member), http.StatusOK) // an admin may; the member now owns it

	for name, payload := range map[string]string{
		"to another member": owned(other),
		"to nobody":         `{"title":"Handed down","status":"active"}`,
	} {
		t.Run(name, func(t *testing.T) {
			status, body := w.call(w.member, "PATCH", w.proj(), payload)
			if status != http.StatusInternalServerError {
				t.Fatalf("the hand-over answered %d %s: the known defect (500) has changed, update this pin, the changelog and the issue", status, body)
			}
			got := decodeAs[projectView](t, w.must(t, w.owner, "GET", w.proj(), "", http.StatusOK))
			if got.OwnerID == nil || *got.OwnerID != w.member.ID.String() {
				t.Fatalf("a refused hand-over left the owner as %v, want it unchanged", got.OwnerID)
			}
		})
	}
}

func TestProjectDeleteIsForWorkspaceAdminsOnly(t *testing.T) {
	w := newCoverageWorld(t)
	w.must(t, w.member, "DELETE", w.proj(), "", http.StatusForbidden)
	w.must(t, w.outsider, "DELETE", w.proj(), "", http.StatusNotFound)
	w.must(t, w.owner, "DELETE", "/api/v1/projects/"+uuid.NewString(), "", http.StatusNotFound)
	w.must(t, w.owner, "DELETE", "/api/v1/projects/not-a-uuid", "", http.StatusBadRequest)
	w.must(t, w.owner, "GET", w.proj(), "", http.StatusOK) // every refused call left it standing

	w.must(t, w.owner, "DELETE", w.proj(), "", http.StatusOK)
	w.must(t, w.owner, "GET", w.proj(), "", http.StatusNotFound)
	w.must(t, w.owner, "DELETE", w.proj(), "", http.StatusNotFound)
	if board := decodeAs[[]projectView](t, w.must(t, w.owner, "GET", w.ws()+"/projects", "", http.StatusOK)); len(board) != 0 {
		t.Fatalf("board after the delete = %+v, want it empty", board)
	}
}
