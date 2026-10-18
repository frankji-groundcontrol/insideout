package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// Review-4 USER-05: `setup --check` asked GET /me with no credentials, so a
// signed-out, expired or revoked login still read "ok ... (as expected)".
// With a credential in play it now asks as that user and reports the answer;
// with none it keeps the anonymous probe (cli_setup_check_test.go).

// checkAPIServer answers GET /me 200 for the one access token it knows and 401
// for anything else; POST /auth/refresh is answered by refresh. It records
// "METHOD path Authorization" per request.
func checkAPIServer(t *testing.T, known string, refresh func(w http.ResponseWriter, body string)) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		seen = append(seen, r.Method+" "+strings.TrimPrefix(r.URL.Path, "/api/v1")+" "+r.Header.Get("Authorization"))
		mu.Unlock()
		switch {
		case strings.HasSuffix(r.URL.Path, "/auth/refresh"):
			refresh(w, string(body))
		case r.Header.Get("Authorization") == "Bearer "+known:
			w.Write([]byte(`{"email":"you@example.org"}`))
		default:
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid or expired session"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
}

func refuseRefresh(w http.ResponseWriter, _ string) { w.WriteHeader(http.StatusUnauthorized) }

func checkWith(t *testing.T, cfg clientconfig.Config) (code int, out string) {
	t.Helper()
	pinnedNoRepo(t)
	path, err := clientconfig.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := clientconfig.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	out = captureStdout(t, func() { _, code = runSetupCommand([]string{"--check"}) })
	return code, out
}

func TestCheckAPIAsksAsTheSavedUserAndSaysSignedIn(t *testing.T) {
	srv, seen := checkAPIServer(t, "tok-a", refuseRefresh)

	code, out := checkWith(t, clientconfig.Config{APIBase: srv.URL + "/api/v1", Token: "tok-a"})

	if got := seen(); len(got) != 1 || got[0] != "GET /me Bearer tok-a" {
		t.Errorf("server saw %q, want one GET /me carrying the saved token", got)
	}
	if code != 0 || !strings.Contains(out, "signed in") {
		t.Errorf("exit %d, output %q; want exit 0 and a line saying signed in", code, out)
	}
}

func TestCheckAPIFailsWhenTheSavedLoginIsRejected(t *testing.T) {
	srv, _ := checkAPIServer(t, "tok-a", refuseRefresh)

	code, out := checkWith(t, clientconfig.Config{APIBase: srv.URL + "/api/v1", Token: "tok-expired"})

	if code != 1 || !strings.Contains(out, "insideout login") {
		t.Errorf("exit %d, output %q; want a failed check that says to sign in again", code, out)
	}
}

func TestCheckAPIRenewsAnExpiredAccessTokenLikeEveryOtherVerb(t *testing.T) {
	srv, seen := checkAPIServer(t, "acc-new", func(w http.ResponseWriter, body string) {
		if !strings.Contains(body, `"ref-old"`) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte(`{"ok":true,"accessToken":"acc-new","refreshToken":"ref-new"}`))
	})

	code, out := checkWith(t, clientconfig.Config{APIBase: srv.URL + "/api/v1", Token: "acc-old", RefreshToken: "ref-old"})

	if code != 0 || !strings.Contains(out, "signed in") {
		t.Errorf("exit %d, output %q; want a renewed, signed-in session\nserver saw %q", code, out, seen())
	}
	if got := savedConfig(t); got.Token != "acc-new" || got.RefreshToken != "ref-new" {
		t.Errorf("saved login = %+v, want the rotated pair", got)
	}
}

// A signed-in user whose server cannot be reached is offline, not signed out;
// a server that answers wrongly is a failure either way.
func TestCheckAPIWithASavedLoginKeepsItsOfflineSkipAndItsServerErrorFailure(t *testing.T) {
	if code, out := checkWith(t, clientconfig.Config{APIBase: "http://127.0.0.1:1/api/v1", Token: "tok-a"}); code != 0 || !strings.Contains(out, "unreachable") {
		t.Errorf("unreachable: exit %d, output %q; want a skip", code, out)
	}

	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
	t.Cleanup(down.Close)
	if code, out := checkWith(t, clientconfig.Config{APIBase: down.URL + "/api/v1", Token: "tok-a"}); code != 1 {
		t.Errorf("503: exit %d, output %q; want a failed check", code, out)
	}
}
