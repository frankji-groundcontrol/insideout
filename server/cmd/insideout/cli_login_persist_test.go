package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// Review-4 USER-03: `insideout login` printed its token and kept nothing, so
// the documented `login` then `whoami` sequence failed with "unauthorized".
// A sign-in now saves the session where `insideout setup` does (one 0600
// config), says where on stderr, and leaves stdout as the bare access token.
// Every test runs the real main() in a child process.

// signInAPI is the sign-in side of the API: the password pw-for-tests signs in
// with the pair acc-<tag>/ref-<tag> (login answers 200, register 201, both
// with the server's session body), and GET /me knows only that access token.
type signInAPI struct {
	*httptest.Server
	mu   sync.Mutex
	seen []string
}

func newSignInAPI(t *testing.T, tag string) *signInAPI {
	t.Helper()
	a := &signInAPI{}
	a.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		a.mu.Lock()
		a.seen = append(a.seen, r.Method+" "+r.URL.Path+" "+r.Header.Get("Authorization"))
		a.mu.Unlock()
		var in struct{ Password string }
		_ = json.Unmarshal(body, &in)
		switch {
		case r.URL.Path == "/me" && r.Header.Get("Authorization") == "Bearer acc-"+tag:
			w.Write([]byte(`{"email":"you@example.org"}`))
		case r.URL.Path == "/auth/login" || r.URL.Path == "/auth/register":
			if in.Password != "pw-for-tests" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"invalid credentials"}`))
				return
			}
			if r.URL.Path == "/auth/register" {
				w.WriteHeader(http.StatusCreated)
			}
			w.Write([]byte(`{"id":"u1","email":"you@example.org","username":"you","accessToken":"acc-` + tag + `","refreshToken":"ref-` + tag + `"}`))
		default:
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"authentication required"}`))
		}
	}))
	t.Cleanup(a.Close)
	return a
}

func (a *signInAPI) last() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.seen[len(a.seen)-1]
}

func configPath(t *testing.T) string {
	t.Helper()
	p, err := clientconfig.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	return p
}

func TestLoginSavesTheSessionSoWhoamiWorksNext(t *testing.T) {
	api := newSignInAPI(t, "a")
	writeClientConfig(t, clientconfig.Config{})

	login := runCLI(t, cliEnv(), "pw-for-tests\n", "login", "--api", api.URL, "you@example.org")
	whoami := runCLI(t, cliEnv(), "", "whoami") // no --api, no env: the saved login names the server

	if login.code != 0 || login.stdout != "acc-a\n" {
		t.Fatalf("login: exit %d, stdout %q; want 0 and the bare access token\n%s", login.code, login.stdout, login.stderr)
	}
	if !strings.Contains(login.stderr, configPath(t)) {
		t.Errorf("login stderr %q does not say where the login was saved (%s)", login.stderr, configPath(t))
	}
	if got, want := savedConfig(t), (clientconfig.Config{APIBase: api.URL, Token: "acc-a", RefreshToken: "ref-a"}); got != want {
		t.Errorf("saved login = %+v, want %+v", got, want)
	}
	if fi, err := os.Stat(configPath(t)); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("config mode = %v (%v), want 0600", fi.Mode().Perm(), err)
	}
	if whoami.code != 0 || !strings.Contains(whoami.stdout, "you@example.org") || api.last() != "GET /me Bearer acc-a" {
		t.Errorf("whoami right after login: exit %d, last request %q\n%s", whoami.code, api.last(), whoami.output())
	}
}

func TestRegisterSavesTheSessionToo(t *testing.T) {
	api := newSignInAPI(t, "a")
	writeClientConfig(t, clientconfig.Config{})

	res := runCLI(t, cliEnv(), "pw-for-tests\n", "register", "--api", api.URL, "you@example.org")

	if res.code != 0 || res.stdout != "acc-a\n" {
		t.Fatalf("register: exit %d, stdout %q\n%s", res.code, res.stdout, res.stderr)
	}
	if got, want := savedConfig(t), (clientconfig.Config{APIBase: api.URL, Token: "acc-a", RefreshToken: "ref-a"}); got != want {
		t.Errorf("saved login = %+v, want %+v", got, want)
	}
}

// There is one saved login per machine: signing in to another server moves it,
// and nothing of the old server's session stays behind to be sent anywhere.
func TestLoginToAnotherServerReplacesTheSavedLogin(t *testing.T) {
	a, b := newSignInAPI(t, "a"), newSignInAPI(t, "b")
	writeClientConfig(t, clientconfig.Config{APIBase: a.URL, Token: "acc-a", RefreshToken: "ref-a"})

	res := runCLI(t, cliEnv(), "pw-for-tests\n", "login", "--api", b.URL, "you@example.org")

	if got, want := savedConfig(t), (clientconfig.Config{APIBase: b.URL, Token: "acc-b", RefreshToken: "ref-b"}); res.code != 0 || got != want {
		t.Errorf("after login --api B: exit %d, saved %+v, want %+v\n%s", res.code, got, want, res.stderr)
	}
}

func TestFailedLoginKeepsTheSavedLogin(t *testing.T) {
	a := newSignInAPI(t, "a")
	kept := clientconfig.Config{APIBase: a.URL, Token: "acc-a", RefreshToken: "ref-a"}
	writeClientConfig(t, kept)

	res := runCLI(t, cliEnv(), "wrong-password\n", "login", "--api", a.URL, "you@example.org")

	if res.code == 0 || res.stdout != "" || savedConfig(t) != kept {
		t.Errorf("failed login: exit %d, stdout %q, saved %+v; want an error, no token, the old login kept", res.code, res.stdout, savedConfig(t))
	}
}

// The sign-in worked, so the token is still printed (INSIDEOUT_TOKEN="$(insideout login …)"
// must keep working on a read-only home); the missing save is said out loud.
func TestLoginThatCannotBeSavedStillPrintsTheToken(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	api := newSignInAPI(t, "a")
	writeClientConfig(t, clientconfig.Config{})
	dir := filepath.Dir(configPath(t))
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })

	res := runCLI(t, cliEnv(), "pw-for-tests\n", "login", "--api", api.URL, "you@example.org")

	if res.code != 0 || res.stdout != "acc-a\n" || !strings.Contains(res.stderr, "could not save") {
		t.Errorf("exit %d, stdout %q, stderr %q; want the token printed and a warning that nothing was saved", res.code, res.stdout, res.stderr)
	}
}
