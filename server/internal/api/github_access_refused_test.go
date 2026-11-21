package api

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// rvGhBindRefuse makes GitHub answer 403 with message to the requests whose path
// ends in suffix; everything else goes on to the fake installed before it. The
// answer carries no rate-limit header unless header/value pairs are given.
func rvGhBindRefuse(t *testing.T, suffix, message string, headers ...string) {
	t.Helper()
	next := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Hostname() != "api.github.com" || !strings.HasSuffix(r.URL.Path, suffix) {
			return next.RoundTrip(r)
		}
		h := http.Header{"Content-Type": {"application/json"}}
		for i := 0; i+1 < len(headers); i += 2 {
			h.Set(headers[i], headers[i+1])
		}
		return &http.Response{
			StatusCode: http.StatusForbidden, Status: "403 Forbidden", Request: r, Header: h,
			Body: io.NopCloser(strings.NewReader(`{"message":` + strconv.Quote(message) + `}`)),
		}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = next })
}

// rv4-FR-F170 (API half of F186, F130): the github client now tells a refusal
// from a rate limit (*github.AccessRefusedError). Answered as a bad gateway the
// user is told GitHub failed and to try later, which cannot help: a blocked repo,
// a permission the App lacks or a suspended installation stays that way. The
// answer is the user's to act on at GitHub, so it is a 400 with its own code and
// GitHub's words; a rate limit keeps its 429.
func TestRvGhBindAGithubRefusalWhileLinkingIsNotABadGateway(t *testing.T) {
	f := newFakeGitHub(t)
	w := appServerWorld(t)
	name := uniqueRepo()
	f.addRepo(name, fakeRepo{private: true, install: 77})
	const why = "Repository access blocked"
	rvGhBindRefuse(t, "/installation", why)

	code, body := putRepo(t, w, name)
	expect(t, "link while GitHub refuses the App", code, body, http.StatusBadRequest)
	got := decodeAs[struct{ Code, Error string }](t, body)
	if got.Code != "github_access_refused" || !strings.Contains(got.Error, why) {
		t.Errorf("answer = %s, want code github_access_refused carrying GitHub's words %q", body, why)
	}
	if repoURL, _, err := w.st.ProjectRepoSync(context.Background(), w.owner.ID, w.project.ID); err != nil || repoURL != "" {
		t.Errorf("a refused link stored repo_url %q (err %v), want nothing", repoURL, err)
	}
}

func TestRvGhBindAGithubRefusalWhileSyncingIsNotABadGateway(t *testing.T) {
	f := newFakeGitHub(t)
	w := appServerWorld(t)
	name := uniqueRepo()
	f.addRepo(name, fakeRepo{private: true, install: 77, shas: []string{"c1"}})
	code, body := putRepo(t, w, name)
	expect(t, "link the private repo", code, body, http.StatusOK)
	const why = "Resource not accessible by integration"
	rvGhBindRefuse(t, "/commits", why)

	code, body = w.call(w.owner, http.MethodPost, syncPath(w), "")
	expect(t, "sync while GitHub refuses the commits read", code, body, http.StatusBadRequest)
	got := decodeAs[struct{ Code, Error string }](t, body)
	if got.Code != "github_access_refused" || !strings.Contains(got.Error, why) {
		t.Errorf("answer = %s, want code github_access_refused carrying GitHub's words %q", body, why)
	}
}

// The refusal is not a rate limit: the same 403 that carries a spent quota keeps
// its 429 and its retry hint.
func TestRvGhBindASpentQuotaIsStillARateLimitNotARefusal(t *testing.T) {
	f := newFakeGitHub(t)
	w := appServerWorld(t)
	name := uniqueRepo()
	f.addRepo(name, fakeRepo{private: true, install: 77})
	rvGhBindRefuse(t, "/installation", "API rate limit exceeded", "X-RateLimit-Remaining", "0")

	code, body := putRepo(t, w, name)
	expect(t, "link while the quota is spent", code, body, http.StatusTooManyRequests)
	if got := decodeAs[struct{ Code string }](t, body).Code; got != "github_rate_limited" {
		t.Errorf("code = %q, want github_rate_limited; body %s", got, body)
	}
}
