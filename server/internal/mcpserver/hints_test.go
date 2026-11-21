package mcpserver

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

// rv5-PARA-04: while a login is held, tools/list hides auth_login_code_verify,
// auth_magic_link_consume and auth_logout, so no error may send the model to
// them. A refused in-process login and a refused identity swap both say to
// restart the process instead.
func TestHintsNameNoToolTheListHides(t *testing.T) {
	pinConfigHome(t)
	t.Cleanup(func() { setSessionToken("") })
	expired := fakeJWT("", "expired", time.Now().Add(-time.Hour), 15*time.Minute)
	api, _ := routedAPI(t, func(w http.ResponseWriter, r *http.Request, _ int) bool {
		if r.URL.Path == "/agent/token" || r.URL.Path == "/me" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return true
		}
		return false
	})
	setSessionToken(expired)
	s := Build(api.URL)
	offered := offeredTools(t, s)

	_, err := client(api.URL).Whoami()
	if err == nil {
		t.Fatal("whoami on a refused login succeeded")
	}
	swap := rvMcpText(t, callTool(t, s, 1, "auth_login_code_verify", `{"email":"a@example.com","code":"123456"}`))
	for _, text := range []string{err.Error(), swap} {
		for _, hidden := range sessionSwapTools {
			if !slices.Contains(offered, hidden) && strings.Contains(text, hidden) {
				t.Errorf("%q names %s, which tools/list does not offer while a login is held", text, hidden)
			}
		}
		if !strings.Contains(text, "restart insideout-mcp") {
			t.Errorf("%q does not say to restart insideout-mcp", text)
		}
	}
}
