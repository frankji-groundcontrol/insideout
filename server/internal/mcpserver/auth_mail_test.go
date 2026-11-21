package mcpserver

import (
	"net/http"
	"strings"
	"testing"
)

// rv5-PARA-05: the emailed sign-in tools pass on the server's mail word, as
// `insideout login --code|--magic` does, so a code that was only logged on a
// server with no mail provider does not read as sent.
func TestEmailedSignInToolsSayWhatTheServerDidWithTheMail(t *testing.T) {
	pinConfigHome(t)
	api, _ := routedAPI(t, func(w http.ResponseWriter, r *http.Request, _ int) bool {
		if r.URL.Path == "/auth/login-code" || r.URL.Path == "/auth/magic-link" {
			w.Write([]byte(`{"ok":true,"mail":"logged"}`))
			return true
		}
		return false
	})
	s := Build(api.URL)
	for i, tool := range []string{"auth_login_code_request", "auth_magic_link_request"} {
		text := rvMcpText(t, callTool(t, s, i+1, tool, `{"email":"a@example.com"}`))
		if !strings.Contains(text, "logged on the server, not delivered") {
			t.Errorf("%s answered %q, want the server's mail status (logged, not delivered)", tool, text)
		}
	}
}
