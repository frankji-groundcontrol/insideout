package api

import (
	"net/http"
	"testing"
)

// Carry-forward from rv4-SEC-06: a repo_url stored before ParseRepoURL learned
// to refuse foreign hosts and dot-only segments is read back by the sync and
// refused by the same parser. That is a link the owner can fix by linking the
// repo again, not a GitHub outage: it answers 400 with its own code, not 502
// GITHUB_UPSTREAM, and nothing is sent to GitHub for it.
func TestSyncOfAStoredButRefusedURLIsABadRequestNotAnOutage(t *testing.T) {
	f := newFakeGitHub(t)
	w := appServerWorld(t)
	for _, stored := range []string{"https://evil.example/a/b", "../..", "own/.."} {
		t.Run(stored, func(t *testing.T) {
			forceRepo(t, w, w.owner, w.project.ID, stored)
			code, body := w.call(w.owner, http.MethodPost, syncPath(w), "")
			expect(t, "sync of "+stored, code, body, http.StatusBadRequest)
			if got := decodeAs[struct{ Code string }](t, body).Code; got != "GITHUB_REPO_URL_INVALID" {
				t.Errorf("code = %q, want GITHUB_REPO_URL_INVALID; body %s", got, body)
			}
		})
	}
	if calls := f.callsMatching(""); len(calls) != 0 {
		t.Errorf("%d request(s) went to GitHub for links that cannot be repos (first: %+v), want none", len(calls), calls[0])
	}
}
