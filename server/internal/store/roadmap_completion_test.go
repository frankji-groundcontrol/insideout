package store

import (
	"context"
	"errors"
	"testing"
)

// Completion links to verifiable evidence and may be overridden by a human
// only with a reason (PRODUCT.md). in_progress → done with neither linked
// evidence nor a one-sentence reason is refused. Evidence, or one sentence
// on the write, still reaches done. Other transitions are unchanged.
func TestRoadmapCompletion_RefusesDoneWithoutEvidenceOrReason(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Completion WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, admin.ID, ws.ID, "Completion project", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	leaf, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, nil, "Ship the pilot", "")
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}
	if _, err := st.UpdateRoadmapNode(ctx, admin.ID, leaf.ID, RoadmapNodeFields{Status: sptr("in_progress")}); err != nil {
		t.Fatalf("pending → in_progress: %v", err)
	}

	_, err = st.UpdateRoadmapNode(ctx, admin.ID, leaf.ID, RoadmapNodeFields{Status: sptr("done")})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("in_progress → done with no evidence and no reason: err = %v, want ErrValidation", err)
	}
	stayed, err := st.ListRoadmap(ctx, admin.ID, proj.ID)
	if err != nil {
		t.Fatalf("list after refusal: %v", err)
	}
	if stayed[0].Status != "in_progress" {
		t.Fatalf("refused node status = %q, want in_progress", stayed[0].Status)
	}

	for _, bad := range []string{"Done.", "It shipped. Then we waited."} {
		_, err = st.UpdateRoadmapNode(ctx, admin.ID, leaf.ID, RoadmapNodeFields{Status: sptr("done"), CompletionReason: sptr(bad)})
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("reason %q: err = %v, want ErrValidation", bad, err)
		}
	}

	reason := "The pilot shipped."
	done, err := st.UpdateRoadmapNode(ctx, admin.ID, leaf.ID, RoadmapNodeFields{Status: sptr("done"), CompletionReason: &reason})
	if err != nil {
		t.Fatalf("one-sentence reason: %v", err)
	}
	if done.Status != "done" || done.CompletionReason != reason {
		t.Fatalf("done node = status %q reason %q", done.Status, done.CompletionReason)
	}
	listed, err := st.ListRoadmap(ctx, admin.ID, proj.ID)
	if err != nil {
		t.Fatalf("list after reason: %v", err)
	}
	if listed[0].CompletionReason != reason {
		t.Fatalf("listed reason = %q, want %q", listed[0].CompletionReason, reason)
	}

	// Reopening does not need a reason. Closing again still does, even
	// though the previous sentence is still on the row.
	if _, err := st.UpdateRoadmapNode(ctx, admin.ID, leaf.ID, RoadmapNodeFields{Status: sptr("in_progress")}); err != nil {
		t.Fatalf("done → in_progress: %v", err)
	}
	if _, err := st.UpdateRoadmapNode(ctx, admin.ID, leaf.ID, RoadmapNodeFields{Status: sptr("done")}); !errors.Is(err, ErrValidation) {
		t.Fatalf("second close without a new reason: err = %v, want ErrValidation", err)
	}

	other, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, nil, "Wire the evidence", "")
	if err != nil {
		t.Fatalf("create evidenced leaf: %v", err)
	}
	if _, err := st.UpdateRoadmapNode(ctx, admin.ID, other.ID, RoadmapNodeFields{Status: sptr("in_progress")}); err != nil {
		t.Fatalf("evidenced leaf → in_progress: %v", err)
	}
	if err := st.AddRoadmapEvidence(ctx, admin.ID, other.ID, "commit", "abc1234 shipped the pilot", ""); err != nil {
		t.Fatalf("add evidence: %v", err)
	}
	evidenced, err := st.UpdateRoadmapNode(ctx, admin.ID, other.ID, RoadmapNodeFields{Status: sptr("done")})
	if err != nil {
		t.Fatalf("in_progress → done with evidence: %v", err)
	}
	if evidenced.Status != "done" {
		t.Fatalf("evidenced status = %q, want done", evidenced.Status)
	}

	// A title edit on an in-progress node is not a completion.
	renamed, err := st.UpdateRoadmapNode(ctx, admin.ID, leaf.ID, RoadmapNodeFields{Title: sptr("Ship the revised pilot")})
	if err != nil {
		t.Fatalf("title edit while in_progress: %v", err)
	}
	if renamed.Status != "in_progress" || renamed.Title != "Ship the revised pilot" {
		t.Fatalf("title edit = status %q title %q", renamed.Status, renamed.Title)
	}
}
