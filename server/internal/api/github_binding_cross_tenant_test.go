package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

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
		if got := decodeAs[struct{ Code string }](t, body).Code; got != "github_repo_already_linked" {
			t.Errorf("%s: code = %q, want github_repo_already_linked; body %s", who, got, body)
		}
	}
}

// The one answer given to "held by a project you cannot read" and to "GitHub does
// not show it" has to be true of both. The squatted owner of F199 gets it with
// the App already installed, and on an instance with no App configured a holder
// is the only cause there can be, so "install the App" alone sends them in a
// circle. The sentence names a holder as a possible cause for everyone alike.
func TestRvGhBindTheNotVisibleAnswerAllowsForAHolder(t *testing.T) {
	f := newFakeGitHub(t)
	w := appServerWorld(t)
	stranger, strangerProject := anotherTenant(t, w)
	held, hidden := uniqueRepo(), uniqueRepo()
	f.addRepo(held, fakeRepo{private: true, install: 77})
	f.addRepo(hidden, fakeRepo{private: true}) // no installation
	code, body := putRepo(t, w, held)
	expect(t, "the holder links its repo", code, body, http.StatusOK)

	for what, name := range map[string]string{"held by another workspace": held, "not shown by GitHub": hidden} {
		code, body := putRepoAs(t, w, stranger, strangerProject.ID, name)
		expect(t, "a stranger links a repo "+what, code, body, http.StatusBadRequest)
		if msg := decodeAs[struct{ Error string }](t, body).Error; !strings.Contains(msg, "another project") {
			t.Errorf("%s: the answer %q does not say that another project may hold the repo", what, msg)
		}
	}
}
