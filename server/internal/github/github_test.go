package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// pages is the fake commit history the test server walks, newest first, three
// per page. FetchCommitsSince must be able to reach any SHA here regardless of
// which page it lands on — that's the F5 window-loss regression.
var pages = [][]string{
	{"A", "B", "C"},
	{"D", "E", "F"},
	{"G", "H"},
}

// newPagedServer serves /repos/{o}/{r}/commits with rel="next" Link headers
// chaining page → page+1, over real HTTP (no mocks).
func newPagedServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			fmt.Sscanf(p, "%d", &page)
		}
		idx := page - 1
		if idx < 0 || idx >= len(pages) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if idx+1 < len(pages) {
			next := fmt.Sprintf("<http://%s%s?page=%d>; rel=\"next\"", r.Host, r.URL.Path, page+1)
			last := fmt.Sprintf("<http://%s%s?page=%d>; rel=\"last\"", r.Host, r.URL.Path, len(pages))
			w.Header().Set("Link", next+", "+last)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "[")
		for i, sha := range pages[idx] {
			if i > 0 {
				fmt.Fprint(w, ",")
			}
			fmt.Fprintf(w, `{"sha":%q,"commit":{"message":%q,"author":{"name":"dev","date":"2026-07-01T00:00:00Z"}}}`, sha, "msg "+sha)
		}
		fmt.Fprint(w, "]")
	}))
}

func shas(cs []Commit) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.SHA)
	}
	return out
}

func TestFetchCommitsSince_CursorOnFirstPage(t *testing.T) {
	srv := newPagedServer(t)
	defer srv.Close()
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = "https://api.github.com" }()

	got, err := FetchCommitsSince(context.Background(), "owner/repo", "C", 3, 5)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	// Cursor C is on page 1; only the two newer commits come back, cursor excluded.
	if want := []string{"A", "B"}; !reflect.DeepEqual(shas(got), want) {
		t.Fatalf("got %v, want %v", shas(got), want)
	}
}

func TestFetchCommitsSince_CrossesPages(t *testing.T) {
	srv := newPagedServer(t)
	defer srv.Close()
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = "https://api.github.com" }()

	got, err := FetchCommitsSince(context.Background(), "owner/repo", "E", 3, 5)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	// Cursor E is on page 2 — a flat first-page fetch would have missed D..H
	// entirely and jumped the cursor forward. Pagination must walk back to E.
	if want := []string{"A", "B", "C", "D"}; !reflect.DeepEqual(shas(got), want) {
		t.Fatalf("got %v, want %v", shas(got), want)
	}
}

func TestFetchCommitsSince_FirstSyncOnePage(t *testing.T) {
	srv := newPagedServer(t)
	defer srv.Close()
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = "https://api.github.com" }()

	// Empty cursor (first-ever sync): just the first page, no history walk.
	got, err := FetchCommitsSince(context.Background(), "owner/repo", "", 3, 5)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if want := []string{"A", "B", "C"}; !reflect.DeepEqual(shas(got), want) {
		t.Fatalf("got %v, want %v", shas(got), want)
	}
}

func TestFetchCommitsSince_CursorNeverFoundBounded(t *testing.T) {
	srv := newPagedServer(t)
	defer srv.Close()
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = "https://api.github.com" }()

	// A cursor that no longer exists (force-push) must not loop forever: it
	// stops at maxPages and returns what it walked.
	got, err := FetchCommitsSince(context.Background(), "owner/repo", "ZZZ", 3, 3)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if want := []string{"A", "B", "C", "D", "E", "F", "G", "H"}; !reflect.DeepEqual(shas(got), want) {
		t.Fatalf("got %v, want %v", shas(got), want)
	}
}

func TestNextLink(t *testing.T) {
	cases := map[string]string{
		`<https://api.github.com/x?page=2>; rel="next", <https://api.github.com/x?page=9>; rel="last"`: "https://api.github.com/x?page=2",
		`<https://api.github.com/x?page=9>; rel="last"`: "",
		``:                 "",
		`<u>; rel="prev"`: "",
	}
	for header, want := range cases {
		if got := nextLink(header); got != want {
			t.Fatalf("nextLink(%q) = %q, want %q", header, got, want)
		}
	}
}

// statusServer answers every commits request with a fixed status (and optional
// headers), for exercising the non-200 branches (F20).
func statusServer(status int, hdr map[string]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range hdr {
			w.Header().Set(k, v)
		}
		w.WriteHeader(status)
	}))
}

func TestFetchCommitsSince_RepoNotFound(t *testing.T) {
	srv := statusServer(http.StatusNotFound, nil)
	defer srv.Close()
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = "https://api.github.com" }()

	_, err := FetchCommitsSince(context.Background(), "owner/repo", "", 3, 5)
	if !errors.Is(err, ErrRepoNotFound) {
		t.Fatalf("want ErrRepoNotFound, got %v", err)
	}
}

func TestFetchCommitsSince_RateLimited(t *testing.T) {
	srv := statusServer(http.StatusForbidden, map[string]string{"Retry-After": "42"})
	defer srv.Close()
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = "https://api.github.com" }()

	_, err := FetchCommitsSince(context.Background(), "owner/repo", "", 3, 5)
	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("want *RateLimitError, got %v", err)
	}
	if rle.RetryAfterSeconds != 42 {
		t.Fatalf("RetryAfterSeconds = %d, want 42", rle.RetryAfterSeconds)
	}
}

func TestFetchCommitsSince_SubjectTruncatedByRune(t *testing.T) {
	// A 300-rune (900-byte) single-line subject must come back capped at 280
	// runes — proving the cap counts runes, not bytes (F19).
	long := strings.Repeat("世", 300)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `[{"sha":"A","commit":{"message":%q,"author":{"name":"dev","date":"2026-07-01T00:00:00Z"}}}]`, long)
	}))
	defer srv.Close()
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = "https://api.github.com" }()

	got, err := FetchCommitsSince(context.Background(), "owner/repo", "", 3, 5)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if n := len([]rune(got[0].Message)); n != maxSubjectRunes {
		t.Fatalf("subject len = %d runes, want %d", n, maxSubjectRunes)
	}
}
