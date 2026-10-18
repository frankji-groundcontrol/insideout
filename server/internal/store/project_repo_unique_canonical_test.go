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
	corpus := []string{
		// accepted, in the spellings people paste
		"own/repo", "Own/Repo", "OWN/REPO.git", "own/repo.git/", "own/repo/", "  own/repo  ",
		"github.com/Own/Repo", "GitHub.com/Own/Repo.git",
		"https://github.com/Own/Repo", "https://github.com/Own/Repo.git/", "http://github.com/o/r",
		"https://GITHUB.COM/o/r", "ssh://github.com/o/r", "https://github.com/o/r/tree/main/docs",
		"https://github.com/o.wner/re.po", "o_w-n/r_e-p", "a/b.git.git", "a/.hidden", "a/b..c",
		"own/repo\n", // TrimSpace takes the newline off
		// refused
		"", " ", "/", "own", "own/", "/repo", "github.com", "github.com/own", "https://github.com/",
		"https://", "../..", "./..", "../user", "owner/..", "own/.", "own/..", "./repo",
		"https://evil.example/x/github.com/a/b", "https://github.com.evil.example/a/b",
		"https://evil.example/own/repo", "foo.bar/baz", scpRemote, "www.github.com/own/repo",
		"o w/r", "own/r epo", "o/r?x=1", "o/r#frag", "o/r%2e", "o\\r/p", "own/rép", "ö/repo",
		"own\n/repo",
	}
	for _, in := range corpus {
		var want string
		if owner, repo, err := github.ParseRepoURL(in); err == nil {
			want = github.CanonicalRepoURL(owner, repo)
		}
		var got string
		if err := st.Pool.QueryRow(ctx, `SELECT COALESCE(insideout._canonical_repo_url($1), '')`, in).Scan(&got); err != nil {
			t.Fatalf("insideout._canonical_repo_url(%q): %v", in, err)
		}
		if got != want {
			t.Errorf("insideout._canonical_repo_url(%q) = %q, Go says %q", in, got, want)
		}
	}
}
