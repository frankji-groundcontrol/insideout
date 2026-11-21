package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// rv5 rows 1.5 and 1.6 (FEAT-03, TREEB-10, PARC-05). A rebuild deletes every
// node and, by cascade, its evidence, so it is refused while the roadmap has
// history unless the caller says force. A move that names no parent keeps the
// parent. Real PostgreSQL.

type replaceConflictView struct {
	Code           string `json:"code"`
	LiveCount      *int   `json:"liveCount"`
	EvidenceCount  *int   `json:"evidenceCount"`
	DoneCount      *int   `json:"doneCount"`
	ValidatedCount *int   `json:"validatedCount"`
}

func TestARebuildOverEvidenceNeedsForce(t *testing.T) {
	w := newBuildWorld(t)
	ctx := context.Background()
	prdID := w.ownerPrd(t, true)
	tree := `{"title":"MVP","children":[{"title":"First"},{"title":"Second"}]}`
	status, body := w.call(w.owner, "POST", buildPath(prdID)+"/confirm", `{"tree":`+tree+`}`)
	expect(t, "first build", status, body, http.StatusCreated)
	projectID := uuid.MustParse(decodeAs[buildDraft](t, body).ProjectID)
	nodes, err := w.st.ListRoadmap(ctx, w.owner.ID, projectID)
	if err != nil || len(nodes) != 3 {
		t.Fatalf("list roadmap: %d nodes, err %v", len(nodes), err)
	}
	var leaf uuid.UUID
	for _, n := range nodes {
		if n.ParentID != nil {
			leaf = n.ID
			break
		}
	}
	if err := w.st.AddRoadmapEvidence(ctx, w.owner.ID, leaf, "commit", "abc1234 first", ""); err != nil {
		t.Fatalf("add evidence: %v", err)
	}

	// No count: the existing conflict, now naming the evidence too.
	status, body = w.call(w.owner, "POST", buildPath(prdID)+"/confirm", `{"tree":`+tree+`}`)
	expect(t, "unconfirmed rebuild", status, body, http.StatusConflict)
	rc := decodeAs[replaceConflictView](t, body)
	if rc.Code != "replace_conflict" || rc.LiveCount == nil || *rc.LiveCount != 3 || rc.EvidenceCount == nil || *rc.EvidenceCount != 1 {
		t.Fatalf("unconfirmed rebuild: body %s, want replace_conflict with liveCount 3 and evidenceCount 1", body)
	}

	// The count confirmed but no force: refused, and nothing is lost.
	status, body = w.call(w.owner, "POST", buildPath(prdID)+"/confirm", `{"expectedCount":3,"tree":`+tree+`}`)
	expect(t, "rebuild over evidence", status, body, http.StatusConflict)
	h := decodeAs[replaceConflictView](t, body)
	if h.Code != "roadmap_replace_has_history" || h.LiveCount == nil || *h.LiveCount != 3 ||
		h.EvidenceCount == nil || *h.EvidenceCount != 1 || h.DoneCount == nil || *h.DoneCount != 0 ||
		h.ValidatedCount == nil || *h.ValidatedCount != 0 {
		t.Fatalf("rebuild over evidence: body %s, want roadmap_replace_has_history with the counts", body)
	}
	if ev, err := w.st.ListRoadmapEvidence(ctx, w.owner.ID, leaf); err != nil || len(ev) != 1 {
		t.Fatalf("a refused rebuild lost evidence: %d rows, err %v", len(ev), err)
	}

	// Force goes through.
	status, body = w.call(w.owner, "POST", buildPath(prdID)+"/confirm", `{"expectedCount":3,"force":true,"tree":`+tree+`}`)
	expect(t, "forced rebuild", status, body, http.StatusCreated)
	after, err := w.st.ListRoadmap(ctx, w.owner.ID, projectID)
	if err != nil || len(after) != 3 {
		t.Fatalf("after forced rebuild: %d nodes, err %v", len(after), err)
	}
	for _, n := range after {
		if n.ID == leaf {
			t.Fatalf("a forced rebuild kept the old node %s", leaf)
		}
	}
}

func TestAPositionOnlyMoveKeepsTheParent(t *testing.T) {
	w := newRouteWorld(t)
	ctx := context.Background()
	parent, err := w.st.CreateRoadmapNode(ctx, w.owner.ID, w.project.ID, nil, "Parent", "")
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	var kids []uuid.UUID
	for _, title := range []string{"A", "B"} {
		n, err := w.st.CreateRoadmapNode(ctx, w.owner.ID, w.project.ID, &parent.ID, title, "")
		if err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
		kids = append(kids, n.ID)
	}
	move := "/api/v1/roadmap/" + kids[1].String() + "/move"

	body := w.must(t, w.owner, "POST", move, `{"position":0}`, http.StatusOK)
	got := decodeAs[roadmapNodeView](t, body)
	if got.ParentID == nil || *got.ParentID != parent.ID.String() || got.Position != 0 {
		t.Fatalf("position-only move: body %s, want parent %s at position 0", body, parent.ID)
	}

	body = w.must(t, w.owner, "POST", move, `{"parentId":null}`, http.StatusOK)
	if got := decodeAs[roadmapNodeView](t, body); got.ParentID != nil {
		t.Fatalf("explicit null parent: body %s, want a root", body)
	}
}
