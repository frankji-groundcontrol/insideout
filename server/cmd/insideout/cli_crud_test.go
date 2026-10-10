package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// `insideout join <code>` dispatches through runClientCommand into the
// CRUD verbs and POSTs the workspace code.
func TestJoinCommand(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.Write([]byte(`{"id":"ws1"}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("INSIDEOUT_API", srv.URL)
	t.Setenv("INSIDEOUT_TOKEN", "tok")

	handled, code := runClientCommand([]string{"join", "ABC123"})
	if !handled || code != 0 {
		t.Fatalf("join = (handled=%v, code=%d), want (true, 0)", handled, code)
	}
	if gotMethod != http.MethodPost || gotPath != "/workspaces/join" {
		t.Errorf("got %s %s, want POST /workspaces/join", gotMethod, gotPath)
	}
	if gotBody != `{"code":"ABC123"}` {
		t.Errorf("body = %q, want %q", gotBody, `{"code":"ABC123"}`)
	}
}

// AIDEV-10: `idea delete` told the user a dropped idea "keeps" showing in the
// list with status "dropped"; the list has hidden dropped ideas since USER-09.
func TestIdeaDeleteSaysDroppedIdeasLeaveTheList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("INSIDEOUT_API", srv.URL)
	t.Setenv("INSIDEOUT_TOKEN", "tok")

	var handled bool
	var code int
	out := captureStdout(t, func() { handled, code = runClientCommand([]string{"idea", "delete", "i1"}) })
	if !handled || code != 0 {
		t.Fatalf("idea delete = (handled=%v, code=%d), want (true, 0)", handled, code)
	}
	if !strings.HasPrefix(out, "dropped") || strings.Contains(out, "keeps it") {
		t.Errorf("idea delete printed %q, want a dropped notice that does not claim the idea stays listed", out)
	}
}
