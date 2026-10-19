package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// Whole-change review of the repo binding, API half (rv4-FR-F193, F205, F200,
// F121; F199's oracle; the typed refusal of rv4-FR-F170 / F186 / F130). The
// store half is in internal/store/rv4_review_github_binding_test.go. The
// entitlement question (who may bind a repo the App can see: F119, F32, F27 and
// F199's squatting) is the owner's and is not tested here: it has no answer yet.

// rvGhBindSecondProject is another project in the workspace of w.project, owned
// by w.owner (the workspace admin).
func rvGhBindSecondProject(t *testing.T, w *routeWorld) *store.Project {
	t.Helper()
	p, err := w.st.CreateProject(context.Background(), w.owner.ID, w.project.WorkspaceID, "Second project", "")
	if err != nil {
		t.Fatalf("create the second project: %v", err)
	}
	return p
}

// rvGhBindOwnerless clears a project's owner the way PATCH /projects/{pid} does
// when no ownerId is sent (UpdateProject writes owner_id = NULL), as w.owner,
// the workspace admin.
func rvGhBindOwnerless(t *testing.T, w *routeWorld, p *store.Project) {
	t.Helper()
	f := store.ProjectUpdateFields{Title: p.Title, Description: p.Description, Status: p.Status}
	if _, err := w.st.UpdateProject(context.Background(), w.owner.ID, p.ID, f); err != nil {
		t.Fatalf("clear the owner of project %s: %v", p.ID, err)
	}
}

func rvGhBindTimelineLines(t *testing.T, w *routeWorld, viewer routeUser, p *store.Project) int {
	t.Helper()
	ups, err := w.st.ListProjectUpdates(context.Background(), viewer.ID, p.ID, 0, nil)
	if err != nil {
		t.Fatalf("list updates: %v", err)
	}
	return len(ups)
}

// A project that kept its link and lost its owner (PATCH without an ownerId)
// still holds the repo: a sibling project in the same workspace is told the repo
// is taken instead of silently taking it, and the admin can release the link.
func TestRvGhBindAnOwnerlessProjectStillHoldsItsRepoOverHTTP(t *testing.T) {
	w := newRouteWorld(t)
	second := rvGhBindSecondProject(t, w)
	name := uniqueRepo()
	code, body := putRepo(t, w, name)
	expect(t, "the project links the repo", code, body, http.StatusOK)
	rvGhBindOwnerless(t, w, w.project)

	code, body = putRepoAs(t, w, w.owner, second.ID, strings.ToLower(name))
	expect(t, "a second project links the repo an ownerless project holds", code, body, http.StatusConflict)
	if got := decodeAs[struct{ Code string }](t, body).Code; got != "GITHUB_REPO_ALREADY_LINKED" {
		t.Errorf("code = %q, want GITHUB_REPO_ALREADY_LINKED; body %s", got, body)
	}

	// The admin can release the ownerless holder, and the repo is then free.
	code, body = putRepo(t, w, "")
	expect(t, "admin clears the ownerless holder's link", code, body, http.StatusOK)
	code, body = putRepoAs(t, w, w.owner, second.ID, name)
	expect(t, "the second project links the freed repo", code, body, http.StatusOK)
}

// Rows that share a repo (they predate the one-project rule) go to the one
// created first. When that one has no owner to act as, the delivery is not
// handed to the other: nothing is synced, and the answer says so.
func TestRvGhBindAPushForAnOwnerlessHolderIsNotHandedToAnotherProject(t *testing.T) {
	f := newFakeGitHub(t)
	w := newRouteWorld(t)
	w.srv.cfg.GithubWebhookSecret = hookSecret
	other, otherProject := anotherTenant(t, w) // created after w.project
	name := uniqueRepo()
	f.addRepo(name, fakeRepo{shas: []string{"c1"}})
	forceRepo(t, w, w.owner, w.project.ID, "https://github.com/"+name)
	forceRepo(t, w, other, otherProject.ID, "https://github.com/"+name)
	rvGhBindOwnerless(t, w, w.project)

	payload := pushPayload(name, 0)
	rec := deliver(w.srv, "push", payload, hmacHeader(hookSecret, payload))
	expect(t, "signed push", rec.Code, rec.Body.String(), http.StatusOK)
	got := decodeAs[struct{ Projects, Skipped, Commits int }](t, rec.Body.String())
	if got.Projects != 0 || got.Skipped != 1 || got.Commits != 0 {
		t.Errorf("delivery answered %s, want 0 projects synced, 1 skipped (the holder has no owner to act as) and 0 commits", rec.Body)
	}
	if n := rvGhBindTimelineLines(t, w, other, otherProject); n != 0 {
		t.Errorf("the other project's timeline has %d line(s), want none: the first project holds the repo, owner or not", n)
	}
}

// A workspace admin can still sync a project that has lost its owner, with the
// credential the project's binding earned: an ownerless holder is still the
// holder commitToken looks for.
func TestRvGhBindAnAdminCanStillSyncAnOwnerlessProjectThatHoldsAPrivateRepo(t *testing.T) {
	f := newFakeGitHub(t)
	w := appServerWorld(t)
	name := uniqueRepo()
	f.addRepo(name, fakeRepo{private: true, install: 77, shas: []string{"c2", "c1"}})
	code, body := putRepo(t, w, name)
	expect(t, "link the private repo", code, body, http.StatusOK)
	rvGhBindOwnerless(t, w, w.project)

	if got := syncAdded(t, w); got != 2 {
		t.Errorf("admin's sync added %d commit(s), want the 2 of the private repo", got)
	}
}

// F199, oracle half: _projects_by_repo is SECURITY DEFINER and sees every
// workspace's projects, so the 409 for a repo another workspace holds told any
// signed-in user which repos are linked anywhere on the instance. A caller who
// cannot read the holder is answered exactly as for a repo GitHub does not show
// to the server. A caller who can (a project of the holder's own workspace) is
// still told the repo is taken: that reveals nothing they could not read.
func TestRvGhBindARepoHeldByAnotherWorkspaceIsAnsweredLikeOneGitHubDoesNotShow(t *testing.T) {
	f := newFakeGitHub(t)
	w := appServerWorld(t)
	stranger, strangerProject := anotherTenant(t, w)
	held, hidden := uniqueRepo(), uniqueRepo()
	f.addRepo(held, fakeRepo{private: true, install: 77})
	f.addRepo(hidden, fakeRepo{private: true}) // no installation: GitHub shows it to nobody
	code, body := putRepo(t, w, held)
	expect(t, "the holder links its repo", code, body, http.StatusOK)

	heldCode, heldBody := putRepoAs(t, w, stranger, strangerProject.ID, held)
	hiddenCode, hiddenBody := putRepoAs(t, w, stranger, strangerProject.ID, hidden)
	if heldCode != hiddenCode || heldBody != hiddenBody {
		t.Errorf("a stranger can tell a repo another workspace holds (%d %s) from one GitHub does not show (%d %s)", heldCode, heldBody, hiddenCode, hiddenBody)
	}
	for _, id := range []string{w.project.ID.String(), w.owner.ID.String()} {
		if strings.Contains(heldBody, id) {
			t.Errorf("the refusal names %s; body %s", id, heldBody)
		}
	}
	if repoURL, _, err := w.st.ProjectRepoSync(context.Background(), stranger.ID, strangerProject.ID); err != nil || repoURL != "" {
		t.Errorf("the stranger's project stored repo_url %q (err %v), want nothing", repoURL, err)
	}

	// Inside the holder's workspace the specific answer stays: a sibling project
	// of the admin, and the project of a plain member.
	mine, err := w.st.CreateProject(context.Background(), w.member.ID, w.project.WorkspaceID, "Member project", "")
	if err != nil {
		t.Fatalf("create the member's project: %v", err)
	}
	for who, link := range map[string]func() (int, string){
		"the admin's second project":   func() (int, string) { return putRepoAs(t, w, w.owner, rvGhBindSecondProject(t, w).ID, held) },
		"a plain member's own project": func() (int, string) { return putRepoAs(t, w, w.member, mine.ID, held) },
	} {
		code, body := link()
		expect(t, who+" links the held repo", code, body, http.StatusConflict)
		if got := decodeAs[struct{ Code string }](t, body).Code; got != "GITHUB_REPO_ALREADY_LINKED" {
			t.Errorf("%s: code = %q, want GITHUB_REPO_ALREADY_LINKED; body %s", who, got, body)
		}
	}
}
