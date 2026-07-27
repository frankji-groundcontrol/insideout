// Package github fetches recent commits for a linked repo so a project's
// progress timeline can sync real development activity. Public repos need no
// credentials (GitHub's unauthenticated rate limit is plenty for a manual
// sync); set GITHUB_TOKEN to raise it. See docs/plans/2026-07-22-idea-to-reality.md.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Commit is the subset of a GitHub commit the progress timeline records.
type Commit struct {
	SHA     string
	Message string
	Author  string
	Date    time.Time
}

// ParseRepoURL extracts owner/repo from a github.com URL, a bare "owner/repo",
// with or without scheme, .git suffix, or trailing slash.
func ParseRepoURL(raw string) (owner, repo string, err error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimSuffix(s, ".git")
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.Index(s, "github.com/"); i >= 0 {
		s = s[i+len("github.com/"):]
	}
	parts := strings.Split(s, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("github: cannot parse repo %q — want owner/repo or a github.com URL", raw)
	}
	return parts[0], parts[1], nil
}

// apiBaseURL is the GitHub API root. A package var (not a const) so tests can
// point it at an httptest server and exercise real pagination/cursor logic.
var apiBaseURL = "https://api.github.com"

// maxCommitPageBytes caps a single commits-page response body so a hostile or
// runaway upstream can't make us buffer an unbounded payload (F19). A real page
// of 100 commits is a few hundred KB; 5 MiB leaves wide margin.
const maxCommitPageBytes = 5 << 20

// maxSubjectRunes caps a recorded commit subject so one pathological message
// can't bloat the timeline (F19). GitHub's own UI truncates around here.
const maxSubjectRunes = 280

// ErrRepoNotFound means GitHub answered 404 — a private repo we can't see or a
// bad owner/repo name. Callers map it to a 404, distinct from an upstream 5xx.
var ErrRepoNotFound = errors.New("github: repository not found")

// RateLimitError means GitHub answered 403/429 — the (unauthenticated) rate
// limit is exhausted. RetryAfterSeconds carries the Retry-After hint so the API
// layer can tell the user when to try again instead of a bare 502 (F20).
type RateLimitError struct{ RetryAfterSeconds int }

func (e *RateLimitError) Error() string { return "github: rate limited" }

// retryAfterSeconds reads GitHub's Retry-After hint (whole seconds), defaulting
// to 60 when absent or unparseable.
func retryAfterSeconds(h http.Header) int {
	if v := h.Get("Retry-After"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			return n
		}
	}
	return 60
}

// FetchCommitsSince returns the commits newer than sinceSHA, newest first,
// following GitHub's rel="next" pagination until sinceSHA is found. This is
// what keeps a busy repo honest: a flat "newest 20" fetch silently drops every
// commit past the first page once a project falls more than a page behind, so
// the sync cursor jumped forward over work that was never recorded.
//
// sinceSHA == "" (a first-ever sync) returns just the first page — we have no
// cursor to walk back to, and pulling a repo's whole history into the timeline
// would be noise. If the cursor is never found within maxPages (e.g. a
// force-push rewrote history and the old SHA no longer exists) we stop at the
// page cap and return what we have; the cursor then advances past the gap.
//
// ponytail: maxPages bounds the walk (default 5 pages × perPage). A repo that
// somehow lands more than that between two manual syncs loses the overflow —
// acceptable for a human-triggered sync; raise maxPages or background the sync
// if it ever matters.
func FetchCommitsSince(ctx context.Context, repoURL, sinceSHA string, perPage, maxPages int) ([]Commit, error) {
	owner, repo, err := ParseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}
	if perPage <= 0 || perPage > 100 {
		perPage = 20
	}
	if maxPages <= 0 {
		maxPages = 5
	}

	url := fmt.Sprintf("%s/repos/%s/%s/commits?per_page=%d", apiBaseURL, owner, repo, perPage)
	var out []Commit
	for page := 0; page < maxPages && url != ""; page++ {
		commits, next, err := fetchCommitPage(ctx, url)
		if err != nil {
			return nil, err
		}
		for _, c := range commits {
			if sinceSHA != "" && c.SHA == sinceSHA {
				return out, nil // cursor reached: everything accumulated is newer
			}
			out = append(out, c)
		}
		if sinceSHA == "" {
			return out, nil // first sync: one page is plenty
		}
		url = next
	}
	return out, nil
}

// fetchCommitPage GETs one commits page and returns its commits plus the
// rel="next" URL ("" on the last page).
func fetchCommitPage(ctx context.Context, url string) ([]Commit, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("github: build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "insideout-app") // GitHub rejects UA-less requests
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("github: fetch commits: %w", err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, "", ErrRepoNotFound
	case resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests:
		return nil, "", &RateLimitError{RetryAfterSeconds: retryAfterSeconds(resp.Header)}
	case resp.StatusCode != http.StatusOK:
		return nil, "", fmt.Errorf("github: commits returned status %d", resp.StatusCode)
	}

	var raw []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name string    `json:"name"`
				Date time.Time `json:"date"`
			} `json:"author"`
		} `json:"commit"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxCommitPageBytes)).Decode(&raw); err != nil {
		return nil, "", fmt.Errorf("github: decode commits: %w", err)
	}

	out := make([]Commit, 0, len(raw))
	for _, c := range raw {
		msg := c.Commit.Message
		if i := strings.IndexByte(msg, '\n'); i >= 0 {
			msg = msg[:i] // subject line only
		}
		msg = strings.TrimSpace(msg)
		if r := []rune(msg); len(r) > maxSubjectRunes {
			msg = string(r[:maxSubjectRunes])
		}
		out = append(out, Commit{SHA: c.SHA, Message: msg, Author: c.Commit.Author.Name, Date: c.Commit.Author.Date})
	}
	return out, nextLink(resp.Header.Get("Link")), nil
}

// nextLink pulls the rel="next" URL out of a GitHub Link header, or "".
func nextLink(header string) string {
	for _, part := range strings.Split(header, ",") {
		segs := strings.Split(part, ";")
		if len(segs) < 2 || !strings.Contains(segs[1], `rel="next"`) {
			continue
		}
		u := strings.TrimSpace(segs[0])
		if len(u) >= 2 && u[0] == '<' && u[len(u)-1] == '>' {
			return u[1 : len(u)-1]
		}
	}
	return ""
}
