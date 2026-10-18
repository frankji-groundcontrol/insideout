package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// rv4-COMP-05, sharing half: two projects of two owners could link one repo
// URL, and a signed push for that repo was then delivered to both, so each
// owner's timeline received the other's commit lines. Default chosen here, a
// repo binds to one project at a time: a second link is refused with a clear
// answer that names nobody, a project can still re-link its own repo, and
// clearing a link frees the repo.

// anotherTenant is a second user with a workspace and a project of their own.
func anotherTenant(t *testing.T, w *routeWorld) (routeUser, *store.Project) {
	t.Helper()
	ctx := context.Background()
	u, err := w.st.CreateUser(ctx, "tenant-"+uuid.NewString()+"@example.org", "x", "tenant")
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	token, err := w.srv.tokens.MintAccessToken(u.ID)
	if err != nil {
		t.Fatalf("mint tenant token: %v", err)
	}
	ws, err := w.st.CreateWorkspace(ctx, u.ID, "Other tenant", "")
	if err != nil {
		t.Fatalf("create tenant workspace: %v", err)
	}
	p, err := w.st.CreateProject(ctx, u.ID, ws.ID, "Other tenant project", "")
	if err != nil {
		t.Fatalf("create tenant project: %v", err)
	}
	return routeUser{u, token}, p
}

func putRepoAs(t *testing.T, w *routeWorld, as routeUser, projectID uuid.UUID, repoURL string) (int, string) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"repoUrl": repoURL})
	if err != nil {
		t.Fatalf("encode body: %v", err)
	}
	return w.call(as, http.MethodPut, "/api/v1/projects/"+projectID.String()+"/repo", string(body))
}

// forceRepo writes projects.repo_url as rows written before the one-project
// rule still sit in the table: as the owner, through RLS, past the handler.
func forceRepo(t *testing.T, w *routeWorld, owner routeUser, projectID uuid.UUID, repoURL string) {
	t.Helper()
	ctx := context.Background()
	tx, err := w.st.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT set_config('app.user_id', $1, true)`, owner.ID.String()); err != nil {
		t.Fatalf("set actor: %v", err)
	}
	tag, err := tx.Exec(ctx, `UPDATE insideout.projects SET repo_url = $2 WHERE id = $1`, projectID, repoURL)
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("force repo_url: rows=%d err=%v", tag.RowsAffected(), err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func TestASecondProjectCannotLinkARepoThatIsAlreadyLinked(t *testing.T) {
	w := newRouteWorld(t)
	other, otherProject := anotherTenant(t, w)
	name := uniqueRepo()

	code, body := putRepo(t, w, name)
	expect(t, "first project links the repo", code, body, http.StatusOK)

	// Same repo in another letter case: GitHub names are not case-sensitive.
	code, body = putRepoAs(t, w, other, otherProject.ID, strings.ToLower(name))
	expect(t, "second project links the same repo", code, body, http.StatusConflict)
	if got := decodeAs[struct{ Code string }](t, body).Code; got != "GITHUB_REPO_ALREADY_LINKED" {
		t.Errorf("code = %q, want GITHUB_REPO_ALREADY_LINKED; body %s", got, body)
	}
	for _, secret := range []string{w.project.ID.String(), w.owner.ID.String()} {
		if strings.Contains(body, secret) {
			t.Errorf("the refusal names %s: it must say the repo is taken, not by whom; body %s", secret, body)
		}
	}
	if repoURL, _, err := w.st.ProjectRepoSync(context.Background(), other.ID, otherProject.ID); err != nil || repoURL != "" {
		t.Errorf("second project stored repo_url %q (err %v), want nothing", repoURL, err)
	}

	// The project that holds the link may link it again, in any spelling.
	code, body = putRepo(t, w, strings.ToLower(name))
	expect(t, "holder links its own repo again", code, body, http.StatusOK)

	// Clearing the link frees the repo for the other project.
	code, body = putRepo(t, w, "")
	expect(t, "holder clears the link", code, body, http.StatusOK)
	code, body = putRepoAs(t, w, other, otherProject.ID, name)
	expect(t, "second project links the freed repo", code, body, http.StatusOK)
}

func TestConcurrentLinksOfOneRepoLetExactlyOneProjectWin(t *testing.T) {
	w := newRouteWorld(t)
	name := uniqueRepo()
	const n = 6
	codes := make([]int, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		user, project := anotherTenant(t, w)
		spelling := name
		if i%2 == 1 {
			spelling = strings.ToLower(name)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			body, _ := json.Marshal(map[string]string{"repoUrl": spelling})
			codes[i], _ = w.call(user, http.MethodPut, "/api/v1/projects/"+project.ID.String()+"/repo", string(body))
		}()
	}
	close(start)
	wg.Wait()

	won, refused := 0, 0
	for _, c := range codes {
		switch c {
		case http.StatusOK:
			won++
		case http.StatusConflict:
			refused++
		}
	}
	if won != 1 || refused != n-1 {
		t.Errorf("%d concurrent links of one repo: %d succeeded and %d were refused (statuses %v), want exactly 1 and %d", n, won, refused, codes, n-1)
	}
}

// Rows that already share a repo (they predate the rule) still exist. A
// delivery goes to exactly one of them, the project created first, so no
// owner is sent the repo's commit lines because someone else typed the same URL.
func TestWebhookDeliversToOneProjectWhenRowsAlreadyShareARepo(t *testing.T) {
	f := newFakeGitHub(t)
	w := newRouteWorld(t)
	w.srv.cfg.GithubWebhookSecret = hookSecret
	other, otherProject := anotherTenant(t, w) // created after w.project
	name := uniqueRepo()
	f.addRepo(name, fakeRepo{shas: []string{"c1"}})
	forceRepo(t, w, w.owner, w.project.ID, "https://github.com/"+name)
	forceRepo(t, w, other, otherProject.ID, "https://github.com/"+name)

	payload := pushPayload(name, 0)
	rec := deliver(w.srv, "push", payload, hmacHeader(hookSecret, payload))
	expect(t, "signed push", rec.Code, rec.Body.String(), http.StatusOK)
	var got struct{ Projects, Commits int }
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode %q: %v", rec.Body, err)
	}
	if got.Projects != 1 {
		t.Errorf("delivery reached %d projects, want 1; body %s", got.Projects, rec.Body)
	}

	lines := func(owner routeUser, p *store.Project) int {
		ups, err := w.st.ListProjectUpdates(context.Background(), owner.ID, p.ID, 0, nil)
		if err != nil {
			t.Fatalf("list updates: %v", err)
		}
		return len(ups)
	}
	if n := lines(w.owner, w.project); n != 1 {
		t.Errorf("first project's timeline has %d line(s), want its 1 commit", n)
	}
	if n := lines(other, otherProject); n != 0 {
		t.Errorf("second project's timeline has %d line(s), want none: that repo is not delivered to it", n)
	}
}
