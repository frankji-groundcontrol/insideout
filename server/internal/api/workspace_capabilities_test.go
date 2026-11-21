package api

import (
	"net/http"
	"testing"
)

// Every workspace view tells its viewer what the server lets them do, so a
// client can hide what it would refuse (rv5-FE-05): the creator edits and
// deletes, another admin only edits, a member neither.
func TestWorkspaceViewsCarryTheViewersEditAndDeleteRights(t *testing.T) {
	type rights struct {
		ID        string `json:"id"`
		CanEdit   bool   `json:"canEdit"`
		CanDelete bool   `json:"canDelete"`
	}
	w := newCoverageWorld(t)
	other := w.newMember(t, "other")
	w.must(t, w.owner, "PATCH", w.ws()+"/members/"+other.ID.String(), `{"role":"admin"}`, http.StatusOK)
	wsID := w.project.WorkspaceID.String()

	for name, c := range map[string]struct {
		as               routeUser
		edit, deleteable bool
	}{
		"creator":     {w.owner, true, true},
		"other admin": {other, true, false},
		"member":      {w.member, false, false},
	} {
		t.Run(name, func(t *testing.T) {
			check := func(where string, got rights) {
				t.Helper()
				if got.CanEdit != c.edit || got.CanDelete != c.deleteable {
					t.Fatalf("%s: canEdit=%v canDelete=%v, want %v %v", where, got.CanEdit, got.CanDelete, c.edit, c.deleteable)
				}
			}
			check("GET", decodeAs[rights](t, w.must(t, c.as, "GET", w.ws(), "", http.StatusOK)))
			found := false
			for _, got := range decodeAs[[]rights](t, w.must(t, c.as, "GET", "/api/v1/workspaces", "", http.StatusOK)) {
				if got.ID == wsID {
					found = true
					check("list", got)
				}
			}
			if !found {
				t.Fatal("the workspace is missing from the viewer's list")
			}
			if c.edit {
				check("PATCH", decodeAs[rights](t, w.must(t, c.as, "PATCH", w.ws(), `{"title":"Route tests"}`, http.StatusOK)))
			}
		})
	}

	made := decodeAs[rights](t, w.must(t, w.outsider, "POST", "/api/v1/workspaces", `{"title":"Mine"}`, http.StatusCreated))
	if !made.CanEdit || !made.CanDelete {
		t.Fatalf("created workspace = %+v, want its creator able to edit and delete", made)
	}
	code := decodeAs[workspaceView](t, w.must(t, w.owner, "GET", w.ws(), "", http.StatusOK)).Code
	joined := decodeAs[rights](t, w.must(t, w.outsider, "POST", "/api/v1/workspaces/join", `{"code":"`+code+`"}`, http.StatusOK))
	if joined.CanEdit || joined.CanDelete {
		t.Fatalf("joined workspace = %+v, want a new member able to do neither", joined)
	}
}
