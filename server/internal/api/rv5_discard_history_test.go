package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// rv5 TREEB-10, the half wave 1 left: deleting a node cascades through its
// subtree, so a subtree holding evidence or finished work is deleted only
// with force, and any forced discard (a delete or a rebuild) leaves a note in
// the project feed. Real PostgreSQL.

// discardNotes returns the feed's note rows whose text mentions what.
func discardNotes(t *testing.T, w *routeWorld, projectID uuid.UUID, what string) []string {
	t.Helper()
	ups, err := w.st.ListProjectUpdates(context.Background(), w.owner.ID, projectID, 0, nil)
	if err != nil {
		t.Fatalf("list updates: %v", err)
	}
	var out []string
	for _, u := range ups {
		if u.Kind == "note" && strings.Contains(u.Content, what) {
			out = append(out, u.Content)
		}
	}
	return out
}

func TestDeletingASubtreeWithHistoryNeedsForce(t *testing.T) {
	w := newTreeWorld(t)
	ctx := context.Background()
	if err := w.st.AddRoadmapEvidence(ctx, w.owner.ID, w.card.ID, "commit", "abc1234 card", ""); err != nil {
		t.Fatalf("evidence: %v", err)
	}
	status, body := w.call(w.owner, "DELETE", nodePath(w.payments.ID), "")
	expect(t, "delete over evidence", status, body, http.StatusConflict)
	h := decodeAs[replaceConflictView](t, body)
	if h.Code != "roadmap_delete_has_history" || h.EvidenceCount == nil || *h.EvidenceCount != 1 ||
		h.DoneCount == nil || *h.DoneCount != 0 || h.ValidatedCount == nil || *h.ValidatedCount != 0 {
		t.Fatalf("delete over evidence: %s, want roadmap_delete_has_history with the counts", body)
	}
	if _, err := w.st.GetRoadmapNode(ctx, w.owner.ID, w.card.ID); err != nil {
		t.Fatalf("a refused delete lost Card: %v", err)
	}

	w.must(t, w.owner, "DELETE", nodePath(w.shipping.ID), "", http.StatusOK)
	if notes := discardNotes(t, w.routeWorld, w.project.ID, "Shipping"); len(notes) != 0 {
		t.Errorf("deleting a subtree with no history left notes: %v", notes)
	}
	w.must(t, w.owner, "DELETE", nodePath(w.payments.ID)+"?force=true", "", http.StatusOK)
	if notes := discardNotes(t, w.routeWorld, w.project.ID, "Payments"); len(notes) != 1 || !strings.Contains(notes[0], "1 evidence") {
		t.Errorf("forced delete notes: %v, want one naming Payments and its 1 evidence row", notes)
	}
}

func TestAForcedRebuildLeavesANote(t *testing.T) {
	w := newBuildWorld(t)
	ctx := context.Background()
	prdID := w.ownerPrd(t, true)
	tree := `{"title":"MVP","children":[{"title":"First"},{"title":"Second"}]}`
	status, body := w.call(w.owner, "POST", buildPath(prdID)+"/confirm", `{"tree":`+tree+`}`)
	expect(t, "first build", status, body, http.StatusCreated)
	projectID := uuid.MustParse(decodeAs[buildDraft](t, body).ProjectID)
	nodes, err := w.st.ListRoadmap(ctx, w.owner.ID, projectID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if err := w.st.AddRoadmapEvidence(ctx, w.owner.ID, nodes[len(nodes)-1].ID, "commit", "abc1234 first", ""); err != nil {
		t.Fatalf("evidence: %v", err)
	}
	w.must(t, w.owner, "POST", buildPath(prdID)+"/confirm", `{"expectedCount":3,"force":true,"tree":`+tree+`}`, http.StatusCreated)
	if notes := discardNotes(t, w, projectID, "1 evidence"); len(notes) != 1 {
		t.Errorf("forced rebuild notes: %v, want one naming the discarded evidence", notes)
	}
}
