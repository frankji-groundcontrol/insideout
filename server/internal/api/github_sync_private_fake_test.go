package api

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/github"
)

// The GitHub half of the rv4 row 3.10 tests (commit sync with the App
// credential, binding a repo, deployment evidence). fakeGitHub answers what
// api.github.com would, in-process: no network, no real credential. It models
// the one rule the findings turn on: a private repo is invisible (404) to any
// request that does not carry the token of the installation it belongs to.
//
// It is NOT the real GitHub: these tests prove how this server behaves toward
// a GitHub that answers like this, not that the App installation flow works
// on github.com.

// fakeRepo is one repository the fake knows.
type fakeRepo struct {
	private bool
	install int64    // the installation that can see it (0 = none)
	shas    []string // commits, newest first
	link    string   // Link header on the commits reply
	guide   string   // insideout.yaml ("" = the repo has none)
}

// fakeCall is one outbound request the server made.
type fakeCall struct{ Host, Method, Path, Auth string }

type fakeGitHub struct {
	mu    sync.Mutex
	repos map[string]*fakeRepo // lower-cased owner/repo
	calls []fakeCall
	down  bool // every api.github.com call answers 500
}

// fakeToken is the installation token the fake mints for an installation.
func fakeToken(install int64) string { return fmt.Sprintf("ghs_inst%d", install) }

// newFakeGitHub puts the fake behind http.DefaultTransport for one test: the
// github package's clients have no transport of their own, so they follow it
// at request time. Package api runs its tests sequentially, so the swap is
// race-free (same trick as fakeResend). Any host other than api.github.com
// under a reserved TLD is recorded and refused instead of resolved.
func newFakeGitHub(t *testing.T) *fakeGitHub {
	t.Helper()
	f := &fakeGitHub{repos: map[string]*fakeRepo{}}
	prev := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		f.record(r)
		switch host := r.URL.Hostname(); {
		case host == "api.github.com":
			rec := httptest.NewRecorder()
			f.serve(rec, r)
			resp := rec.Result()
			resp.Request = r
			return resp, nil
		case strings.HasSuffix(host, ".example"):
			return nil, fmt.Errorf("fake github: refused %s", r.URL.Host)
		}
		return prev.RoundTrip(r)
	})
	t.Cleanup(func() { http.DefaultTransport = prev })
	return f
}

func (f *fakeGitHub) record(r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fakeCall{r.URL.Hostname(), r.Method, r.URL.Path, r.Header.Get("Authorization")})
}

// addRepo registers owner/repo; name is matched case-insensitively, like GitHub.
func (f *fakeGitHub) addRepo(name string, r fakeRepo) *fakeRepo {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.repos[strings.ToLower(name)] = &r
	return &r
}

// setCommits replaces the commits page the repo answers with.
func (f *fakeGitHub) setCommits(name, link string, shas ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.repos[strings.ToLower(name)]
	r.shas, r.link = shas, link
}

func (f *fakeGitHub) setDown(down bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.down = down
}

// callsMatching returns the recorded calls whose method+path contains sub.
func (f *fakeGitHub) callsMatching(sub string) []fakeCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []fakeCall
	for _, c := range f.calls {
		if strings.Contains(c.Method+" "+c.Path, sub) {
			out = append(out, c)
		}
	}
	return out
}

// foreignCalls are the requests that went anywhere but api.github.com.
func (f *fakeGitHub) foreignCalls() []fakeCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []fakeCall
	for _, c := range f.calls {
		if c.Host != "api.github.com" {
			out = append(out, c)
		}
	}
	return out
}

// sawSecret reports whether any outbound request carried s in Authorization.
func (f *fakeGitHub) sawSecret(s string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if strings.Contains(c.Auth, s) {
			return true
		}
	}
	return false
}

func (f *fakeGitHub) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	auth := r.Header.Get("Authorization")
	if f.down {
		http.Error(w, "boom", http.StatusInternalServerError)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	switch {
	case r.Method == http.MethodPost && len(parts) == 4 && parts[0] == "app" && parts[3] == "access_tokens":
		var id int64
		fmt.Sscan(parts[2], &id)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"token":%q,"expires_at":%q}`, fakeToken(id), time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
	case r.Method == http.MethodGet && len(parts) >= 3 && parts[0] == "repos":
		repo := f.repos[strings.ToLower(parts[1]+"/"+parts[2])]
		sub := strings.Join(parts[3:], "/")
		switch {
		case repo == nil:
			http.NotFound(w, r)
		case sub == "installation": // App-JWT authenticated: is the App installed here?
			if repo.install == 0 || strings.Count(strings.TrimPrefix(auth, "Bearer "), ".") != 2 {
				http.NotFound(w, r)
				return
			}
			fmt.Fprintf(w, `{"id":%d}`, repo.install)
		case repo.private && auth != "Bearer "+fakeToken(repo.install):
			http.NotFound(w, r) // what GitHub answers for a private repo you cannot see
		case sub == "":
			fmt.Fprint(w, `{"private":false}`)
		case sub == "commits":
			if repo.link != "" {
				w.Header().Set("Link", repo.link)
			}
			fmt.Fprint(w, "[")
			for i, sha := range repo.shas {
				if i > 0 {
					fmt.Fprint(w, ",")
				}
				fmt.Fprintf(w, `{"sha":%q,"commit":{"message":"change %s\nbody","author":{"name":"Dev","date":"2026-10-01T10:00:00Z"}}}`, sha, sha)
			}
			fmt.Fprint(w, "]")
		case sub == "contents/insideout.yaml" && repo.guide != "":
			fmt.Fprint(w, repo.guide)
		default:
			http.NotFound(w, r)
		}
	default:
		http.NotFound(w, r)
	}
}

var (
	appKeyOnce sync.Once
	appKey     *rsa.PrivateKey
)

// appServerWorld is a route world whose server holds a GitHub App credential
// (an in-process key; the fake never verifies the JWT's signature) and a
// webhook secret, so signed deliveries and installation tokens both work.
func appServerWorld(t *testing.T) *routeWorld {
	t.Helper()
	w := newRouteWorld(t)
	appKeyOnce.Do(func() {
		var err error
		if appKey, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
			panic(err)
		}
	})
	w.srv.cfg.GithubWebhookSecret = hookSecret
	w.srv.ghTokens = github.NewInstallationTokens("424242", appKey)
	return w
}
