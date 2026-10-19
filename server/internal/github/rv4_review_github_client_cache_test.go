package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

// How long an answer is kept is read off the map itself: a clock that could be
// wound would be a second moving part to test the first.
func TestRvGhClientAnswersAreKeptForTheirTime(t *testing.T) {
	for _, c := range []struct {
		name   string
		status int
		ttl    time.Duration
	}{{"public", http.StatusOK, publicTTL}, {"not public", http.StatusNotFound, notPublicTTL}} {
		t.Run(c.name, func(t *testing.T) {
			rvGhClientNewGitHub(t, c.status)
			repo := rvGhClientRepo()
			before := time.Now()
			if _, err := RepoIsPublic(context.Background(), "own", repo); err != nil {
				t.Fatal(err)
			}
			publicMu.Lock()
			a, ok := publicAnswers["own/"+repo]
			publicMu.Unlock()
			if !ok {
				t.Fatal("the answer was not kept")
			}
			if d := a.until.Sub(before); d < c.ttl || d > c.ttl+10*time.Second {
				t.Errorf("kept for %v, want %v", d, c.ttl)
			}
		})
	}
	if publicTTL < 2*time.Minute || notPublicTTL >= publicTTL {
		t.Errorf("publicTTL = %v, notPublicTTL = %v: want a few minutes for a yes and less for a no", publicTTL, notPublicTTL)
	}
}

// A kept answer stops counting when its time is up, and the new one replaces it.
func TestRvGhClientAnExpiredAnswerIsAskedAgain(t *testing.T) {
	g := rvGhClientNewGitHub(t, http.StatusOK)
	repo := rvGhClientRepo()
	publicMu.Lock()
	publicAnswers["own/"+repo] = publicAnswer{public: false, until: time.Now().Add(-time.Second)}
	publicMu.Unlock()
	for range 2 {
		if got, err := RepoIsPublic(context.Background(), "own", repo); err != nil || !got {
			t.Fatalf("RepoIsPublic = %v, %v; want true, the stale 'no' must not be served", got, err)
		}
	}
	if n := g.public.Load(); n != 1 {
		t.Errorf("asked %d times, want 1: once to replace the stale answer, then none", n)
	}
}

// What is remembered is bounded: a full memory is emptied, and the answer that
// found it full is kept.
func TestRvGhClientTheMemoryIsBounded(t *testing.T) {
	g := rvGhClientNewGitHub(t, http.StatusOK)
	publicMu.Lock()
	for i := range publicMax {
		publicAnswers[fmt.Sprintf("own/fill-%d", i)] = publicAnswer{public: true, until: time.Now().Add(time.Hour)}
	}
	publicMu.Unlock()
	repo := rvGhClientRepo()
	for range 2 {
		if _, err := RepoIsPublic(context.Background(), "own", repo); err != nil {
			t.Fatal(err)
		}
	}
	publicMu.Lock()
	n := len(publicAnswers)
	publicMu.Unlock()
	if n < 1 || n > publicMax {
		t.Errorf("%d answers kept, want 1 to %d", n, publicMax)
	}
	if got := g.public.Load(); got != 1 {
		t.Errorf("asked %d times, want 1: the answer that emptied the memory was dropped with it", got)
	}
}

// Questions arrive from request goroutines; run under -race.
func TestRvGhClientConcurrentQuestionsAreSafe(t *testing.T) {
	rvGhClientNewGitHub(t, http.StatusOK)
	repos := []string{rvGhClientRepo(), rvGhClientRepo(), rvGhClientRepo(), rvGhClientRepo()}
	var wg sync.WaitGroup
	for i := range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got, err := RepoIsPublic(context.Background(), "own", repos[i%len(repos)]); err != nil || !got {
				t.Errorf("RepoIsPublic = %v, %v; want true", got, err)
			}
		}()
	}
	wg.Wait()
}
