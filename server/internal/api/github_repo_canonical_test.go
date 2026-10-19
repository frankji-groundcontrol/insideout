package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/github"
	"github.com/google/uuid"
)

// rv4-SEC-06: PUT /projects/{pid}/repo parsed the URL to validate it and then
// stored the raw string. Two things followed: a hostile string (a dot-only
// segment, a foreign host) was kept in the row the sync later reads, and the
// webhook, which matches the stored text exactly against
// https://github.com/<full_name>, never found a project whose owner typed
// "Owner/Repo" or "github.com/Owner/Repo.git/".

func repoPath(w *routeWorld) string {
	return "/api/v1/projects/" + w.project.ID.String() + "/repo"
}

func putRepo(t *testing.T, w *routeWorld, repoURL string) (int, string) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"repoUrl": repoURL})
	if err != nil {
		t.Fatalf("encode body: %v", err)
	}
	return w.call(w.owner, http.MethodPut, repoPath(w), string(body))
}

// storedRepo reads the column the sync and the webhook read.
func storedRepo(t *testing.T, w *routeWorld) string {
	t.Helper()
	repoURL, _, err := w.st.ProjectRepoSync(context.Background(), w.owner.ID, w.project.ID)
	if err != nil {
		t.Fatalf("read the stored repo: %v", err)
	}
	return repoURL
}

func TestSetProjectRepoStoresTheCanonicalURL(t *testing.T) {
	w := newRouteWorld(t)
	// Unique per run: the webhook lookup below is database-wide.
	id := uuid.NewString()[:8]
	owner, repo := "Own"+id, "Repo"+id
	want := "https://github.com/" + owner + "/" + repo

	for name, in := range map[string]string{
		"scheme, dot git and slash": "https://github.com/" + owner + "/" + repo + ".git/",
		"no scheme":                 "github.com/" + owner + "/" + repo,
		"bare owner/repo":           owner + "/" + repo,
		"a pasted browse URL":       "https://github.com/" + owner + "/" + repo + "/tree/main",
		"already canonical":         want,
	} {
		t.Run(name, func(t *testing.T) {
			code, body := putRepo(t, w, in)
			expect(t, "PUT repo "+in, code, body, http.StatusOK)
			if got := decodeAs[projectView](t, body).RepoURL; got != want {
				t.Errorf("response repoUrl = %q, want %q", got, want)
			}
			if got := storedRepo(t, w); got != want {
				t.Errorf("stored repo_url = %q, want %q", got, want)
			}
			// The webhook asks for https://github.com/<full_name> and the
			// lookup is an exact match, so this is the string that must be stored.
			targets, err := w.st.ProjectsByRepo(context.Background(), github.RepoURLFor(owner+"/"+repo))
			if err != nil {
				t.Fatalf("webhook lookup: %v", err)
			}
			if len(targets) != 1 || targets[0].ProjectID != w.project.ID {
				t.Errorf("webhook lookup for %s/%s found %d project(s), want exactly this one", owner, repo, len(targets))
			}
		})
	}
}

func TestSetProjectRepoRefusesHostileURLsAndStoresNothing(t *testing.T) {
	w := newRouteWorld(t)
	for _, in := range []string{
		"../..",
		"./..",
		"../user",
		"owner/..",
		"https://evil.example/x/github.com/a/b",
		"https://github.com.evil.example/a/b",
	} {
		t.Run(in, func(t *testing.T) {
			code, body := putRepo(t, w, in)
			expect(t, "PUT repo "+in, code, body, http.StatusBadRequest)
		})
	}
	if got := storedRepo(t, w); got != "" {
		t.Errorf("stored repo_url = %q after refused requests, want it untouched (empty)", got)
	}
}

// Clearing the link stays allowed, and does not turn into a canonical URL of
// nothing. (A name nobody else links: a repo binds to one project, database-wide.)
func TestSetProjectRepoStillClears(t *testing.T) {
	w := newRouteWorld(t)
	code, body := putRepo(t, w, uniqueRepo())
	expect(t, "link", code, body, http.StatusOK)
	code, body = putRepo(t, w, "")
	expect(t, "clear", code, body, http.StatusOK)
	if got := storedRepo(t, w); got != "" {
		t.Errorf("stored repo_url = %q after clearing, want empty", got)
	}
}
