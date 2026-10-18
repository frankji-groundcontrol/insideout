package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// rv4-FEAT-04: the commit-timeline sync read one server-wide GITHUB_TOKEN
// (usually unset) and never the GitHub App credential. A private repo then
// answered 404 whatever the App could see, and every public repo shared
// GitHub's 60-requests-an-hour anonymous limit. The sync now borrows the
// installation token of the repo it syncs, on the manual route and on the
// webhook alike, and never sends any credential anywhere but api.github.com.

// decoyPAT is a server-wide token planted in the environment. Nothing may send
// it: it is the credential the App token replaces.
const decoyPAT = "ghp_decoy_server_wide_token"

// uniqueRepo is an owner/repo name nobody else in the shared test database
// has linked: the binding rules (and the webhook lookup) are database-wide.
func uniqueRepo() string {
	id := uuid.NewString()[:8]
	return "Own" + id + "/Repo" + id
}

func syncPath(w *routeWorld) string {
	return "/api/v1/projects/" + w.project.ID.String() + "/sync-github"
}

func syncAdded(t *testing.T, w *routeWorld) int {
	t.Helper()
	code, body := w.call(w.owner, http.MethodPost, syncPath(w), "")
	expect(t, "sync-github", code, body, http.StatusOK)
	return decodeAs[struct{ Added int }](t, body).Added
}

func pushPayload(repo string, installation int64) string {
	return fmt.Sprintf(`{"ref":"refs/heads/main","installation":{"id":%d},"repository":{"full_name":%q},"commits":[]}`, installation, repo)
}

func TestSyncUsesTheInstallationCredentialForAPrivateRepo(t *testing.T) {
	f := newFakeGitHub(t)
	w := appServerWorld(t)
	t.Setenv("GITHUB_TOKEN", decoyPAT)
	name := uniqueRepo()
	f.addRepo(name, fakeRepo{private: true, install: 77, shas: []string{"c2", "c1"}})

	code, body := putRepo(t, w, name)
	expect(t, "link the private repo", code, body, http.StatusOK)

	if got := syncAdded(t, w); got != 2 {
		t.Errorf("added = %d, want the 2 commits of the private repo", got)
	}
	calls := f.callsMatching("GET /repos/" + name + "/commits")
	if len(calls) == 0 {
		t.Fatal("no commits request reached the fake GitHub")
	}
	for _, c := range calls {
		if want := "Bearer " + fakeToken(77); c.Auth != want {
			t.Errorf("commits request carried Authorization %q, want the installation token %q", c.Auth, want)
		}
	}
	if f.sawSecret(decoyPAT) {
		t.Error("the server-wide GITHUB_TOKEN was sent to GitHub; the installation token replaces it")
	}
}

func TestWebhookPushSyncsAPrivateRepoThroughItsInstallation(t *testing.T) {
	f := newFakeGitHub(t)
	w := appServerWorld(t)
	name := uniqueRepo()
	f.addRepo(name, fakeRepo{private: true, install: 77, shas: []string{"c2", "c1"}})
	code, body := putRepo(t, w, name)
	expect(t, "link the private repo", code, body, http.StatusOK)

	payload := pushPayload(name, 77)
	rec := deliver(w.srv, "push", payload, hmacHeader(hookSecret, payload))
	expect(t, "signed push", rec.Code, rec.Body.String(), http.StatusOK)
	var got struct{ Projects, Commits int }
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode %q: %v", rec.Body, err)
	}
	if got.Projects != 1 || got.Commits != 2 {
		t.Errorf("delivery reached %d project(s) and synced %d commit(s), want 1 and 2; body %s", got.Projects, got.Commits, rec.Body)
	}
}

// Without an App credential there is nothing to borrow: the sync is anonymous
// (public repos only) and never falls back to a server-wide token that any
// member could aim at any repo it can see.
func TestSyncWithoutAnAppIsAnonymousAndIgnoresTheServerWideToken(t *testing.T) {
	f := newFakeGitHub(t)
	w := newRouteWorld(t)
	t.Setenv("GITHUB_TOKEN", decoyPAT)
	name := uniqueRepo()
	f.addRepo(name, fakeRepo{shas: []string{"c1"}})
	code, body := putRepo(t, w, name)
	expect(t, "link the public repo", code, body, http.StatusOK)

	if got := syncAdded(t, w); got != 1 {
		t.Errorf("added = %d, want 1", got)
	}
	for _, c := range f.callsMatching("GET /repos/" + name + "/commits") {
		if c.Auth != "" {
			t.Errorf("anonymous sync carried Authorization %q, want none", c.Auth)
		}
	}
	if f.sawSecret(decoyPAT) {
		t.Error("the server-wide GITHUB_TOKEN was sent to GitHub")
	}
}

// The sync follows rel="next" links only while they stay on the API host: a
// link that points elsewhere would carry the token with it. The repo is public
// so the paging is reached with or without a credential; the assertion is that
// nothing at all leaves for another host.
func TestSyncNeverFollowsANextLinkOffTheGitHubHost(t *testing.T) {
	f := newFakeGitHub(t)
	w := appServerWorld(t)
	t.Setenv("GITHUB_TOKEN", decoyPAT)
	name := uniqueRepo()
	f.addRepo(name, fakeRepo{install: 77, shas: []string{"c1"}})
	code, body := putRepo(t, w, name)
	expect(t, "link the repo", code, body, http.StatusOK)
	syncAdded(t, w) // first sync: stores the cursor c1

	// c1 is gone from the first page (a force push), so the walk pages on,
	// and the page says the next one lives on another host.
	f.setCommits(name, `<https://evil.example/repos/x/y/commits?page=2>; rel="next"`, "c9")
	w.call(w.owner, http.MethodPost, syncPath(w), "")

	if calls := f.foreignCalls(); len(calls) != 0 {
		t.Errorf("the sync sent %d request(s) to another host (first: %+v), want none", len(calls), calls[0])
	}
	if f.sawSecret(decoyPAT) {
		t.Error("the server-wide GITHUB_TOKEN was sent to GitHub")
	}
}
