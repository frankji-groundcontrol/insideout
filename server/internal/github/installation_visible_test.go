package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// rv4-COMP-05: a repo may be linked to a project only if GitHub shows it to
// this server, which is the App being installed on it or the repo being public.

// RepoIsPublic asks without a credential, so GitHub's answer is the one an
// anonymous visitor gets: 200 for a public repo, 404 for everything else.
func TestRepoIsPublic(t *testing.T) {
	var mu sync.Mutex
	var auths []string
	status, retry := http.StatusOK, ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		auths = append(auths, r.Header.Get("Authorization"))
		mu.Unlock()
		if r.URL.Path != "/repos/own/repo" {
			http.NotFound(w, r)
			return
		}
		if retry != "" {
			w.Header().Set("Retry-After", retry)
		}
		w.WriteHeader(status)
		fmt.Fprint(w, `{"private":false}`)
	}))
	defer srv.Close()
	point(t, srv)

	for _, tc := range []struct {
		name   string
		status int
		retry  string
		want   bool
		err    func(error) bool
	}{
		{"public", http.StatusOK, "", true, nil},
		{"private or missing", http.StatusNotFound, "", false, nil},
		{"rate limited", http.StatusForbidden, "30", false, func(e error) bool {
			var rle *RateLimitError
			return errors.As(e, &rle) && rle.RetryAfterSeconds == 30
		}},
		{"an outage is not 'private'", http.StatusBadGateway, "", false, func(e error) bool { return e != nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, retry = tc.status, tc.retry
			got, err := RepoIsPublic(context.Background(), "own", "repo")
			if got != tc.want {
				t.Errorf("RepoIsPublic = %v, want %v", got, tc.want)
			}
			if tc.err == nil && err != nil || tc.err != nil && !tc.err(err) {
				t.Errorf("err = %v", err)
			}
		})
	}
	mu.Lock()
	defer mu.Unlock()
	for _, a := range auths {
		if a != "" {
			t.Errorf("a request carried Authorization %q, want every request anonymous", a)
		}
	}
}

func TestRepoIsPublicRefusesNamesThatAreNotARepo(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits++ }))
	defer srv.Close()
	point(t, srv)
	for _, name := range [][2]string{{"..", "repo"}, {"own", "."}, {"", "repo"}, {"own", "re#po"}, {"a/b", "repo"}} {
		if _, err := RepoIsPublic(context.Background(), name[0], name[1]); err == nil {
			t.Errorf("RepoIsPublic(%q, %q) = no error, want a refusal", name[0], name[1])
		}
	}
	if hits != 0 {
		t.Errorf("%d request(s) went out for names that are not repos", hits)
	}
}

// RepoVisible: installed -> visible (the public lookup is not even made);
// not installed -> whatever an anonymous visitor sees; a failed question ->
// an error, never an answer.
func TestRepoVisible(t *testing.T) {
	var mu sync.Mutex
	installation, public := http.StatusOK, http.StatusOK
	var publicAsked int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case strings.HasSuffix(r.URL.Path, "/installation"):
			w.WriteHeader(installation)
			fmt.Fprint(w, `{"id":77}`)
		case r.URL.Path == "/repos/own/repo":
			publicAsked++
			w.WriteHeader(public)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	point(t, srv)
	app := testApp(t)

	set := func(i, p int) { mu.Lock(); installation, public, publicAsked = i, p, 0; mu.Unlock() }
	for _, tc := range []struct {
		name         string
		install, pub int
		want         bool
		wantErr      bool
		publicAsked  int
	}{
		{"installed, private", http.StatusOK, http.StatusNotFound, true, false, 0},
		{"not installed, public", http.StatusNotFound, http.StatusOK, true, false, 1},
		{"not installed, private or missing", http.StatusNotFound, http.StatusNotFound, false, false, 1},
		{"installation lookup fails", http.StatusBadGateway, http.StatusOK, false, true, 0},
		{"public lookup fails", http.StatusNotFound, http.StatusBadGateway, false, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			set(tc.install, tc.pub)
			got, err := app.RepoVisible(context.Background(), "own", "repo")
			if got != tc.want || (err != nil) != tc.wantErr {
				t.Errorf("RepoVisible = %v, %v; want %v, error=%v", got, err, tc.want, tc.wantErr)
			}
			mu.Lock()
			defer mu.Unlock()
			if publicAsked != tc.publicAsked {
				t.Errorf("anonymous lookups = %d, want %d", publicAsked, tc.publicAsked)
			}
		})
	}
}
