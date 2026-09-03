package mcpserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// invite_revoke is the MCP projection of `insideout invite revoke <id>`.
func TestInviteRevokeTool(t *testing.T) {
	var gotMethod, gotPath string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(api.Close)

	res := drive(t, api.URL, []string{
		toolCall(1, "invite_revoke", `{"invitation_id":"inv-1"}`),
	})
	if res[1].Result.IsError {
		t.Fatalf("invite_revoke failed: %+v", res[1])
	}
	if gotMethod != http.MethodPost || gotPath != "/admin/invitations/inv-1/revoke" {
		t.Errorf("got %s %s, want POST /admin/invitations/inv-1/revoke", gotMethod, gotPath)
	}
}

// signup_mode with no mode reads the public /auth/config; with a mode it
// writes /admin/settings — the CLI signup-mode verb's two shapes.
func TestSignupModeTool(t *testing.T) {
	type request struct{ method, path, body string }
	var requests []request
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		requests = append(requests, request{r.Method, r.URL.Path, string(raw)})
		switch r.URL.Path {
		case "/auth/config":
			w.Write([]byte(`{"signupMode":"invitation_only"}`))
		case "/admin/settings":
			w.Write([]byte(`{"signupMode":"open"}`))
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	t.Cleanup(api.Close)

	res := drive(t, api.URL, []string{
		toolCall(1, "signup_mode", `{}`),
		toolCall(2, "signup_mode", `{"mode":"open"}`),
	})
	text := func(r rpcResult) string {
		if len(r.Result.Content) == 0 {
			return ""
		}
		return r.Result.Content[0].Text
	}
	if r := res[1]; r.Result.IsError || !strings.Contains(text(r), "invitation_only") {
		t.Errorf("signup_mode read = %+v, want the current mode", r)
	}
	if r := res[2]; r.Result.IsError || !strings.Contains(text(r), "open") {
		t.Errorf("signup_mode write = %+v, want the API's answer", r)
	}
	want := []request{
		{http.MethodGet, "/auth/config", ""},
		{http.MethodPut, "/admin/settings", `{"signupMode":"open"}`},
	}
	if len(requests) != len(want) {
		t.Fatalf("requests = %+v, want %+v", requests, want)
	}
	for i, w := range want {
		if requests[i] != w {
			t.Errorf("request %d = %+v, want %+v", i, requests[i], w)
		}
	}
}
