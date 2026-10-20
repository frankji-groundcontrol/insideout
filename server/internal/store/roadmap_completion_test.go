package store

import (
	"context"
	"errors"
	"testing"
)

// Completion links to verifiable evidence and may be overridden by a human
// only with a reason (PRODUCT.md). Moving in_progress to done with neither
// linked evidence nor a completion reason must be refused.
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
}
