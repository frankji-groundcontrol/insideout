package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// Review-4 USER-04: the access token a login saves lives 15 minutes, so a
// saved login that cannot renew itself dies at the first verb after that. The
// server hands out a rotating one-time refresh token with every sign-in
// (POST /auth/refresh takes it as {"refreshToken": ...} and answers a new
// pair); the CLI must use it once on a 401, save the rotated pair and retry.
// Every test runs the real main() in a child process.

// expiredAPI answers GET /me only for "Bearer acc-new" (anything else gets the
// server's plain 401) and hands POST /auth/refresh to refresh. It records every
// request as "METHOD path Authorization body".
type expiredAPI struct {
	*httptest.Server
	mu   sync.Mutex
	seen []string
}

func newExpiredAPI(t *testing.T, refresh func(w http.ResponseWriter, api *expiredAPI, body string)) *expiredAPI {
	t.Helper()
	a := &expiredAPI{}
	a.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		a.mu.Lock()
		a.seen = append(a.seen, strings.Join(strings.Fields(r.Method+" "+r.URL.Path+" "+r.Header.Get("Authorization")+" "+string(body)), " "))
		a.mu.Unlock()
		switch {
		case r.URL.Path == "/auth/refresh":
			refresh(w, a, string(body))
		case r.Header.Get("Authorization") == "Bearer acc-new":
			w.Write([]byte(`{"email":"a@example.org"}`))
		default:
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid or expired session"}`))
		}
	}))
	t.Cleanup(a.Close)
	return a
}

func (a *expiredAPI) requests() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.seen...)
}

// swapPair accepts the refresh token ref-old, once, for the pair acc-new/ref-new.
func swapPair(w http.ResponseWriter, a *expiredAPI, body string) {
	if !strings.Contains(body, `"ref-old"`) {
		rejectRefresh(w, a, body)
		return
	}
	w.Write([]byte(`{"ok":true,"accessToken":"acc-new","refreshToken":"ref-new"}`))
}

func rejectRefresh(w http.ResponseWriter, _ *expiredAPI, _ string) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"refresh token expired or already used"}`))
}

func savedConfig(t *testing.T) clientconfig.Config {
	t.Helper()
	path, err := clientconfig.DefaultPath()
	if err != nil {
		t.Fatalf("resolve config path: %v", err)
	}
	cfg, err := clientconfig.Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg
}

func TestExpiredTokenIsRenewedOnceAndTheRotatedPairIsSaved(t *testing.T) {
	api := newExpiredAPI(t, swapPair)
	writeClientConfig(t, clientconfig.Config{APIBase: api.URL, Token: "acc-old", RefreshToken: "ref-old"})

	res := runCLI(t, cliEnv(), "", "whoami")

	if res.code != 0 || !strings.Contains(res.stdout, "a@example.org") {
		t.Fatalf("whoami with an expired access token failed: exit %d\n%s", res.code, res.output())
	}
	want := []string{
		"GET /me Bearer acc-old",
		`POST /auth/refresh {"refreshToken":"ref-old"}`,
		"GET /me Bearer acc-new",
	}
	if got := api.requests(); !reflect.DeepEqual(got, want) {
		t.Errorf("server saw\n%q\nwant\n%q", got, want)
	}
	// ref-old is dead on the server now: the file must hold the rotated pair.
	if got, want := savedConfig(t), (clientconfig.Config{APIBase: api.URL, Token: "acc-new", RefreshToken: "ref-new"}); got != want {
		t.Errorf("saved config = %+v, want %+v", got, want)
	}
}

func TestRejectedRefreshIsTriedOnceAndSaysToLogInAgain(t *testing.T) {
	api := newExpiredAPI(t, rejectRefresh)
	old := clientconfig.Config{APIBase: api.URL, Token: "acc-old", RefreshToken: "ref-old"}
	writeClientConfig(t, old)

	res := runCLI(t, cliEnv(), "", "whoami")

	if res.code == 0 {
		t.Fatalf("exit 0 although the session cannot be renewed\n%s", res.output())
	}
	if !strings.Contains(res.stderr, "insideout login") {
		t.Errorf("stderr does not tell the user to sign in again:\n%s", res.stderr)
	}
	want := []string{"GET /me Bearer acc-old", `POST /auth/refresh {"refreshToken":"ref-old"}`}
	if got := api.requests(); !reflect.DeepEqual(got, want) {
		t.Errorf("server saw\n%q\nwant exactly one try of each\n%q", got, want)
	}
	if got := savedConfig(t); got != old {
		t.Errorf("a failed refresh changed the saved login: %+v", got)
	}
}

// Two commands started together hold the same pair; the server rotates it for
// the first and refuses the replay. The second must pick up what the first
// saved, not give up and not overwrite it.
func TestAnotherCommandThatRotatedFirstIsAdopted(t *testing.T) {
	api := newExpiredAPI(t, func(w http.ResponseWriter, a *expiredAPI, body string) {
		path, _ := clientconfig.DefaultPath()
		clientconfig.Save(path, clientconfig.Config{APIBase: a.URL, Token: "acc-new", RefreshToken: "ref-new"})
		rejectRefresh(w, a, body)
	})
	writeClientConfig(t, clientconfig.Config{APIBase: api.URL, Token: "acc-old", RefreshToken: "ref-old"})

	res := runCLI(t, cliEnv(), "", "whoami")

	if res.code != 0 || !strings.Contains(res.stdout, "a@example.org") {
		t.Fatalf("whoami did not adopt the pair the other command saved: exit %d\n%s", res.code, res.output())
	}
	want := []string{"GET /me Bearer acc-old", `POST /auth/refresh {"refreshToken":"ref-old"}`, "GET /me Bearer acc-new"}
	if got := api.requests(); !reflect.DeepEqual(got, want) {
		t.Errorf("server saw\n%q\nwant\n%q", got, want)
	}
	if got, want := savedConfig(t), (clientconfig.Config{APIBase: api.URL, Token: "acc-new", RefreshToken: "ref-new"}); got != want {
		t.Errorf("saved config = %+v, want the other command's pair %+v", got, want)
	}
}

// A token the caller supplied (INSIDEOUT_TOKEN) is not the saved session, and
// the saved session belongs to its own server: neither may be renewed with it,
// or one identity would be swapped for another behind the caller's back.
func TestOnlyTheSavedSessionOfTheSameServerIsRenewed(t *testing.T) {
	api, other := newExpiredAPI(t, swapPair), newExpiredAPI(t, swapPair)
	saved := clientconfig.Config{APIBase: api.URL, Token: "acc-old", RefreshToken: "ref-old"}
	writeClientConfig(t, saved)

	envRes := runCLI(t, cliEnv("INSIDEOUT_TOKEN=tok-env"), "", "whoami")
	otherRes := runCLI(t, cliEnv(), "", "whoami", "--api", other.URL)

	if envRes.code == 0 || otherRes.code == 0 {
		t.Errorf("a 401 must stay an error: exits %d and %d", envRes.code, otherRes.code)
	}
	for name, a := range map[string]*expiredAPI{"saved server": api, "other server": other} {
		for _, r := range a.requests() {
			if strings.Contains(r, "/auth/refresh") || strings.Contains(r, "ref-old") {
				t.Errorf("%s was asked to renew a session that is not the caller's: %q", name, r)
			}
		}
	}
	if got := savedConfig(t); got != saved {
		t.Errorf("saved config changed: %+v", got)
	}
}
