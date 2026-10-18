package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// rv4-SEC-06: ParseRepoURL let "." and ".." through as owner/repo (both are
// built from allowed characters), so "../.." parsed as a repo and the commits
// request path became /repos/../../commits with the server-wide GITHUB_TOKEN
// attached. It also found "github.com/" anywhere in the input, so
// "https://evil.example/x/github.com/a/b" parsed as a/b.

func TestParseRepoURLRejectsDotSegments(t *testing.T) {
	bad := []string{
		"../..",
		"./..",
		"../user",
		"owner/..",
		"owner/.",
		"owner/...",
		".../x",
		"github.com/../repo",
		"github.com/owner/..",
		"https://github.com/../..",
	}
	for _, raw := range bad {
		if o, r, err := ParseRepoURL(raw); err == nil {
			t.Errorf("ParseRepoURL(%q) = %q,%q with no error, want rejection: a segment made only of dots reshapes the request path", raw, o, r)
		}
	}
}

func TestParseRepoURLRequiresTheGithubHost(t *testing.T) {
	bad := []string{
		"https://evil.example/x/github.com/a/b", // the review's exhibit: parsed as a/b
		"http://evil.example/github.com/a/b",
		"evil.example/x/github.com/a/b", // no scheme, same trick
		"https://github.com.evil.example/a/b",
		"https://github.com@evil.example/a/b", // github.com is userinfo here
		"https://github.com:8443/a/b",
		"https://www.github.com/a/b",
		"https://notgithub.com/a/b",
		"ftp://evil.example/a/b",
	}
	for _, raw := range bad {
		if o, r, err := ParseRepoURL(raw); err == nil {
			t.Errorf("ParseRepoURL(%q) = %q,%q with no error, want rejection: the host must be exactly github.com", raw, o, r)
		}
	}
}

func TestParseRepoURLReadsEveryRealForm(t *testing.T) {
	cases := []struct{ raw, owner, repo string }{
		{"owner/repo", "owner", "repo"},
		{"owner/.github", "owner", ".github"}, // a real repo name that starts with a dot
		{"github.com/Owner/Repo.git/", "Owner", "Repo"},
		{"https://github.com/Owner/Repo", "Owner", "Repo"},
		{"https://github.com/Owner/Repo/tree/main", "Owner", "Repo"}, // a pasted browse URL
		{"GitHub.com/Owner/Repo", "Owner", "Repo"},                   // a host compares without case
		{"HTTPS://GITHUB.COM/Owner/Repo", "Owner", "Repo"},
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

// The token is the point. Whatever the parser accepts is interpolated into
// the commits URL, so a repo that is not a plain owner/repo pair must stop
// before any request is built: GitHub sees nothing, and neither does anyone
// who could read what it was sent.
func TestFetchCommitsSinceSendsNothingForAHostileRepo(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.URL.Path+" Authorization="+r.Header.Get("Authorization"))
		mu.Unlock()
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = "https://api.github.com" }()

	for _, raw := range []string{"../..", "./..", "../user", "https://evil.example/x/github.com/a/b"} {
		if _, err := FetchCommitsSince(context.Background(), "not-a-real-token", raw, "", 3, 5); err == nil {
			t.Errorf("FetchCommitsSince(%q) = no error, want the repo refused", raw)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 0 {
		t.Errorf("GitHub received %d request(s) for hostile repos, want none: %q", len(seen), seen)
	}
}

// The stored spelling is the one the webhook asks for, built from GitHub's own
// full_name: same scheme, same host, same case. If the two ever drift apart the
// lookup silently matches nothing.
func TestCanonicalRepoURLIsWhatTheWebhookLookupMatches(t *testing.T) {
	if got, want := CanonicalRepoURL("Owner", "Repo"), "https://github.com/Owner/Repo"; got != want {
		t.Errorf("CanonicalRepoURL = %q, want %q", got, want)
	}
	owner, repo, err := ParseRepoURL("github.com/Owner/Repo.git/")
	if err != nil {
		t.Fatalf("ParseRepoURL: %v", err)
	}
	if got, want := CanonicalRepoURL(owner, repo), RepoURLFor("Owner/Repo"); got != want {
		t.Errorf("canonical %q differs from the webhook's %q", got, want)
	}
}
