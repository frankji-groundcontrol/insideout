package mcpserver

import (
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// rv4-FR-F158, F163, F168, F175. An auth_* sign-in keeps only the access token.
// The in-process session cannot renew, always beats the saved login, and a final
// 401 tells the user to run `insideout login`, which writes a file this process
// will not read. These tests use the real apiclient against an httptest API.

func rvMcpText(t *testing.T, r rpcResult) string {
	t.Helper()
	if len(r.Result.Content) == 0 {
		t.Fatalf("tool result has no content: %+v", r)
	}
	return r.Result.Content[0].Text
}

// TestRvMcpInProcessSignInRenewsAndRearms covers the shared root of F158, F163,
// F168 and F175: the refresh token from an auth_* sign-in has to renew an expired
// access token, and a later sign-in has to arm renewal again after a refusal.
func TestRvMcpInProcessSignInRenewsAndRearms(t *testing.T) {
	pinConfigHome(t)
	t.Cleanup(func() { setSessionToken("") })
	expired1 := fakeJWT("", "e1", time.Now().Add(-time.Hour), 15*time.Minute)
	fresh1 := fakeJWT("", "f1", time.Now(), 15*time.Minute)
	expired2 := fakeJWT("", "e2", time.Now().Add(-time.Hour), 15*time.Minute)
	fresh2 := fakeJWT("", "f2", time.Now(), 15*time.Minute)
	g1 := agentGrant("a1", time.Now(), 15*time.Minute)
	g2 := agentGrant("a2", time.Now(), 15*time.Minute)

	var logins int
	sealed := false
	api, seen := routedAPI(t, func(w http.ResponseWriter, r *http.Request, _ int) bool {
		authz := r.Header.Get("Authorization")
		switch {
		case r.URL.Path == "/auth/login-code/verify":
			logins++
			access, refresh := expired1, "ref-1"
			if logins > 1 {
				access, refresh = expired2, "ref-2"
			}
			fmt.Fprintf(w, `{"email":"a@example.com","username":"A","accessToken":%q,"refreshToken":%q}`, access, refresh)
			return true
		case r.URL.Path == "/auth/refresh":
			raw, _ := io.ReadAll(r.Body)
			body := string(raw)
			switch {
			case strings.Contains(body, "ref-2"):
				fmt.Fprintf(w, `{"accessToken":%q,"refreshToken":"ref-2b"}`, fresh2)
			case sealed:
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
			default:
				fmt.Fprintf(w, `{"accessToken":%q,"refreshToken":"ref-1b"}`, fresh1)
			}
			return true
		case r.Method == http.MethodPost && r.URL.Path == "/agent/token":
			switch {
			case authz == "Bearer "+fresh1 && !sealed:
				w.Write([]byte(g1.body()))
			case authz == "Bearer "+fresh2:
				w.Write([]byte(g2.body()))
			default:
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
			}
			return true
		case r.URL.Path == "/me":
			if authz == "Bearer "+g1.token || authz == "Bearer "+g2.token {
				w.Write([]byte(`{"email":"a@example.com","username":"A"}`))
				return true
			}
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return true
		}
		return false
	})
	s := Build(api.URL)
	verify := func(id int, code string) rpcResult {
		return callTool(t, s, id, "auth_login_code_verify", fmt.Sprintf(`{"email":"a@example.com","code":%q}`, code))
	}

	first := verify(1, "111111")
	if first.Result.IsError {
		t.Fatalf("auth_login_code_verify with an expired access token and a live refresh token: %s", rvMcpText(t, first))
	}
	if !slices.Contains(seen(), "POST /auth/refresh ") {
		t.Fatalf("the sign-in never renewed; the API saw %q", seen())
	}

	sealed = true
	apiclient.ForgetAgentToken()
	failed := callTool(t, s, 2, "whoami", `{}`)
	if !failed.Result.IsError || !strings.Contains(rvMcpText(t, failed), "auth_login_code_request") {
		t.Fatalf("a refused in-process refresh must name the auth_* tools, got %+v", failed.Result)
	}

	// Password tools are gone while the JWT is held, and the code-verify tool
	// stays. Sign out, then sign in again: the new client has to be armed,
	// because a refusal stays on the client that saw it until SetRenewal.
	if out := callTool(t, s, 3, "auth_logout", `{}`); out.Result.IsError {
		t.Fatalf("auth_logout: %s", rvMcpText(t, out))
	}
	second := verify(4, "222222")
	if second.Result.IsError {
		t.Fatalf("a new sign-in did not re-arm renewal after the refusal: %s", rvMcpText(t, second))
	}
	who := callTool(t, s, 5, "whoami", `{}`)
	if who.Result.IsError || !strings.Contains(rvMcpText(t, who), "a@example.com") {
		t.Fatalf("whoami after the second sign-in: %+v\nAPI saw %q", who.Result, seen(expired1, "<e1>", fresh1, "<f1>", expired2, "<e2>", fresh2, "<f2>", g1.token, "<g1>", g2.token, "<g2>"))
	}
}

// TestRvMcpRejectedInProcessFallsBackToSavedLogin is F158 and F175: once the
// in-process credential is finally refused, the saved login (what `insideout
// login` writes) has to win. At HEAD the expired token is sent forever.
func TestRvMcpRejectedInProcessFallsBackToSavedLogin(t *testing.T) {
	pinConfigHome(t)
	t.Cleanup(func() { setSessionToken("") })
	expired := fakeJWT("", "expired", time.Now().Add(-time.Hour), 15*time.Minute)
	api, seen := routedAPI(t, func(w http.ResponseWriter, r *http.Request, _ int) bool {
		if r.Header.Get("Authorization") == "Bearer "+expired {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return true
		}
		return false
	})
	saveConfig(t, clientconfig.Config{APIBase: api.URL, Token: "tok-saved", RefreshToken: "ref-saved"})
	setSessionToken(expired)

	raw, err := client(api.URL).Whoami()
	if err != nil {
		t.Fatalf("whoami = %v; the in-process token still shadows the saved login", err)
	}
	if !strings.Contains(string(raw), "a@example.com") {
		t.Fatalf("whoami = %s", raw)
	}
	got := seen(expired, "<expired>", "tok-saved", "<saved>")
	if !slices.Contains(got, "GET /me Bearer <saved>") {
		t.Fatalf("the saved login was not used: %q", got)
	}
}

// TestRvMcpInProcessFinal401NamesAuthTools is F158, F163 and F175 with nothing
// saved to fall back to: the advice has to name the auth_* tools, not
// `insideout login`, which cannot change this process.
func TestRvMcpInProcessFinal401NamesAuthTools(t *testing.T) {
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

	_, err := client(api.URL).Whoami()
	if err == nil || !strings.Contains(err.Error(), "auth_login_code_request") || !strings.Contains(err.Error(), "auth_logout") {
		t.Fatalf("whoami = %v, want advice that names auth_login_code_request and auth_logout", err)
	}
}

// TestRvMcpLogoutRestoresPasswordTools is the F168 half a self-hosted process
// without mail hits: auth_login is withdrawn on the first JWT and never comes
// back, so auth_logout leaves no way to sign in again.
func TestRvMcpLogoutRestoresPasswordTools(t *testing.T) {
	pinConfigHome(t)
	t.Cleanup(func() { setSessionToken("") })
	human := humanJWT()
	grant := agentGrant("a1", time.Now(), 15*time.Minute)
	mint := mintsAs(grant)
	api, _ := routedAPI(t, func(w http.ResponseWriter, r *http.Request, nth int) bool {
		if r.URL.Path == "/auth/login" {
			fmt.Fprintf(w, `{"email":"a@example.com","username":"A","accessToken":%q,"refreshToken":"ref-1"}`, human)
			return true
		}
		return mint(w, r, nth)
	})
	s := Build(api.URL)
	if r := callTool(t, s, 1, "auth_login", `{"email":"a@example.com","password":"pw123456"}`); r.Result.IsError {
		t.Fatalf("auth_login: %s", rvMcpText(t, r))
	}
	if slices.Contains(offeredTools(t, s), "auth_login") {
		t.Fatal("auth_login is still offered while a JWT login is held")
	}
	if r := callTool(t, s, 2, "auth_logout", `{}`); r.Result.IsError {
		t.Fatalf("auth_logout: %s", rvMcpText(t, r))
	}
	if !slices.Contains(offeredTools(t, s), "auth_login") {
		t.Fatal("auth_logout left auth_login withdrawn, and nothing re-adds it")
	}
}

// TestRvMcpHeldLoginRefusesSessionSwap is F176. The agent profile keeps
// auth_login_code_verify, auth_magic_link_consume and auth_logout on the list
// (withholding them is an owner decision). Calling a swap tool while a login is
// held must not replace the session; auth_logout must demand confirm=true.
func TestRvMcpHeldLoginRefusesSessionSwap(t *testing.T) {
	pinConfigHome(t)
	t.Cleanup(func() { setSessionToken("") })
	human := humanJWT()
	attacker := fakeJWT("", "attacker", time.Now(), 15*time.Minute)
	t.Setenv("INSIDEOUT_TOKEN", human)
	owner := agentGrant("owner", time.Now(), 15*time.Minute)
	stolen := agentGrant("stolen", time.Now(), 15*time.Minute)
	api, seen := routedAPI(t, func(w http.ResponseWriter, r *http.Request, nth int) bool {
		switch {
		case r.URL.Path == "/auth/magic-link/consume":
			fmt.Fprintf(w, `{"email":"attacker@example.com","username":"Atk","accessToken":%q,"refreshToken":"ref-atk"}`, attacker)
			return true
		case r.URL.Path == "/auth/login-code/verify":
			fmt.Fprintf(w, `{"email":"attacker@example.com","username":"Atk","accessToken":%q,"refreshToken":"ref-atk"}`, attacker)
			return true
		case r.Method == http.MethodPost && r.URL.Path == "/agent/token":
			if r.Header.Get("Authorization") == "Bearer "+attacker {
				w.Write([]byte(stolen.body()))
				return true
			}
			w.Write([]byte(owner.body()))
			return true
		}
		return mintsAs(owner)(w, r, nth)
	})
	s := Build(api.URL)
	names := offeredTools(t, s)
	for _, tool := range []string{"auth_login_code_verify", "auth_magic_link_consume", "auth_logout"} {
		if !slices.Contains(names, tool) {
			t.Errorf("%s left the tool list; refuse the call instead of hiding the tool", tool)
		}
	}

	swap := callTool(t, s, 1, "auth_magic_link_consume", `{"token":"attacker-link"}`)
	if !swap.Result.IsError || !strings.Contains(rvMcpText(t, swap), "auth_logout") {
		t.Fatalf("auth_magic_link_consume while a login is held: %s", rvMcpText(t, swap))
	}
	code := callTool(t, s, 2, "auth_login_code_verify", `{"email":"a@example.com","code":"999999"}`)
	if !code.Result.IsError || !strings.Contains(rvMcpText(t, code), "auth_logout") {
		t.Fatalf("auth_login_code_verify while a login is held: %s", rvMcpText(t, code))
	}
	if out := callTool(t, s, 3, "auth_logout", `{}`); !out.Result.IsError || !strings.Contains(rvMcpText(t, out), "confirm") {
		t.Fatalf("auth_logout without confirm: %s", rvMcpText(t, out))
	}

	who := callTool(t, s, 4, "whoami", `{}`)
	if who.Result.IsError {
		t.Fatalf("whoami: %s", rvMcpText(t, who))
	}
	got := seen(human, "<human>", attacker, "<attacker>", owner.token, "<owner>", stolen.token, "<stolen>")
	for _, line := range got {
		if strings.Contains(line, "<attacker>") || strings.Contains(line, "<stolen>") || strings.Contains(line, "/auth/magic-link/consume") || strings.Contains(line, "/auth/login-code/verify") {
			t.Fatalf("the held login was replaced: %q", got)
		}
	}

	if out := callTool(t, s, 5, "auth_logout", `{"confirm":true}`); out.Result.IsError {
		t.Fatalf("auth_logout confirm=true: %s", rvMcpText(t, out))
	}
	again := callTool(t, s, 6, "auth_magic_link_consume", `{"token":"attacker-link"}`)
	if again.Result.IsError || !strings.Contains(rvMcpText(t, again), "attacker@example.com") {
		t.Fatalf("after auth_logout, a sign-in must work: %s", rvMcpText(t, again))
	}
}
