package apiclient

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// rv4-AIDEV-04 (docs/architecture/agent-token-scope.md): the exchange that turns
// the user's login into the agent-scope token a tool call carries.

// agentTestJWT shapes a token like the server's (sub, iat, exp, optional scope).
// Nothing here verifies a signature, so none is real; tag keeps tokens apart.
func agentTestJWT(sub, scope, tag string, issued time.Time, ttl time.Duration) string {
	claims := map[string]any{"sub": sub, "tag": tag, "iat": issued.Unix(), "exp": issued.Add(ttl).Unix()}
	if scope != "" {
		claims["scope"] = scope
	}
	payload, _ := json.Marshal(claims)
	enc := base64.RawURLEncoding
	return enc.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`)) + "." + enc.EncodeToString(payload) + "." + enc.EncodeToString([]byte("sig"))
}

func mintBody(token string) string { return fmt.Sprintf(`{"accessToken":%q,"scope":"agent"}`, token) }

// mintServer records "METHOD path Authorization" and answers the nth
// POST /agent/token with answer(n); any other route gets {"email":...}.
func mintServer(t *testing.T, answer func(nth int) (int, string)) (*httptest.Server, func(pairs ...string) []string) {
	t.Helper()
	ForgetAgentToken()
	t.Cleanup(ForgetAgentToken)
	var mu sync.Mutex
	var seen []string
	mints := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Method+" "+r.URL.Path+" "+r.Header.Get("Authorization"))
		isMint := r.Method == http.MethodPost && r.URL.Path == "/agent/token"
		if isMint {
			mints++
		}
		n := mints
		mu.Unlock()
		if isMint {
			status, body := answer(n)
			w.WriteHeader(status)
			w.Write([]byte(body))
			return
		}
		w.Write([]byte(`{"email":"a@example.com"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, func(pairs ...string) []string {
		mu.Lock()
		defer mu.Unlock()
		r := strings.NewReplacer(pairs...)
		out := make([]string, len(seen))
		for i, s := range seen {
			out[i] = r.Replace(s)
		}
		return out
	}
}

func wantRequests(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("the server saw\n  %s\nwant\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

func clientWith(base, token string) *Client {
	c := New(base)
	c.SetToken(token)
	return c
}

func TestParseTokenClaims(t *testing.T) {
	issued := time.Unix(1_800_000_000, 0)
	enc := base64.RawURLEncoding.EncodeToString
	header := enc([]byte(`{"alg":"HS256"}`))
	cases := []struct {
		name string
		raw  string
		ok   bool
		want TokenClaims
	}{
		{"a full session", agentTestJWT("u1", "", "x", issued, time.Hour), true,
			TokenClaims{Subject: "u1", IssuedAt: issued, ExpiresAt: issued.Add(time.Hour)}},
		{"an agent token", agentTestJWT("u1", "agent", "x", issued, time.Hour), true,
			TokenClaims{Subject: "u1", Scope: "agent", IssuedAt: issued, ExpiresAt: issued.Add(time.Hour)}},
		{"no expiry claim", header + "." + enc([]byte(`{"sub":"u1"}`)) + ".s", true, TokenClaims{Subject: "u1"}},
		{"empty", "", false, TokenClaims{}},
		{"opaque", "tok-abc", false, TokenClaims{}},
		{"two parts", header + "." + enc([]byte(`{}`)), false, TokenClaims{}},
		{"four parts", header + "." + enc([]byte(`{}`)) + ".s.s", false, TokenClaims{}},
		{"not base64", "!!!.@@@.###", false, TokenClaims{}},
		{"padded base64", header + "." + base64.URLEncoding.EncodeToString([]byte(`{"sub":"u"}`)) + ".s", false, TokenClaims{}},
		{"payload is not json", header + "." + enc([]byte(`{`)) + ".s", false, TokenClaims{}},
		{"payload is an array", header + "." + enc([]byte(`[1]`)) + ".s", false, TokenClaims{}},
		{"header is not an object", enc([]byte(`"HS256"`)) + "." + enc([]byte(`{}`)) + ".s", false, TokenClaims{}},
		{"scope is not a string", header + "." + enc([]byte(`{"scope":5}`)) + ".s", false, TokenClaims{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ParseTokenClaims(tc.raw)
			if ok != tc.ok || (ok && got != tc.want) {
				t.Errorf("ParseTokenClaims = %+v, %v; want %+v, %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestAgentScopedTradesTheLoginOnceAndSendsOnlyTheAgentToken(t *testing.T) {
	now := time.Now()
	login := agentTestJWT("u1", "", "login", now, 15*time.Minute)
	agent := agentTestJWT("u1", "agent", "a1", now, 15*time.Minute)
	srv, seen := mintServer(t, func(int) (int, string) { return http.StatusOK, mintBody(agent) })
	c := clientWith(srv.URL, login)

	for i := 0; i < 2; i++ {
		if _, err := AgentScoped(c).Whoami(); err != nil {
			t.Fatal(err)
		}
	}

	wantRequests(t, seen(login, "<login>", agent, "<agent>"), []string{
		"POST /agent/token Bearer <login>", "GET /me Bearer <agent>", "GET /me Bearer <agent>",
	})
}

// The token is minted again once about a third of its life is left (the margins
// are a minute either side of the boundary of a 90-minute life).
func TestAgentScopedMintsAgainAtAThirdOfTheLife(t *testing.T) {
	for _, tc := range []struct {
		name       string
		issuedAgo  time.Duration
		wantMints  int
		wantSecond string
	}{
		{"a minute before the boundary", 59 * time.Minute, 1, "a1"},
		{"a minute past the boundary", 61 * time.Minute, 2, "a2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now()
			login := agentTestJWT("u1", "", "login", now, time.Hour)
			a1 := agentTestJWT("u1", "agent", "a1", now.Add(-tc.issuedAgo), 90*time.Minute)
			a2 := agentTestJWT("u1", "agent", "a2", now, 90*time.Minute)
			srv, seen := mintServer(t, func(n int) (int, string) { return http.StatusOK, mintBody([]string{a1, a2}[min(n, 2)-1]) })
			c := clientWith(srv.URL, login)

			for i := 0; i < 2; i++ {
				if _, err := AgentScoped(c).Whoami(); err != nil {
					t.Fatal(err)
				}
			}

			mints := 0
			for _, s := range seen() {
				if strings.HasPrefix(s, "POST /agent/token") {
					mints++
				}
			}
			if mints != tc.wantMints {
				t.Errorf("minted %d times, want %d: %q", mints, tc.wantMints, seen(a1, "<a1>", a2, "<a2>", login, "<login>"))
			}
			last := seen(a1, "a1", a2, "a2")
			if got := last[len(last)-1]; got != "GET /me Bearer "+tc.wantSecond {
				t.Errorf("the second call sent %q, want the token %s", got, tc.wantSecond)
			}
		})
	}
}

// One slot, keyed by server and user: another user, or a forgotten login, mints.
func TestTheHeldAgentTokenBelongsToOneUserUntilForgotten(t *testing.T) {
	now := time.Now()
	agent := agentTestJWT("u1", "agent", "a", now, 15*time.Minute)
	srv, seen := mintServer(t, func(int) (int, string) { return http.StatusOK, mintBody(agent) })
	u1 := clientWith(srv.URL, agentTestJWT("u1", "", "l1", now, 15*time.Minute))
	u2 := clientWith(srv.URL, agentTestJWT("u2", "", "l2", now, 15*time.Minute))

	for _, c := range []*Client{u1, u1, u2, u2} {
		AgentScoped(c)
	}
	ForgetAgentToken()
	AgentScoped(u2)

	mints := 0
	for _, s := range seen() {
		if strings.HasPrefix(s, "POST /agent/token") {
			mints++
		}
	}
	if mints != 3 {
		t.Errorf("minted %d times for u1, u1, u2, u2, forget, u2; want 3 (u1, u2, u2 again)", mints)
	}
}

// The token minted on one server is never carried to another: the held token
// belongs to the server that issued it, whoever the user is (the same rule as
// the in-process session's, review-4 SEC-05).
func TestTheHeldAgentTokenIsNeverSentToAnotherServer(t *testing.T) {
	now := time.Now()
	login := agentTestJWT("u1", "", "login", now, 15*time.Minute)
	tokenA := agentTestJWT("u1", "agent", "from-a", now, 15*time.Minute)
	tokenB := agentTestJWT("u1", "agent", "from-b", now, 15*time.Minute)
	a, seenA := mintServer(t, func(int) (int, string) { return http.StatusOK, mintBody(tokenA) })
	b, seenB := mintServer(t, func(int) (int, string) { return http.StatusOK, mintBody(tokenB) })

	for _, srv := range []*httptest.Server{a, b} {
		if _, err := AgentScoped(clientWith(srv.URL, login)).Whoami(); err != nil {
			t.Fatal(err)
		}
	}

	wantRequests(t, seenA(login, "<login>", tokenA, "<from a>"), []string{"POST /agent/token Bearer <login>", "GET /me Bearer <from a>"})
	wantRequests(t, seenB(login, "<login>", tokenB, "<from b>"), []string{"POST /agent/token Bearer <login>", "GET /me Bearer <from b>"})
}

// A token the exchange cannot improve on is sent as it is: one that is not a JWT
// cannot carry a scope, one that already has a scope needs no trade (and the
// server would refuse the trade from an agent token).
func TestAgentScopedLeavesTokensThatAreNotAFullSessionAlone(t *testing.T) {
	srv, seen := mintServer(t, func(int) (int, string) { return http.StatusInternalServerError, `{}` })
	now := time.Now()
	for name, token := range map[string]string{
		"no token":      "",
		"opaque":        "tok-abc",
		"an agent":      agentTestJWT("u1", "agent", "a", now, time.Hour),
		"another scope": agentTestJWT("u1", "future", "a", now, time.Hour),
	} {
		c := clientWith(srv.URL, token)
		if got := AgentScoped(c); got != c {
			t.Errorf("%s: got a different client", name)
		}
	}
	wantRequests(t, seen(), []string{})
}

func TestAgentScopedFailsClosedWithTheReason(t *testing.T) {
	now := time.Now()
	login := agentTestJWT("u1", "", "login", now, 15*time.Minute)
	cases := []struct {
		name         string
		status       int
		body         string
		wantInErrors string
	}{
		{"a server that predates the route", http.StatusNotFound, `404 page not found`, "predates agent tokens"},
		{"refused", http.StatusForbidden, `{"error":"forbidden"}`, "403"},
		{"the login expired", http.StatusUnauthorized, `{"error":"unauthorized"}`, "unauthorized"},
		{"not json", http.StatusOK, `<html></html>`, "decode"},
		{"no token", http.StatusOK, `{"scope":"agent"}`, "not an agent-scope"},
		{"a token that is not a JWT", http.StatusOK, mintBody("opaque"), "not an agent-scope"},
		{"a full session answered as the agent token", http.StatusOK, mintBody(login), "not an agent-scope"},
		{"an agent token with no expiry", http.StatusOK, mintBody(strings.Join([]string{
			base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`)),
			base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"u1","scope":"agent"}`)), "s"}, ".")), "not an agent-scope"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, seen := mintServer(t, func(int) (int, string) { return tc.status, tc.body })

			_, err := AgentScoped(clientWith(srv.URL, login)).Whoami()

			if err == nil || !strings.Contains(err.Error(), tc.wantInErrors) {
				t.Errorf("error = %v, want one mentioning %q", err, tc.wantInErrors)
			}
			wantRequests(t, seen(login, "<login>"), []string{"POST /agent/token Bearer <login>"}) // no GET /me, with or without the login
		})
	}
}

// The scoped client carries the agent token alone: a refusal of it is final, and
// the saved session's refresh token is never spent on its behalf.
func TestTheAgentScopedClientNeverRenewsTheLogin(t *testing.T) {
	now := time.Now()
	login := agentTestJWT("u1", "", "login", now, 15*time.Minute)
	agent := agentTestJWT("u1", "agent", "a", now, 15*time.Minute)
	ForgetAgentToken()
	t.Cleanup(ForgetAgentToken)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/agent/token":
			w.Write([]byte(mintBody(agent)))
		case "/auth/refresh":
			t.Errorf("the saved session's refresh token was used for the agent client")
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	t.Cleanup(srv.Close)
	c := clientWith(srv.URL, login)
	c.SetRenewal("ref-1", Renewal{
		Reload: func() (string, string, bool) { return "", "", false },
		Save:   func(string, string) { t.Errorf("a rotated pair was saved") },
	})

	_, err := AgentScoped(c).Whoami()

	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("error = %v, want the refusal to stand", err)
	}
}
