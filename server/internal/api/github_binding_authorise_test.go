package api

import (
	"net/http"
	"testing"
)

// rv4-COMP-05: who may link a repo is decided before GitHub is asked anything.
// Asking first would let a caller who may not link the project spend the App's
// rate limit, and read the difference between "not visible" (400) and "not
// yours" (403/404) as an answer to "is the App installed on this private repo".
func TestAnUnauthorisedLinkNeverAsksGitHub(t *testing.T) {
	for _, tc := range []struct {
		who  string
		pick func(*routeWorld) routeUser
		want int
	}{
		{"a workspace member who is not the owner", func(w *routeWorld) routeUser { return w.member }, http.StatusForbidden},
		{"someone outside the workspace", func(w *routeWorld) routeUser { return w.outsider }, http.StatusNotFound},
	} {
		t.Run(tc.who, func(t *testing.T) {
			f := newFakeGitHub(t)
			w := appServerWorld(t)
			name := uniqueRepo()
			f.addRepo(name, fakeRepo{private: true, install: 77})

			code, body := putRepoAs(t, w, tc.pick(w), w.project.ID, name)
			expect(t, "link by "+tc.who, code, body, tc.want)
			if calls := f.callsMatching(""); len(calls) != 0 {
				t.Errorf("%d request(s) went to GitHub for a link the caller may not make (first: %s %s), want none", len(calls), calls[0].Method, calls[0].Path)
			}
			if got := storedRepo(t, w); got != "" {
				t.Errorf("stored repo_url = %q after a refused link, want nothing stored", got)
			}
		})
	}
}
