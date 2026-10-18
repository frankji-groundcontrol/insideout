package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// rv4-FEAT-01: POST /prds/{id}/status is the one route the agent scope opens
// halfway. An author submits a PRD for review (draft, reviewing) and takes it
// back; approving or rejecting is the review gate, and it stays with the human
// session. The verdict reads the request body, so it has to hold for every
// shape of body the handler's own decoder could read as something else.
func TestAnAgentTokenMaySubmitAPrdForReviewAndNothingMore(t *testing.T) {
	h := scopeServer().Handler()
	token := agentTokenFor(t, uuid.New())
	path := "/api/v1/prds/" + uuid.NewString() + "/status"

	cases := []struct {
		name string
		body string
		open bool
	}{
		{"draft", `{"status":"draft"}`, true},
		{"reviewing", `{"status":"reviewing"}`, true},
		{"approved", `{"status":"approved"}`, false},
		{"rejected", `{"status":"rejected"}`, false},
		{"no body", ``, false},
		{"truncated json", `{"status":`, false},
		{"not an object", `["reviewing"]`, false},
		{"unknown status", `{"status":"published"}`, false},
		{"wrong case", `{"status":"APPROVED"}`, false},
		{"key in another case", `{"STATUS":"approved"}`, false},
		{"approved after reviewing", `{"status":"reviewing","status":"approved"}`, false},
		{"reviewing after approved", `{"status":"approved","status":"reviewing"}`, true}, // the handler's decoder also keeps the last
		{"large body", `{"status":"reviewing","note":"` + strings.Repeat("x", 16<<10) + `"}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := sendWith(h, token, http.MethodPost, path, tc.body)
			if refused := scopeRefused(status, body); refused == tc.open {
				t.Errorf("agent token with body %.60q: status %d body %s; scope refused = %v, want %v", tc.body, status, body, refused, !tc.open)
			}
		})
	}
}
