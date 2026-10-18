package github

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

var (
	testKeyOnce sync.Once
	testKey     *rsa.PrivateKey
)

// testApp is an App credential with a throwaway key; the stub servers below
// never verify the JWT's signature, only its shape.
func testApp(t *testing.T) *InstallationTokens {
	t.Helper()
	testKeyOnce.Do(func() {
		var err error
		if testKey, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
			panic(err)
		}
	})
	return NewInstallationTokens("424242", testKey)
}

// point aims the package at srv for one test.
func point(t *testing.T, srv *httptest.Server) {
	t.Helper()
	prev := apiBaseURL
	apiBaseURL = srv.URL
	t.Cleanup(func() { apiBaseURL = prev })
}

// rv4-FEAT-04: which installation covers a repo is GitHub's answer for that
// repo, asked with the App JWT. A 404 is "not installed", not a failure.
func TestInstallationForRepo(t *testing.T) {
	var hits atomic.Int32
	var auth string
	status, body := http.StatusOK, `{"id":77}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		auth = r.Header.Get("Authorization")
		if r.URL.Path != "/repos/own/repo/installation" {
			http.NotFound(w, r)
			return
		}
		if status == http.StatusForbidden {
			w.Header().Set("Retry-After", "42")
		}
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	defer srv.Close()
	point(t, srv)
	app := testApp(t)

	t.Run("installed", func(t *testing.T) {
		id, err := app.InstallationForRepo(context.Background(), "own", "repo")
		if err != nil || id != 77 {
			t.Fatalf("id=%d err=%v, want 77", id, err)
		}
		if tok := strings.TrimPrefix(auth, "Bearer "); tok == auth || strings.Count(tok, ".") != 2 {
			t.Errorf("Authorization = %q, want the App JWT as a bearer token", auth)
		}
	})
	t.Run("not installed", func(t *testing.T) {
		status, body = http.StatusNotFound, `{}`
		if _, err := app.InstallationForRepo(context.Background(), "own", "repo"); !errors.Is(err, ErrNotInstalled) {
			t.Fatalf("err = %v, want ErrNotInstalled", err)
		}
	})
	t.Run("rate limited", func(t *testing.T) {
		status = http.StatusForbidden
		var rle *RateLimitError
		if _, err := app.InstallationForRepo(context.Background(), "own", "repo"); !errors.As(err, &rle) || rle.RetryAfterSeconds != 42 {
			t.Fatalf("err = %v, want a RateLimitError with the Retry-After hint", err)
		}
	})
	t.Run("an outage is not 'not installed'", func(t *testing.T) {
		status = http.StatusInternalServerError
		if _, err := app.InstallationForRepo(context.Background(), "own", "repo"); err == nil || errors.Is(err, ErrNotInstalled) {
			t.Fatalf("err = %v, want a plain failure", err)
		}
	})
	t.Run("names that are not a repo never leave", func(t *testing.T) {
		before := hits.Load()
		for _, name := range [][2]string{{"..", "repo"}, {"own", "."}, {"", "repo"}, {"own", "re#po"}, {"a/b", "repo"}, {"own", "re po"}} {
			if _, err := app.InstallationForRepo(context.Background(), name[0], name[1]); err == nil {
				t.Errorf("InstallationForRepo(%q, %q) = no error, want a refusal", name[0], name[1])
			}
		}
		if n := hits.Load() - before; n != 0 {
			t.Errorf("%d request(s) carrying the App JWT went out for names that are not repos", n)
		}
	})
}

// RepoToken puts the two calls together: an installed repo yields that
// installation's token, an uninstalled one yields none (and no error), and a
// lookup that fails is an error, not a silent fall back to anonymous.
func TestRepoToken(t *testing.T) {
	lookup := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/installation"):
			w.WriteHeader(lookup)
			fmt.Fprint(w, `{"id":77}`)
		case r.Method == http.MethodPost && r.URL.Path == "/app/installations/77/access_tokens":
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"token":"ghs_inst77","expires_at":"2099-01-01T00:00:00Z"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	point(t, srv)
	app := testApp(t)

	if tok, err := app.RepoToken(context.Background(), "own", "repo"); err != nil || tok != "ghs_inst77" {
		t.Fatalf("installed: token=%q err=%v, want ghs_inst77", tok, err)
	}
	lookup = http.StatusNotFound
	if tok, err := app.RepoToken(context.Background(), "own", "repo"); err != nil || tok != "" {
		t.Fatalf("not installed: token=%q err=%v, want no token and no error", tok, err)
	}
	lookup = http.StatusBadGateway
	if tok, err := app.RepoToken(context.Background(), "own", "repo"); err == nil || tok != "" {
		t.Fatalf("lookup down: token=%q err=%v, want an error and no token", tok, err)
	}
}
