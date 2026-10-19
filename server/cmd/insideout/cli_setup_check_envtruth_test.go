package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// rv4-FEAT-04 (residue R3-27): the server no longer reads GITHUB_TOKEN, so
// `insideout setup --check` is the only reader left. With no token set, its
// github line used to say "evidence sync runs unauthenticated" - a claim about
// the server's sync, which the token no longer influences. It must say there is
// nothing to validate and that only this check reads the variable.

// rvEnvTruthGitHubLine runs `setup --check` with no GITHUB_TOKEN anywhere and
// returns the output line of the github probe.
func rvEnvTruthGitHubLine(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	pinnedNoRepo(t) // also empties GITHUB_TOKEN, DATABASE_URL and the LLM key
	t.Setenv("INSIDEOUT_API", srv.URL+"/api/v1")

	var code int
	out := captureStdout(t, func() { _, code = runSetupCommand([]string{"--check"}) })
	if code != 0 {
		t.Fatalf("setup --check exit %d, output %q; an absent optional token must not fail the run", code, out)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "github") {
			return line
		}
	}
	t.Fatalf("no github line in the output %q", out)
	return ""
}

func TestRvEnvTruthGitHubSkipSaysWhoReadsTheToken(t *testing.T) {
	line := rvEnvTruthGitHubLine(t)

	if !strings.Contains(line, "skip") {
		t.Errorf("github line %q: want a skip, an absent optional token is not a failure", line)
	}
	if strings.Contains(strings.ToLower(line), "unauthenticated") {
		t.Errorf("github line %q claims an unauthenticated sync, which this token does not control", line)
	}
	for _, want := range []string{"optional", "only this check reads it"} {
		if !strings.Contains(line, want) {
			t.Errorf("github line %q: want it to say %q", line, want)
		}
	}
}
