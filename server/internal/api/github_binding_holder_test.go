package api

import (
	"net/http"
	"testing"
)

// rv4-COMP-05: rows that predate the one-project rule can still share a repo,
// and a project that names a repo must not be handed the credential of a
// binding it does not hold. The project the repo is bound to (the one created
// first) reads a private repo with the installation token; the other reads
// anonymously, which for a private repo is "not found".
func TestOnlyTheProjectHoldingARepoSyncsItWithTheInstallationToken(t *testing.T) {
	f := newFakeGitHub(t)
	w := appServerWorld(t) // w.project is created before the other tenant's
	other, otherProject := anotherTenant(t, w)
	name := uniqueRepo()
	f.addRepo(name, fakeRepo{private: true, install: 77, shas: []string{"c2", "c1"}})
	forceRepo(t, w, w.owner, w.project.ID, "https://github.com/"+name)
	forceRepo(t, w, other, otherProject.ID, "https://github.com/"+name)

	if got := syncAdded(t, w); got != 2 {
		t.Errorf("holder synced %d commit(s), want the 2 of the private repo", got)
	}

	code, body := w.call(other, http.MethodPost, "/api/v1/projects/"+otherProject.ID.String()+"/sync-github", "")
	expect(t, "sync by the project that does not hold the repo", code, body, http.StatusNotFound)

	var withToken, anonymous int
	for _, c := range f.callsMatching("GET /repos/" + name + "/commits") {
		switch c.Auth {
		case "Bearer " + fakeToken(77):
			withToken++
		case "":
			anonymous++
		default:
			t.Errorf("commits request carried Authorization %q", c.Auth)
		}
	}
	if withToken != 1 || anonymous != 1 {
		t.Errorf("commits requests: %d with the installation token and %d anonymous, want 1 and 1 (the holder's and the other's)", withToken, anonymous)
	}
}
