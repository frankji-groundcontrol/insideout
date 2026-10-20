package mcpserver

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

// sessionSwapToolNames are advertised only before this process holds a login.
// That is how a human signs in. Once a login is held they leave the offered
// list; a call still has to refuse, and a human restarts the process to
// change identity.
var sessionSwapToolNames = []string{
	"auth_login_code_verify", "auth_magic_link_consume", "auth_logout",
}

func assertSessionSwapListing(t *testing.T, names []string, held bool) {
	t.Helper()
	for _, tool := range sessionSwapToolNames {
		listed := slices.Contains(names, tool)
		if held && listed {
			t.Errorf("after a login, %s is still advertised", tool)
		}
		if !held && !listed {
			t.Errorf("before any login, %s is missing", tool)
		}
	}
	for _, tool := range []string{"auth_login_code_request", "auth_magic_link_request"} {
		if !slices.Contains(names, tool) {
			t.Errorf("%s left the list", tool)
		}
	}
}

// TestSessionSwapToolsLeaveTheListAfterLogin is option A. Before a login the
// three swap tools are listed. After a successful sign-in they are not, and
// calling one anyway still refuses.
func TestSessionSwapToolsLeaveTheListAfterLogin(t *testing.T) {
	cases := []struct {
		name string
		path string
		tool string
		args string
	}{
		{"login code", "/auth/login-code/verify", "auth_login_code_verify", `{"email":"a@example.com","code":"123456"}`},
		{"magic link", "/auth/magic-link/consume", "auth_magic_link_consume", `{"token":"link-token"}`},
		{"password", "/auth/login", "auth_login", `{"email":"a@example.com","password":"pw123456"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pinConfigHome(t)
			t.Cleanup(func() { setSessionToken("") })
			human := humanJWT()
			grant := agentGrant("a1", time.Now(), 15*time.Minute)
			mint := mintsAs(grant)
			api, _ := routedAPI(t, func(w http.ResponseWriter, r *http.Request, nth int) bool {
				if r.URL.Path == tc.path {
					fmt.Fprintf(w, `{"email":"a@example.com","username":"A","accessToken":%q,"refreshToken":"ref-1"}`, human)
					return true
				}
				return mint(w, r, nth)
			})
			s := Build(api.URL)
			assertSessionSwapListing(t, offeredTools(t, s), false)

			if r := callTool(t, s, 1, tc.tool, tc.args); r.Result.IsError {
				t.Fatalf("%s: %s", tc.tool, rvMcpText(t, r))
			}
			assertSessionSwapListing(t, offeredTools(t, s), true)

			again := callTool(t, s, 2, "auth_login_code_verify", `{"email":"a@example.com","code":"999999"}`)
			if !again.Result.IsError || !strings.Contains(rvMcpText(t, again), "a login is already held") {
				t.Fatalf("verify while a login is held: %s", rvMcpText(t, again))
			}
			out := callTool(t, s, 3, "auth_logout", `{}`)
			if !out.Result.IsError || !strings.Contains(rvMcpText(t, out), "confirm") {
				t.Fatalf("auth_logout without confirm: %s", rvMcpText(t, out))
			}
		})
	}
}
