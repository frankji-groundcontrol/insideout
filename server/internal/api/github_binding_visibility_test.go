package api

import (
	"net/http"
	"testing"
)

// rv4-COMP-05, verification half: PUT /projects/{pid}/repo stored any URL a
// member typed. Once the server reads repos with the App's installation token
// (commit sync, guide loading), the stored link is what authorises it to read
// that repo for the project, so a link is refused unless GitHub shows the repo
// to this server: the App is installed on it, or it is public. When GitHub
// cannot be asked, nothing is stored. A deployment with no App configured has
// no credential to protect and keeps linking public repos unverified.

func TestLinkingARepoRequiresGitHubToShowItToTheServer(t *testing.T) {
	cases := []struct {
		name     string
		repo     *fakeRepo // nil: GitHub has no such repo
		down     bool
		noApp    bool
		wantCode int
		wantErr  string // the body's "code"
	}{
		{"private repo the App is installed on", &fakeRepo{private: true, install: 77}, false, false, http.StatusOK, ""},
		{"public repo the App is not installed on", &fakeRepo{}, false, false, http.StatusOK, ""},
		{"private repo the App cannot see", &fakeRepo{private: true}, false, false, http.StatusBadRequest, "GITHUB_REPO_NOT_VISIBLE"},
		{"repo GitHub has never heard of", nil, false, false, http.StatusBadRequest, "GITHUB_REPO_NOT_VISIBLE"},
		{"GitHub is down", &fakeRepo{private: true, install: 77}, true, false, http.StatusBadGateway, "GITHUB_UPSTREAM"},
		{"no App configured: nothing to verify against", &fakeRepo{private: true}, false, true, http.StatusOK, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeGitHub(t)
			w := appServerWorld(t)
			if tc.noApp {
				w.srv.ghTokens = nil
			}
			name := uniqueRepo()
			if tc.repo != nil {
				f.addRepo(name, *tc.repo)
			}
			f.setDown(tc.down)

			code, body := putRepo(t, w, name)
			expect(t, "link "+tc.name, code, body, tc.wantCode)
			if tc.wantErr != "" {
				if got := decodeAs[struct{ Code string }](t, body).Code; got != tc.wantErr {
					t.Errorf("code = %q, want %q; body %s", got, tc.wantErr, body)
				}
				if got := storedRepo(t, w); got != "" {
					t.Errorf("stored repo_url = %q after a refused link, want nothing stored", got)
				}
			}
		})
	}
}
