package api

import (
	"net/http"
	"testing"
)

// rv5 wave 2 follow-up: every answer that is a project view carries the
// computed canWritePlan, the repo link included. Real PostgreSQL.
func TestTheRepoLinkAnswerSaysWhetherTheCallerWritesThePlan(t *testing.T) {
	w := newRouteWorld(t)
	got := decodeAs[map[string]any](t, w.must(t, w.owner, "PUT", w.proj()+"/repo", `{"repoUrl":""}`, http.StatusOK))
	if got["canWritePlan"] != true {
		t.Errorf("owner's PUT repo response: canWritePlan = %v, want true", got["canWritePlan"])
	}
}
