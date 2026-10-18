package apiclient

import (
	"encoding/json"
	"testing"
)

// Review-4 FULL-07. Expanding a roadmap node drafts its sub-tasks with the same
// planner model as a full build (POST /roadmap/{id}/expand), so the slow provider
// that cuts a build off cuts an expand off too. The deadline each request is sent
// with comes from its route; this is the second route that waits on the model.
func TestNodeExpandOutlivesTheQuickDeadline(t *testing.T) {
	c, rec := recordedClient(t)
	if _, err := c.ExpandNode("n1"); err != nil {
		t.Fatalf("expand: %v", err)
	}
	if _, err := c.ConfirmExpandNode("n1", json.RawMessage(`[]`)); err != nil {
		t.Fatalf("confirm expand: %v", err)
	}

	wantDeadline(t, rec, "POST /api/v1/roadmap/n1/expand", true)          // drafts the sub-tasks with the planner's model
	wantDeadline(t, rec, "POST /api/v1/roadmap/n1/expand/confirm", false) // only writes the sub-tasks the caller holds
}
