package mcpserver

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// rv4-AIDEV-04 (docs/architecture/agent-token-scope.md): insideout-mcp held the
// user's own credential and offered all 52 tools on it, Commit and the proposal
// decision included. Now the registry follows the credential (a process that holds
// a login withholds the tools a human must run) and the tools that remain run on
// an agent-scope token traded for that login, never on the login itself.

func TestAProcessHoldingALoginDoesNotOfferTheHumanOnlyTools(t *testing.T) {
	cases := map[string]func(t *testing.T) (api string){
		"INSIDEOUT_TOKEN": func(t *testing.T) string {
			t.Setenv("INSIDEOUT_TOKEN", humanJWT())
			return exampleAPI
		},
		"saved login": func(t *testing.T) string {
			saveConfig(t, clientconfig.Config{APIBase: exampleAPI, Token: humanJWT(), RefreshToken: "ref-1"})
			return ""
		},
		"an agent-scope token": func(t *testing.T) string {
			t.Setenv("INSIDEOUT_TOKEN", fakeJWT("agent", "pre", time.Now(), 15*time.Minute))
			return exampleAPI
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			pinConfigHome(t)
			names := offeredTools(t, Build(setup(t)))
			if got := stillOffered(names); len(got) != 0 {
				t.Errorf("offered to a process holding a login: %v", got)
			}
			for _, want := range agentTools {
				if !slices.Contains(names, want) {
					t.Errorf("the agent's tool %q is missing", want)
				}
			}
		})
	}
}

// Without a login there is nothing to scope: a process that still has to sign in
// or register keeps the whole registry. A token that is not a JWT cannot carry a
// scope (no server accepts one), so it is passed through untouched.
func TestWithoutAJWTLoginTheRegistryAndTheTokenAreUnchanged(t *testing.T) {
	pinConfigHome(t)
	if got := stillOffered(offeredTools(t, Build(exampleAPI))); len(got) != len(humanOnlyTools) {
		t.Errorf("no login: offered %v, want all %v", got, humanOnlyTools)
	}

	t.Setenv("INSIDEOUT_TOKEN", "opaque-token")
	api, seen := routedAPI(t, nil)
	s := Build(api.URL)
	if got := stillOffered(offeredTools(t, s)); len(got) != len(humanOnlyTools) {
		t.Errorf("opaque token: offered %v, want all %v", got, humanOnlyTools)
	}
	if r := callTool(t, s, 1, "whoami", `{}`); r.Result.IsError {
		t.Fatalf("whoami: %+v", r)
	}
	wantSeen(t, seen(), []string{"GET /me Bearer opaque-token"})
}

func TestToolCallsRideAnAgentScopeTokenMintedOnce(t *testing.T) {
	pinConfigHome(t)
	human := humanJWT()
	t.Setenv("INSIDEOUT_TOKEN", human)
	grant := agentGrant("a1", time.Now(), 15*time.Minute)
	api, seen := routedAPI(t, mintsAs(grant))
	s := Build(api.URL)

	for i := 1; i <= 2; i++ {
		if r := callTool(t, s, i, "whoami", `{}`); r.Result.IsError {
			t.Fatalf("whoami %d: %+v", i, r)
		}
	}

	wantSeen(t, seen(human, "<human>", grant.token, "<agent a1>"), []string{
		"POST /agent/token Bearer <human>", "GET /me Bearer <agent a1>", "GET /me Bearer <agent a1>",
	})
}

func TestAnAgentTokenNearExpiryIsMintedAgain(t *testing.T) {
	pinConfigHome(t)
	human := humanJWT()
	t.Setenv("INSIDEOUT_TOKEN", human)
	old := agentGrant("old", time.Now().Add(-14*time.Minute), 15*time.Minute) // 1 minute of 15 left
	fresh := agentGrant("fresh", time.Now(), 15*time.Minute)
	api, seen := routedAPI(t, mintsAs(old, fresh))
	s := Build(api.URL)

	for i := 1; i <= 3; i++ {
		if r := callTool(t, s, i, "whoami", `{}`); r.Result.IsError {
			t.Fatalf("whoami %d: %+v", i, r)
		}
	}

	wantSeen(t, seen(human, "<human>", old.token, "<old>", fresh.token, "<fresh>"), []string{
		"POST /agent/token Bearer <human>", "GET /me Bearer <old>", // the token just minted is used as it is
		"POST /agent/token Bearer <human>", "GET /me Bearer <fresh>", // the next call finds it near expiry
		"GET /me Bearer <fresh>", // and the fresh one is kept
	})
}

// A failed exchange is a tool error. The full login is never the fallback, on
// any route, whatever the reason the server gave.
func TestAFailedExchangeIsAToolErrorAndNeverFallsBackToTheLogin(t *testing.T) {
	human := humanJWT()
	fullBack := fmt.Sprintf(`{"accessToken":%q,"expiresAt":%q,"scope":"agent"}`, human, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"a server older than the route", http.StatusNotFound, `{"error":"not found"}`},
		{"refused", http.StatusForbidden, `{"error":"forbidden"}`},
		{"the login expired", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"a server error", http.StatusInternalServerError, `{"error":"internal error"}`},
		{"not json", http.StatusOK, `<html>hello</html>`},
		{"no token", http.StatusOK, `{"scope":"agent"}`},
		{"a full session handed back as an agent token", http.StatusOK, fullBack},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pinConfigHome(t)
			t.Setenv("INSIDEOUT_TOKEN", human)
			api, seen := routedAPI(t, func(w http.ResponseWriter, r *http.Request, _ int) bool {
				if r.Method != http.MethodPost || r.URL.Path != "/agent/token" {
					return false
				}
				w.WriteHeader(tc.status)
				w.Write([]byte(tc.body))
				return true
			})

			r := callTool(t, Build(api.URL), 1, "whoami", `{}`)

			if !r.Result.IsError {
				t.Errorf("whoami succeeded with no agent token: %+v", r)
			}
			for _, got := range seen(human, "<human>") {
				if strings.HasSuffix(got, " Bearer <human>") && got != "POST /agent/token Bearer <human>" {
					t.Errorf("the full login was sent on %q", got)
				}
			}
		})
	}
}

// The login the exchange runs on is renewed by its own client like any other
// call: a saved session whose access token has expired still gets its agent
// token.
func TestTheSavedSessionRenewsInsideTheExchange(t *testing.T) {
	pinConfigHome(t)
	expired := fakeJWT("", "expired", time.Now().Add(-time.Hour), 15*time.Minute)
	renewed := fakeJWT("", "renewed", time.Now(), 15*time.Minute)
	grant := agentGrant("a1", time.Now(), 15*time.Minute)
	mint := mintsAs(grant)
	api, seen := routedAPI(t, func(w http.ResponseWriter, r *http.Request, nth int) bool {
		switch {
		case r.URL.Path == "/auth/refresh":
			fmt.Fprintf(w, `{"accessToken":%q,"refreshToken":"ref-2"}`, renewed)
			return true
		case r.Method == http.MethodPost && r.URL.Path == "/agent/token" && r.Header.Get("Authorization") != "Bearer "+renewed:
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return true
		}
		return mint(w, r, nth)
	})
	saveConfig(t, clientconfig.Config{APIBase: api.URL, Token: expired, RefreshToken: "ref-1"})

	if r := callTool(t, Build(""), 1, "whoami", `{}`); r.Result.IsError {
		t.Fatalf("whoami: %+v", r)
	}

	wantSeen(t, seen(expired, "<expired>", renewed, "<renewed>", grant.token, "<agent a1>"), []string{
		"POST /agent/token Bearer <expired>", "POST /auth/refresh ",
		"POST /agent/token Bearer <renewed>", "GET /me Bearer <agent a1>",
	})
}

// A token that already has the agent scope (the user minted one and handed it to
// the process) needs no exchange: it is sent as it is.
func TestAnAgentScopeTokenGivenToTheProcessIsSentAsItIs(t *testing.T) {
	pinConfigHome(t)
	agent := fakeJWT("agent", "given", time.Now(), 15*time.Minute)
	t.Setenv("INSIDEOUT_TOKEN", agent)
	api, seen := routedAPI(t, nil)

	if r := callTool(t, Build(api.URL), 1, "whoami", `{}`); r.Result.IsError {
		t.Fatalf("whoami: %+v", r)
	}

	wantSeen(t, seen(agent, "<agent>"), []string{"GET /me Bearer <agent>"})
}

// Signing a process in with the auth tools gives it a login too: the registry
// narrows right then, and the sign-in's own confirmation already rides an agent
// token.
func TestSigningInWithTheAuthToolsWithdrawsTheHumanOnlyTools(t *testing.T) {
	pinConfigHome(t)
	t.Cleanup(func() { setSessionToken("") })
	human := humanJWT()
	grant := agentGrant("a1", time.Now(), 15*time.Minute)
	mint := mintsAs(grant)
	api, seen := routedAPI(t, func(w http.ResponseWriter, r *http.Request, nth int) bool {
		if r.URL.Path == "/auth/login-code/verify" {
			fmt.Fprintf(w, `{"email":"a@example.com","username":"A","accessToken":%q,"refreshToken":"ref-1"}`, human)
			return true
		}
		return mint(w, r, nth)
	})
	s := Build(api.URL)
	if got := stillOffered(offeredTools(t, s)); len(got) != len(humanOnlyTools) {
		t.Fatalf("before any sign-in offered %v, want all %v", got, humanOnlyTools)
	}

	if r := callTool(t, s, 1, "auth_login_code_verify", `{"email":"a@example.com","code":"123456"}`); r.Result.IsError {
		t.Fatalf("auth_login_code_verify: %+v", r)
	}

	if got := stillOffered(offeredTools(t, s)); len(got) != 0 {
		t.Errorf("offered after the sign-in: %v", got)
	}
	wantSeen(t, seen(human, "<human>", grant.token, "<agent a1>"), []string{
		"POST /auth/login-code/verify ",
		"POST /agent/token Bearer <human>", "GET /me Bearer <agent a1>",
	})
}
