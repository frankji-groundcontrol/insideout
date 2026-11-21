package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// rv5-PARB-10: the free-text fields beside the capped ones get the same
// 2000-character bound, answering 400 field_too_long, and a checkpoint's
// nodeId must name a node of its project. Real PostgreSQL.

func expectTooLong(t *testing.T, what string, status int, body string) {
	t.Helper()
	if status != http.StatusBadRequest || decodeAs[struct{ Code string }](t, body).Code != "field_too_long" {
		t.Fatalf("%s: %d %s, want 400 field_too_long", what, status, body)
	}
}

func TestTheProposalAndCommitTextFieldsAreBounded(t *testing.T) {
	w := newCoverageWorld(t)
	pid := w.project.ID.String()

	propose := func(detail string) (int, string) {
		return w.send(t, w.owner, "POST", "/api/v1/agent/propose", map[string]any{
			"projectId": pid, "kind": "scope", "summary": "trim the scope", "detail": detail})
	}
	status, body := propose(cjk(2001))
	expectTooLong(t, "propose detail", status, body)
	status, body = propose(cjk(2000))
	expect(t, "propose detail at the bound", status, body, http.StatusCreated)

	update := w.proposeAsOwner(t, "Bounded")
	status, body = w.send(t, w.owner, "POST", "/api/v1/agent/proposals/"+update+"/decision",
		map[string]any{"decision": "rejected", "reason": cjk(2001)})
	expectTooLong(t, "decision reason", status, body)
	if w.decisionOf(t, update) != "" {
		t.Fatalf("a refused decision was recorded")
	}

	prd := w.prd(t)
	for _, field := range []string{"changeSummary", "decisionNote"} {
		status, body = w.send(t, w.owner, "POST", "/api/v1/prds/"+prd+"/commit",
			map[string]any{"name": "v1", "primaryAudience": "decision", field: cjk(2001)})
		expectTooLong(t, "commit "+field, status, body)
	}
	status, body = w.send(t, w.owner, "POST", "/api/v1/prds/"+prd+"/commit",
		map[string]any{"name": "v1", "primaryAudience": "decision", "changeSummary": cjk(2000), "decisionNote": cjk(2000)})
	expect(t, "commit at the bound", status, body, http.StatusCreated)
}

func TestACheckpointNodeMustBeANodeOfItsProject(t *testing.T) {
	w := newTreeWorld(t)
	other, err := w.st.CreateProject(context.Background(), w.owner.ID, w.project.WorkspaceID, "Other", "")
	if err != nil {
		t.Fatalf("other project: %v", err)
	}
	elsewhere, err := w.st.CreateRoadmapNode(context.Background(), w.owner.ID, other.ID, nil, "Elsewhere", "")
	if err != nil {
		t.Fatalf("other node: %v", err)
	}
	checkpoint := func(node string) (int, string) {
		return w.send(t, w.member, "POST", "/api/v1/agent/checkpoint",
			map[string]any{"projectId": w.project.ID.String(), "nodeId": node, "summary": "tests pass"})
	}
	for node, want := range map[string]int{
		"not-a-node":          http.StatusBadRequest,
		uuid.NewString():      http.StatusNotFound,
		elsewhere.ID.String(): http.StatusNotFound,
		w.card.ID.String():    http.StatusCreated,
		"":                    http.StatusCreated,
	} {
		status, body := checkpoint(node)
		expect(t, "checkpoint on node "+node, status, body, want)
	}
}
