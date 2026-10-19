package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

// rv4-FR-F177 and rv4-FR-F185: without the server-wide token every link to a
// public or uninstalled repo spends one request of GitHub's anonymous quota,
// 60 an hour for the whole server address. The question "is this repo public?"
// has the same answer a minute later, so it is asked once per repo and
// remembered, and an answer GitHub did not give (a limit, a refusal, an
// outage) is never remembered.

var rvGhClientRepoSeq atomic.Int64

// rvGhClientRepo is a repo name no other test has asked about, so what one test
// leaves remembered can never be another's answer.
func rvGhClientRepo() string { return fmt.Sprintf("cache-%d", rvGhClientRepoSeq.Add(1)) }

// rvGhClientGitHub answers every repo under "own", in any case: the anonymous
// question with status, and the App's installation lookup with a 404. It counts
// both kinds of question.
type rvGhClientGitHub struct {
	status  atomic.Int32 // the answer to GET /repos/own/<name>
	public  atomic.Int32 // times the anonymous question was asked
	install atomic.Int32 // times the App was asked which installation covers a repo
}

func rvGhClientNewGitHub(t *testing.T, status int) *rvGhClientGitHub {
	t.Helper()
	g := &rvGhClientGitHub{}
	g.status.Store(int32(status))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.ToLower(r.URL.Path)
		switch {
		case strings.HasSuffix(path, "/installation"):
			g.install.Add(1)
			http.NotFound(w, r)
		case strings.HasPrefix(path, "/repos/own/"):
			g.public.Add(1)
			w.WriteHeader(int(g.status.Load()))
			fmt.Fprint(w, `{"message":"stub"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	point(t, srv)
	return g
}

func TestRvGhClientRepoIsPublicIsAskedOncePerRepo(t *testing.T) {
	for _, c := range []struct {
		name   string
		status int
		want   bool
	}{{"public", http.StatusOK, true}, {"private or missing", http.StatusNotFound, false}} {
		t.Run(c.name, func(t *testing.T) {
			g := rvGhClientNewGitHub(t, c.status)
			repo := rvGhClientRepo()
			for range 3 {
				if got, err := RepoIsPublic(context.Background(), "own", repo); err != nil || got != c.want {
					t.Fatalf("RepoIsPublic = %v, %v; want %v", got, err, c.want)
				}
			}
			if n := g.public.Load(); n != 1 {
				t.Errorf("GitHub was asked %d times about one repo, want 1", n)
			}
		})
	}
}

// The link route asks RepoVisible. The anonymous question behind it is asked
// once; the App's own lookup is asked every time, because an installation made a
// minute ago has to count at once.
func TestRvGhClientRepoVisibleAsksThePublicQuestionOnce(t *testing.T) {
	g := rvGhClientNewGitHub(t, http.StatusOK)
	app := testApp(t)
	repo := rvGhClientRepo()
	for range 3 {
		if got, err := app.RepoVisible(context.Background(), "own", repo); err != nil || !got {
			t.Fatalf("RepoVisible = %v, %v; want true", got, err)
		}
	}
	if n := g.public.Load(); n != 1 {
		t.Errorf("anonymous questions = %d, want 1", n)
	}
	if n := g.install.Load(); n != 3 {
		t.Errorf("installation lookups = %d, want 3: they are not remembered", n)
	}
}

// GitHub reads owner and repo names without regard to case; two spellings are
// one repo. A different repo is a different question.
func TestRvGhClientAnswersAreKeptPerRepoWhateverTheCase(t *testing.T) {
	g := rvGhClientNewGitHub(t, http.StatusOK)
	repo := rvGhClientRepo()
	ctx := context.Background()
	for _, ask := range [][2]string{{"own", repo}, {"OWN", strings.ToUpper(repo)}, {"Own", repo}} {
		if got, err := RepoIsPublic(ctx, ask[0], ask[1]); err != nil || !got {
			t.Fatalf("RepoIsPublic(%q, %q) = %v, %v; want true", ask[0], ask[1], got, err)
		}
	}
	if n := g.public.Load(); n != 1 {
		t.Errorf("one repo in three spellings: asked %d times, want 1", n)
	}
	if got, err := RepoIsPublic(ctx, "own", rvGhClientRepo()); err != nil || !got {
		t.Fatalf("another repo: %v, %v; want true", got, err)
	}
	if n := g.public.Load(); n != 2 {
		t.Errorf("a second repo: asked %d times in all, want 2", n)
	}
}

// Only an answer is remembered. A limit, a refusal or an outage says nothing
// about the repo, so the next link asks again and sees the real answer.
func TestRvGhClientAnErrorIsNotRemembered(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			g := rvGhClientNewGitHub(t, status)
			repo := rvGhClientRepo()
			if got, err := RepoIsPublic(context.Background(), "own", repo); err == nil || got {
				t.Fatalf("while GitHub says %d: %v, %v; want an error", status, got, err)
			}
			g.status.Store(http.StatusOK)
			if got, err := RepoIsPublic(context.Background(), "own", repo); err != nil || !got {
				t.Errorf("after it recovers: %v, %v; want true", got, err)
			}
			if n := g.public.Load(); n != 2 {
				t.Errorf("asked %d times, want 2: the failure must not be an answer", n)
			}
		})
	}
}
