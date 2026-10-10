package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// SEC-06: owner/repo segments and the ref become parts of outbound GitHub
// API URLs. Anything outside GitHub's name charset in a segment — '?', '#',
// '%', spaces — can reshape the request (inject query parameters, start a
// fragment), so ParseRepoURL must reject it instead of passing it through.

func TestParseRepoURLRejectsURLShapingCharacters(t *testing.T) {
	bad := []string{
		"github.com/foo/bar?per_page=1", // the finding's scenario: repo parses to "bar?per_page=1"
		"foo/bar#section",
		"foo/ba%20r",
		"fo o/bar",
		"foo/bar\tx",
		"github.com/fo?o/bar",
	}
	for _, raw := range bad {
		if _, _, err := ParseRepoURL(raw); err == nil {
			t.Errorf("ParseRepoURL(%q) = no error, want rejection: the segment contains URL-shaping characters", raw)
		}
	}
}

func TestParseRepoURLAcceptsRealNames(t *testing.T) {
	cases := []struct {
		raw   string
		owner string
		repo  string
	}{
		{"owner/repo", "owner", "repo"},
		{"https://github.com/frank/inside-out/", "frank", "inside-out"},
		{"github.com/a.b/_c-d.git", "a.b", "_c-d"},
	}
	for _, c := range cases {
		owner, repo, err := ParseRepoURL(c.raw)
		if err != nil {
			t.Errorf("ParseRepoURL(%q): %v", c.raw, err)
			continue
		}
		if owner != c.owner || repo != c.repo {
			t.Errorf("ParseRepoURL(%q) = %q,%q; want %q,%q", c.raw, owner, repo, c.owner, c.repo)
		}
	}
}

// FetchGuideFile must escape what it interpolates: a '#' in the repo would
// turn the rest of the URL into a fragment, and a '&' in the ref would
// inject extra query parameters. The test server records what actually
// arrived, so an unescaped build fails on the captured request.
func TestFetchGuideFileEscapesURLParts(t *testing.T) {
	var got struct {
		path     string
		ref      string
		nQueries int
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path = r.URL.EscapedPath()
		got.ref = r.URL.Query().Get("ref")
		got.nQueries = len(r.URL.Query())
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("guide"))
	}))
	defer srv.Close()
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = "https://api.github.com" }()

	if _, err := FetchGuideFile(context.Background(), "", "own", "re#po", "main&tracked=1"); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if want := "/repos/own/re%23po/contents/insideout.yaml"; got.path != want {
		t.Errorf("path = %q, want %q (the '#' must not start a fragment)", got.path, want)
	}
	if want := "main&tracked=1"; got.ref != want {
		t.Errorf("ref = %q, want %q (the '&' must not split the query)", got.ref, want)
	}
	if got.nQueries != 1 {
		t.Errorf("query parameters = %d, want exactly 1 (ref)", got.nQueries)
	}

	// Clean values must keep producing the plain URL.
	if _, err := FetchGuideFile(context.Background(), "", "own", "repo", "main"); err != nil {
		t.Fatalf("clean fetch: %v", err)
	}
	if want := "/repos/own/repo/contents/insideout.yaml"; !reflect.DeepEqual(got.path, want) || got.ref != "main" {
		t.Errorf("clean values: path=%q ref=%q, want path=%q ref=main", got.path, got.ref, want)
	}
}
