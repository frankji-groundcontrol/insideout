package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// The admin verbs against a live HTTP API: invite revoke is a subcommand
// of invite (invite revoke <id>), not a flag.
func TestInviteRevokeSubcommand(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)
	writeClientConfig(t, clientconfig.Config{})
	t.Setenv("INSIDEOUT_TOKEN", "tok")

	handled, code := runAdminCommand("invite", []string{"--api", srv.URL, "revoke", "inv-1"})
	if !handled || code != 0 {
		t.Fatalf("invite revoke = (handled=%v, code=%d), want (true, 0)", handled, code)
	}
	if gotMethod != http.MethodPost || gotPath != "/admin/invitations/inv-1/revoke" {
		t.Errorf("got %s %s, want POST /admin/invitations/inv-1/revoke", gotMethod, gotPath)
	}
}

// invite without the revoke subcommand keeps minting invitations.
func TestInviteCreateUnchanged(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.Write([]byte(`{"code":"c"}`))
	}))
	t.Cleanup(srv.Close)
	writeClientConfig(t, clientconfig.Config{})
	t.Setenv("INSIDEOUT_TOKEN", "tok")

	handled, code := runAdminCommand("invite", []string{"--api", srv.URL, "a@b.c"})
	if !handled || code != 0 {
		t.Fatalf("invite = (handled=%v, code=%d), want (true, 0)", handled, code)
	}
	if gotPath != "/admin/invitations" || gotBody != `{"email":"a@b.c"}` {
		t.Errorf("got %s body %q, want POST /admin/invitations", gotPath, gotBody)
	}
}
