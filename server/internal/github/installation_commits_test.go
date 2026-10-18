package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// rv4-FEAT-04, library half: FetchCommitsSince sends the credential it is
// handed, and nothing else, to the API host and nowhere else.

// An anonymous read sends no Authorization header at all, and the environment
// is not a credential source: a GITHUB_TOKEN set on the server is ignored.
func TestFetchCommitsSinceIsAnonymousWithoutAToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_decoy_server_wide_token")
	var mu sync.Mutex
	var auths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		auths = append(auths, r.Header.Get("Authorization"))
		mu.Unlock()
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()
	point(t, srv)

	if _, err := FetchCommitsSince(context.Background(), "", "own/repo", "", 3, 5); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(auths) != 1 || auths[0] != "" {
		t.Errorf("Authorization headers = %q, want one request with none", auths)
	}
}

func TestFetchCommitsSinceSendsTheTokenItWasGiven(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()
	point(t, srv)

	if _, err := FetchCommitsSince(context.Background(), "ghs_inst77", "own/repo", "", 3, 5); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if want := "Bearer ghs_inst77"; got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

// A rel="next" link that leaves the API origin ends the walk with an error and
// is never requested, whatever the credential. The walk is on page one here
// because the cursor is not in it.
func TestFetchCommitsSinceDoesNotFollowANextLinkToAnotherHost(t *testing.T) {
	var foreign []string
	var mu sync.Mutex
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		foreign = append(foreign, r.Header.Get("Authorization"))
		mu.Unlock()
		fmt.Fprint(w, `[]`)
	}))
	defer other.Close()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", fmt.Sprintf(`<%s/repos/own/repo/commits?page=2>; rel="next"`, other.URL))
		fmt.Fprint(w, `[{"sha":"A","commit":{"message":"m","author":{"name":"d","date":"2026-07-01T00:00:00Z"}}}]`)
	}))
	defer api.Close()
	point(t, api)

	_, err := FetchCommitsSince(context.Background(), "ghs_inst77", "own/repo", "CURSOR-NOT-ON-PAGE-ONE", 3, 5)
	if err == nil {
		t.Error("err = nil, want the walk refused")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(foreign) != 0 {
		t.Errorf("the other host received %d request(s) (Authorization %q), want none", len(foreign), foreign)
	}
}

// The same link is not an error when the walk is not going to follow it: the
// cursor was found on this page, or it is a first sync.
func TestFetchCommitsSinceIgnoresAForeignLinkItNeverFollows(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<https://evil.example/x>; rel="next"`)
		fmt.Fprint(w, `[{"sha":"B","commit":{"message":"m","author":{"name":"d","date":"2026-07-01T00:00:00Z"}}},`+
			`{"sha":"A","commit":{"message":"m","author":{"name":"d","date":"2026-07-01T00:00:00Z"}}}]`)
	}))
	defer api.Close()
	point(t, api)

	got, err := FetchCommitsSince(context.Background(), "", "own/repo", "A", 3, 5)
	if err != nil || len(got) != 1 || got[0].SHA != "B" {
		t.Errorf("cursor found: commits=%v err=%v, want just B", shas(got), err)
	}
	got, err = FetchCommitsSince(context.Background(), "", "own/repo", "", 3, 5)
	if err != nil || len(got) != 2 {
		t.Errorf("first sync: commits=%v err=%v, want the page", shas(got), err)
	}
}
