package main

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// rv5-PARA-06: an update corrected and removed, the profile edited, a member's
// role changed and the member removed, and the workspace deleted, from the CLI
// on the real API. DATABASE_URL-gated.
func TestCLIPeopleVerbsOnTheRealAPI(t *testing.T) {
	w := newContractWorld(t)
	c := apiclient.New(w.base)
	if err := c.Login(w.email, w.password); err != nil {
		t.Fatalf("login: %v", err)
	}
	writeClientConfig(t, clientconfig.Config{APIBase: w.base, Token: c.Token(), RefreshToken: c.RefreshToken()})

	sfx := uuid.NewString()[:8]
	ws := cliJSON[doc](t, "workspaces", "create", "--title", "People "+sfx)
	pr := cliJSON[doc](t, "projects", "create", "--title", "Board "+sfx, ws.ID)

	note := cliJSON[doc](t, "update", "add", "--kind", "note", "--content", "first draft", pr.ID)
	if got := cliJSON[doc](t, "update", "edit", "--content", "second draft", note.ID); got.Content != "second draft" {
		t.Fatalf("update edit = %+v, want the new text", got)
	}
	cliOut(t, "update", "delete", note.ID)
	if page := cliJSON[updatesPage](t, "updates", pr.ID); slices.ContainsFunc(page.Updates, func(u doc) bool { return u.ID == note.ID }) {
		t.Fatalf("updates after update delete = %+v, want the note gone", page)
	}

	type profile struct{ Username, Bio string }
	cliOut(t, "profile", "update", "--bio", "Builds boards")
	if got := cliJSON[profile](t, "profile", "update", "--username", "Renamed "+sfx); got.Username != "Renamed "+sfx || got.Bio != "Builds boards" {
		t.Fatalf("profile update --username = %+v, want the new name with the bio kept", got)
	}
	if got := cliJSON[profile](t, "profile", "update", "--bio", "clear"); got.Bio != "" || got.Username != "Renamed "+sfx {
		t.Fatalf("profile update --bio clear = %+v, want no bio and the name kept", got)
	}

	// A second person joins with the workspace code.
	email, password := "member-"+uuid.NewString()+"@example.org", "pw-for-tests-"+uuid.NewString()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	other, err := w.st.CreateUser(context.Background(), email, hash, "member-user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	joiner := apiclient.New(w.base)
	if err := joiner.Login(email, password); err != nil {
		t.Fatalf("member login: %v", err)
	}
	if _, err := joiner.JoinWorkspace(ws.Code); err != nil {
		t.Fatalf("join: %v", err)
	}
	cliOut(t, "member", "role", "--role", "admin", ws.ID, other.ID.String())
	if out := cliOut(t, "members", ws.ID); !strings.Contains(out, `"admin"`) || strings.Count(out, `"admin"`) != 2 {
		t.Fatalf("members after member role = %s, want two admins", out)
	}
	cliOut(t, "member", "remove", ws.ID, other.ID.String())
	if out := cliOut(t, "members", ws.ID); strings.Contains(out, other.ID.String()) {
		t.Fatalf("members after member remove = %s, want the member gone", out)
	}

	if res := runCLI(t, cliEnv(), "", "workspace", "delete", ws.ID); res.code != 2 {
		t.Fatalf("workspace delete without --yes: exit %d\n%s\nwant a usage error", res.code, res.output())
	}
	cliOut(t, "workspace", "delete", "--yes", ws.ID)
	if res := runCLI(t, cliEnv(), "", "workspace", ws.ID); res.code == 0 {
		t.Fatalf("workspace after delete: exit 0\n%s\nwant the API's not-found error", res.output())
	}
}
