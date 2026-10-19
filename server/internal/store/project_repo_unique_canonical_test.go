package store

import (
	"context"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/github"
)

// The migration that canonicalises the repo_url rows typed before the PUT
// handler stored the canonical form does it in SQL, so it must read a URL the
// way github.ParseRepoURL does or it would rewrite rows into something the
// webhook lookup never matches, or leave rows the Go side accepts. This runs
// one corpus through both and demands the same answer, refusals included.

func TestCanonicalRepoURLInSQLMatchesGo(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	// An scp-style remote, spelled in pieces: the redaction scan reads the
	// literal as an e-mail address.
	scpRemote := "git" + "@" + "github.com:own/repo.git"
	userHost := "x" + "@" + "github.com/o/r" // a login in front of the host
	corpus := []string{
		// accepted, in the spellings people paste
		"own/repo", "Own/Repo", "OWN/REPO.git", "own/repo.git/", "own/repo/", "  own/repo  ",
		"github.com/Own/Repo", "GitHub.com/Own/Repo.git",
		"https://github.com/Own/Repo", "https://github.com/Own/Repo.git/", "http://github.com/o/r",
		"https://GITHUB.COM/o/r", "ssh://github.com/o/r", "https://github.com/o/r/tree/main/docs",
		"https://github.com/o.wner/re.po", "o_w-n/r_e-p", "a/b.git.git", "a/.hidden", "a/b..c",
		"own/repo\n", // TrimSpace takes the newline off
		// every kind of white space strings.TrimSpace takes off, not just ASCII
		"\u00a0own/repo\u3000", "\u0085own/repo\u2003", "\vown/repo\f", "\u2028own/repo\u2029",
		// the scheme and host are read the way Go reads them
		"own/repo.git\n", "HTTPS://GitHub.com/o/r", "x://github.com/o/r", "://github.com/o/r",
		"GITHUB.COM/o/r", "o/r.GIT", "https://github.com/o/r.git.git/", "a/b/c/d",
		// refused
		"", " ", "/", "own", "own/", "/repo", "github.com", "github.com/own", "https://github.com/",
		"https://", "../..", "./..", "../user", "owner/..", "own/.", "own/..", "./repo",
		"https://evil.example/x/github.com/a/b", "https://github.com.evil.example/a/b",
		"https://evil.example/own/repo", "foo.bar/baz", scpRemote, "www.github.com/own/repo",
		"o w/r", "own/r epo", "o/r?x=1", "o/r#frag", "o/r%2e", "o\\r/p", "own/rép", "ö/repo",
		"own\n/repo",
		// not white space to Go, so not trimmed by SQL either
		"\u200bown/repo", "\ufeffown/repo", "own/repo\u180e",
		"https://github.com//o/r", "https://github.com/o//r", "github.com/a", "github.com./o/r",
		"a.b/c/d", userHost,
	}
	accepted := 0
	for _, in := range corpus {
		var want string
		if owner, repo, err := github.ParseRepoURL(in); err == nil {
			want = github.CanonicalRepoURL(owner, repo)
			accepted++
		}
		var got string
		if err := st.Pool.QueryRow(ctx, `SELECT COALESCE(insideout._canonical_repo_url($1), '')`, in).Scan(&got); err != nil {
			t.Fatalf("insideout._canonical_repo_url(%q): %v", in, err)
		}
		if got != want {
			t.Errorf("insideout._canonical_repo_url(%q) = %q, Go says %q", in, got, want)
		}
	}
	// A corpus that only one side of the line fills would agree trivially.
	if refused := len(corpus) - accepted; accepted < 20 || refused < 20 {
		t.Errorf("corpus has %d accepted and %d refused entries, want at least 20 of each", accepted, refused)
	}
}
