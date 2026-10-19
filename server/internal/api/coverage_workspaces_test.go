package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestWorkspaceIsCreatedListedAndReadByItsMembers(t *testing.T) {
	w := newCoverageWorld(t)
	for name, payload := range map[string]string{
		"blank title":    `{"title":"  "}`,
		"title too long": `{"title":"` + strings.Repeat("x", 201) + `"}`,
		"not json":       `title=T`,
	} {
		t.Run(name, func(t *testing.T) {
			w.must(t, w.outsider, "POST", "/api/v1/workspaces", payload, http.StatusBadRequest)
		})
	}
	if body := w.must(t, w.outsider, "GET", "/api/v1/workspaces", "", http.StatusOK); strings.TrimSpace(body) != "[]" {
		t.Fatalf("workspaces of someone who made none = %s, want an empty list", body)
	}

	made := decodeAs[workspaceView](t, w.must(t, w.outsider, "POST", "/api/v1/workspaces", `{"title":"  Their own  ","description":"d"}`, http.StatusCreated))
	if made.Title != "Their own" || made.MyRole != "admin" || made.MemberCount != 1 || made.Code == "" {
		t.Fatalf("created workspace = %+v, want a trimmed title, its creator as the only admin, and a join code", made)
	}
	listed := decodeAs[[]workspaceView](t, w.must(t, w.outsider, "GET", "/api/v1/workspaces", "", http.StatusOK))
	if len(listed) != 1 || listed[0].ID != made.ID {
		t.Fatalf("their workspaces = %+v, want only the one they made", listed)
	}

	// The world's workspace has two people in it, seen from each side.
	for wantRole, as := range map[string]routeUser{"admin": w.owner, "member": w.member} {
		got := decodeAs[workspaceView](t, w.must(t, as, "GET", w.ws(), "", http.StatusOK))
		if got.MyRole != wantRole || got.MemberCount != 2 || got.Title != "Route tests" {
			t.Fatalf("workspace as %s = %+v, want that role, 2 members", wantRole, got)
		}
	}
	w.must(t, w.outsider, "GET", w.ws(), "", http.StatusNotFound)
	w.must(t, w.owner, "GET", "/api/v1/workspaces/"+uuid.NewString(), "", http.StatusNotFound)
	w.must(t, w.owner, "GET", "/api/v1/workspaces/not-a-uuid", "", http.StatusBadRequest)
}

func TestWorkspaceIsJoinedWithItsCodeOnce(t *testing.T) {
	w := newCoverageWorld(t)
	code := decodeAs[workspaceView](t, w.must(t, w.owner, "GET", w.ws(), "", http.StatusOK)).Code
	join := func(as routeUser, payload string, want int) string {
		t.Helper()
		return w.must(t, as, "POST", "/api/v1/workspaces/join", payload, want)
	}

	join(w.outsider, `{"code":""}`, http.StatusBadRequest)
	join(w.outsider, `{"code":"   "}`, http.StatusBadRequest)
	join(w.outsider, `{"code":`, http.StatusBadRequest)
	join(w.outsider, `{"code":"`+code+`","admin":true}`, http.StatusBadRequest)
	join(w.outsider, `{"code":"no-such-code-`+uuid.NewString()[:8]+`"}`, http.StatusNotFound)
	join(w.member, `{"code":"`+code+`"}`, http.StatusConflict) // already in
	w.must(t, w.outsider, "GET", w.ws(), "", http.StatusNotFound)

	got := decodeAs[workspaceView](t, join(w.outsider, `{"code":"  `+code+`  "}`, http.StatusOK))
	if got.ID != w.project.WorkspaceID.String() || got.MyRole != "member" || got.MemberCount != 3 {
		t.Fatalf("joined workspace = %+v, want the world's, as a plain member, 3 members", got)
	}
	join(w.outsider, `{"code":"`+code+`"}`, http.StatusConflict)
	w.must(t, w.outsider, "GET", w.proj(), "", http.StatusOK) // the workspace's projects are theirs to read now
	w.must(t, w.outsider, "DELETE", w.proj(), "", http.StatusForbidden)
}

func TestWorkspaceEditsAreForAdminsAndDeletionForItsCreatorAlone(t *testing.T) {
	w := newCoverageWorld(t)
	edit := func(as routeUser, payload string, want int) string {
		t.Helper()
		return w.must(t, as, "PATCH", w.ws(), payload, want)
	}
	title := func() string {
		t.Helper()
		return decodeAs[workspaceView](t, w.must(t, w.owner, "GET", w.ws(), "", http.StatusOK)).Title
	}

	edit(w.member, `{"title":"Taken over"}`, http.StatusForbidden)
	edit(w.outsider, `{"title":"Taken over"}`, http.StatusNotFound)
	for name, payload := range map[string]string{
		"blank title":      `{"title":" "}`,
		"title too long":   `{"title":"` + strings.Repeat("x", 201) + `"}`,
		"http cover":       `{"title":"T","coverUrl":"http://example.com/c.png"}`,
		"script cover":     `{"title":"T","coverUrl":"javascript:alert(1)"}`,
		"cover too long":   `{"title":"T","coverUrl":"https://example.com/` + strings.Repeat("a", 2048) + `"}`,
		"not json":         `title=T`,
		"unknown property": `{"title":"T","code":"mine"}`,
	} {
		t.Run(name, func(t *testing.T) { edit(w.owner, payload, http.StatusBadRequest) })
	}
	if got := title(); got != "Route tests" {
		t.Fatalf("refused edits renamed the workspace to %q", got)
	}
	got := decodeAs[workspaceView](t, edit(w.owner, `{"title":"  Renamed  ","description":"d","coverUrl":"https://example.com/c.png"}`, http.StatusOK))
	if got.Title != "Renamed" || got.Description != "d" || got.CoverURL == nil || *got.CoverURL != "https://example.com/c.png" {
		t.Fatalf("edited workspace = %+v, want the new title, description and cover", got)
	}
	edit(w.owner, `{"title":"Renamed"}`, http.StatusOK) // a null cover clears it
	if got := decodeAs[workspaceView](t, w.must(t, w.member, "GET", w.ws(), "", http.StatusOK)); got.CoverURL != nil {
		t.Fatalf("cover after an edit without one = %v, want it cleared", *got.CoverURL)
	}

	// An admin made by promotion may edit but not delete: deleting is the creator's.
	w.must(t, w.owner, "PATCH", w.ws()+"/members/"+w.member.ID.String(), `{"role":"admin"}`, http.StatusOK)
	edit(w.member, `{"title":"By a promoted admin"}`, http.StatusOK)
	w.must(t, w.member, "DELETE", w.ws(), "", http.StatusForbidden)
	w.must(t, w.outsider, "DELETE", w.ws(), "", http.StatusForbidden)
	w.must(t, w.owner, "DELETE", "/api/v1/workspaces/"+uuid.NewString(), "", http.StatusForbidden)
	w.must(t, w.owner, "DELETE", "/api/v1/workspaces/not-a-uuid", "", http.StatusBadRequest)
	if got := title(); got != "By a promoted admin" {
		t.Fatalf("workspace is %q after the refused deletes, want it standing", got)
	}

	w.must(t, w.owner, "DELETE", w.ws(), "", http.StatusOK)
	for _, as := range []routeUser{w.owner, w.member} {
		w.must(t, as, "GET", w.ws(), "", http.StatusNotFound)
		w.must(t, as, "GET", w.proj(), "", http.StatusNotFound) // its projects went with it
	}
}
