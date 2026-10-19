package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// rv4-FR-F170, rv4-FR-F186, rv4-FR-F130 and the retry half of rv4-FR-F177:
// GitHub answers 403 for more than a rate limit. A repository it has blocked,
// an App that was not granted a permission ("Resource not accessible by
// integration") and a suspended installation all say 403 too, and every one of
// them was reported as "rate limit hit, try again shortly" with a 60 second
// hint. A 403 is a rate limit only when GitHub says so, with Retry-After or a
// quota that has nothing left; a 429 always is. The hint comes from those
// headers. Every place the package asks GitHub a question is held to the same
// rule, so the table below runs each case against each of them.

const rvGhClientRefusal = "Resource not accessible by integration"

// rvGhClientQuestion is one place the package asks GitHub something. path is
// the request its answer turns on: the stub answers that path as told and 404s
// every other, so a question that asks twice (RepoVisible) reaches the reply on
// its own leg.
type rvGhClientQuestion struct {
	name string
	path string
	ask  func(t *testing.T) error
}

func rvGhClientQuestions() []rvGhClientQuestion {
	ctx := context.Background()
	return []rvGhClientQuestion{
		{"RepoIsPublic", "/repos/own/repo", func(t *testing.T) error {
			_, err := RepoIsPublic(ctx, "own", "repo")
			return err
		}},
		{"InstallationForRepo", "/repos/own/repo/installation", func(t *testing.T) error {
			_, err := testApp(t).InstallationForRepo(ctx, "own", "repo")
			return err
		}},
		{"RepoVisible, installation leg", "/repos/own/repo/installation", func(t *testing.T) error {
			_, err := testApp(t).RepoVisible(ctx, "own", "repo")
			return err
		}},
		{"RepoVisible, public leg", "/repos/own/repo", func(t *testing.T) error {
			_, err := testApp(t).RepoVisible(ctx, "own", "repo")
			return err
		}},
		{"FetchCommitsSince", "/repos/own/repo/commits", func(t *testing.T) error {
			_, err := FetchCommitsSince(ctx, "", "own/repo", "", 3, 5)
			return err
		}},
	}
}

// rvGhClientServe aims the package at a server that answers path with status,
// headers and body, and 404s everything else.
func rvGhClientServe(t *testing.T, path string, status int, headers map[string]string, body string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	point(t, srv)
}

// rvGhClientReset is X-RateLimit-Reset for a quota that refills in d.
func rvGhClientReset(d time.Duration) string {
	return strconv.FormatInt(time.Now().Add(d).Unix(), 10)
}

func rvGhClientCases() []struct {
	name    string
	status  int
	headers map[string]string
	limit   bool // true: a rate limit, false: a refusal
	lo, hi  int  // the retry hint a rate limit carries
} {
	h := func(kv ...string) map[string]string {
		m := map[string]string{}
		for i := 0; i < len(kv); i += 2 {
			m[kv[i]] = kv[i+1]
		}
		return m
	}
	return []struct {
		name    string
		status  int
		headers map[string]string
		limit   bool
		lo, hi  int
	}{
		// Not rate limits: nothing says the quota or the request rate is the problem.
		{"403 without headers", 403, nil, false, 0, 0},
		{"403 with quota left", 403, h("X-RateLimit-Remaining", "4999", "X-RateLimit-Reset", rvGhClientReset(50*time.Minute)), false, 0, 0},
		// Rate limits, and when to come back.
		{"403 quota spent, refills in 3000 s", 403, h("X-RateLimit-Remaining", "0", "X-RateLimit-Reset", rvGhClientReset(3000*time.Second)), true, 2990, 3001},
		{"403 quota spent, refills in 30 s", 403, h("X-RateLimit-Remaining", "0", "X-RateLimit-Reset", rvGhClientReset(30*time.Second)), true, 25, 31},
		{"403 quota spent, refill time already past", 403, h("X-RateLimit-Remaining", "0", "X-RateLimit-Reset", rvGhClientReset(-5*time.Minute)), true, 1, 5},
		{"403 quota spent, no refill time", 403, h("X-RateLimit-Remaining", "0"), true, 60, 60},
		{"403 with Retry-After", 403, h("Retry-After", "42"), true, 42, 42},
		{"403 with an unreadable Retry-After", 403, h("Retry-After", "soon"), true, 60, 60},
		{"Retry-After beats the refill time", 403, h("Retry-After", "42", "X-RateLimit-Remaining", "0", "X-RateLimit-Reset", rvGhClientReset(3000*time.Second)), true, 42, 42},
		{"429 without headers", 429, nil, true, 60, 60},
		{"429 with Retry-After", 429, h("Retry-After", "7"), true, 7, 7},
		{"429 quota spent, refills in 120 s", 429, h("X-RateLimit-Remaining", "0", "X-RateLimit-Reset", rvGhClientReset(120*time.Second)), true, 115, 121},
		// A secondary limit with quota left: the refill time is not when to retry.
		{"429 with quota left", 429, h("X-RateLimit-Remaining", "4000", "X-RateLimit-Reset", rvGhClientReset(3000*time.Second)), true, 60, 60},
	}
}

func TestRvGhClient403And429AreToldApartAtEveryCallSite(t *testing.T) {
	body := `{"message":"` + rvGhClientRefusal + `","documentation_url":"https://docs.github.com/rest"}`
	for _, q := range rvGhClientQuestions() {
		for _, c := range rvGhClientCases() {
			t.Run(q.name+"/"+c.name, func(t *testing.T) {
				rvGhClientServe(t, q.path, c.status, c.headers, body)
				err := q.ask(t)
				var rle *RateLimitError
				isLimit := errors.As(err, &rle)
				switch {
				case c.limit && !isLimit:
					t.Fatalf("err = %v, want a rate limit", err)
				case c.limit && (rle.RetryAfterSeconds < c.lo || rle.RetryAfterSeconds > c.hi):
					t.Errorf("retry hint = %d s, want %d to %d s", rle.RetryAfterSeconds, c.lo, c.hi)
				case !c.limit && isLimit:
					t.Errorf("reported as a rate limit (retry in %d s), want a refusal that keeps GitHub's words", rle.RetryAfterSeconds)
				case !c.limit && (err == nil || !strings.Contains(err.Error(), rvGhClientRefusal)):
					t.Errorf("err = %v, want GitHub's message %q to reach the caller", err, rvGhClientRefusal)
				}
			})
		}
	}
}
