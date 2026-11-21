package api

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// rv5-PARB-04: an agent token edits a project's title, description and status
// (the MCP project_update tool), but never its owner: owner changes are a
// human decision (owner decision 4), as on PATCH /roadmap/{nid}.
func TestAnAgentTokenMayEditAProjectButNotItsOwner(t *testing.T) {
	h := scopeServer().Handler()
	token := agentTokenFor(t, uuid.New())
	path := "/api/v1/projects/" + uuid.NewString()

	for _, tc := range []struct {
		name, body string
		open       bool
	}{
		{"title and status", `{"title":"T","status":"active"}`, true},
		{"a new owner", `{"title":"T","status":"active","ownerId":"` + uuid.NewString() + `"}`, false},
		{"no owner", `{"title":"T","status":"active","ownerId":""}`, false},
		{"null owner", `{"title":"T","status":"active","ownerId":null}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := sendWith(h, token, http.MethodPatch, path, tc.body)
			if refused := scopeRefused(status, body); refused == tc.open {
				t.Errorf("agent token, body %s: status %d body %s; scope refused = %v, want %v", tc.body, status, body, refused, !tc.open)
			}
		})
	}
}
