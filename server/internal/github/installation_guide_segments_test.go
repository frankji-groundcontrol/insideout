package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// Carry-forward from rv4-SEC-06: ParseRepoURL refuses a dot-only or empty
// owner/repo, but FetchGuideFile builds its own URL from names that arrive in a
// delivery payload, and PathEscape leaves "." and ".." alone while an empty
// segment collapses into "//". Either reshapes the path of a request that
// carries the installation token, so they are refused before any request.
// Names with other odd characters stay escaped and sent
// (TestFetchGuideFileEscapesURLParts keeps that contract).
func TestFetchGuideFileRefusesDotOnlyAndEmptyNamesWithoutAsking(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("guide"))
	}))
	defer srv.Close()
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = "https://api.github.com" }()

	for _, tc := range []struct{ owner, repo string }{
		{"..", "repo"}, {"own", ".."}, {".", "repo"}, {"own", "."},
		{"...", "repo"}, {"", "repo"}, {"own", ""}, {"", ""},
	} {
		_, err := FetchGuideFile(context.Background(), "ghs_token", tc.owner, tc.repo, "main")
		if err == nil || errors.Is(err, ErrGuideNotFound) {
			t.Errorf("FetchGuideFile(%q, %q) err = %v, want a refusal that is not \"guide not found\"", tc.owner, tc.repo, err)
		}
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("%d request(s) went out for names that cannot be repos, want none", n)
	}
}
