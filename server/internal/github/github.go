// Package github reads a linked repo for a project: recent commits for the
// progress timeline (commits.go, the URL rules in repo_url.go, the public-repo
// probe in visibility.go) and the insideout.yaml guide plus the GitHub
// App credentials that make a private repo readable (appauth.go). A public repo
// reads without a credential, but GitHub's anonymous rate limit is 60 requests
// an hour per server address, so a deployment that holds the App reads every
// repo it is installed on through that repo's installation token. There is no
// server-wide token: it would be one credential any member could aim at any
// repo it can name (rv4-FEAT-04). See docs/plans/2026-07-22-idea-to-reality.md.
package github

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// apiBaseURL is the GitHub API root. A package var (not a const) so tests can
// point it at an httptest server and exercise real pagination/cursor logic.
var apiBaseURL = "https://api.github.com"

// ErrRepoNotFound means GitHub answered 404 — a private repo we can't see or a
// bad owner/repo name. Callers map it to a 404, distinct from an upstream 5xx.
var ErrRepoNotFound = errors.New("github: repository not found")

// RateLimitError means GitHub said a rate limit is spent: a 429, or a 403 with
// Retry-After or X-RateLimit-Remaining 0 (limitOrRefusal). RetryAfterSeconds is
// when to ask again, so the API layer can say so instead of a bare 502 (F20).
type RateLimitError struct{ RetryAfterSeconds int }

func (e *RateLimitError) Error() string { return "github: rate limited" }

// AccessRefusedError means GitHub answered 403 with no sign of a rate limit: a
// blocked repository, a permission the App was not granted ("Resource not
// accessible by integration") or a suspended installation. Waiting does not
// help. Message is GitHub's own explanation, cleaned and capped.
type AccessRefusedError struct{ Message string }

func (e *AccessRefusedError) Error() string { return "github: access refused: " + e.Message }

// retryAfterSeconds is when to ask GitHub again: its Retry-After (whole seconds);
// else, for a spent quota, the time until X-RateLimit-Reset (epoch seconds), at
// least one; else 60 (a secondary limit with quota left has no reset to wait for).
func retryAfterSeconds(h http.Header, now time.Time) int {
	if n, err := strconv.Atoi(strings.TrimSpace(h.Get("Retry-After"))); err == nil && n > 0 {
		return n
	}
	if strings.TrimSpace(h.Get("X-RateLimit-Remaining")) == "0" {
		if reset, err := strconv.ParseInt(strings.TrimSpace(h.Get("X-RateLimit-Reset")), 10, 64); err == nil {
			return int(max(reset-now.Unix(), 1))
		}
	}
	return 60
}
