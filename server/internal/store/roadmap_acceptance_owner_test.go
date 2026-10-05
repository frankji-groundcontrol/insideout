package store

import (
	"context"
	"testing"
	"time"

)

func sptr(s string) *string { return &s }

// Gap slices 16/17: a leaf carries the sentence that says what done means,
// and a Now item has exactly one human owner (PRODUCT.md). Both ride the
// existing node update, must survive the read path (including the owner
// name join), and the owner must be a workspace member.
func TestRoadmapAcceptanceOwner_RoundTrip(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	owner := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Gap WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, owner.ID, ws.Code); err != nil {
		t.Fatalf("join: %v", err)
	}
	proj, err := st.CreateProject(ctx, admin.ID, ws.ID, "Gap project", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	root, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, nil, "Build", "")
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	leaf, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, &root.ID, "Leaf", "")
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}

	deadline := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	acceptance := "Done when the demo run finishes green twice"
	updated, err := st.UpdateRoadmapNode(ctx, admin.ID, leaf.ID, RoadmapNodeFields{
		Status:     sptr("in_progress"),
		Deadline:   &deadline,
		Acceptance: &acceptance,
		OwnerID:    &owner.ID,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Acceptance != acceptance {
		t.Fatalf("Acceptance = %q, want %q", updated.Acceptance, acceptance)
	}
	if updated.OwnerID == nil || *updated.OwnerID != owner.ID {
		t.Fatalf("OwnerID = %v, want %s", updated.OwnerID, owner.ID)
	}

	nodes, err := st.ListRoadmap(ctx, admin.ID, proj.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, n := range nodes {
		if n.ID != leaf.ID {
			continue
		}
		if n.Acceptance != acceptance {
			t.Fatalf("listed Acceptance = %q, want %q", n.Acceptance, acceptance)
		}
		if n.OwnerID == nil || *n.OwnerID != owner.ID {
			t.Fatalf("listed OwnerID = %v, want %s", n.OwnerID, owner.ID)
		}
		if n.OwnerName != owner.Username {
			t.Fatalf("listed OwnerName = %q, want %q", n.OwnerName, owner.Username)
		}
	}
}

func TestRoadmapOwner_MustBeMember(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	stranger := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Gap WS 2", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, admin.ID, ws.ID, "Gap project 2", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	leaf, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, nil, "Leaf", "")
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}

	if _, err := st.UpdateRoadmapNode(ctx, admin.ID, leaf.ID, RoadmapNodeFields{OwnerID: &stranger.ID}); err == nil {
		t.Fatal("assigning a non-member owner must be rejected")
	}
}

// Gap slice 8: status moves only through the documented flow; a jump from
// pending straight to done is refused.
func TestRoadmap_StatusTransitions(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Gap WS 3", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, admin.ID, ws.ID, "Gap project 3", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	leaf, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, nil, "Leaf", "")
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}

	if _, err := st.UpdateRoadmapNode(ctx, admin.ID, leaf.ID, RoadmapNodeFields{Status: sptr("done")}); err == nil {
		t.Fatal("pending → done must be refused")
	}
	moved, err := st.UpdateRoadmapNode(ctx, admin.ID, leaf.ID, RoadmapNodeFields{Status: sptr("in_progress")})
	if err != nil {
		t.Fatalf("pending → in_progress: %v", err)
	}
	if moved.Status != "in_progress" {
		t.Fatalf("status = %q, want in_progress", moved.Status)
	}
	if _, err := st.UpdateRoadmapNode(ctx, admin.ID, leaf.ID, RoadmapNodeFields{Status: sptr("done")}); err != nil {
		t.Fatalf("in_progress → done: %v", err)
	}
	if _, err := st.UpdateRoadmapNode(ctx, admin.ID, leaf.ID, RoadmapNodeFields{Status: sptr("in_progress")}); err != nil {
		t.Fatalf("done → in_progress reopen: %v", err)
	}
}

// Gap slice 9: a freshly built tree is visible on Progress — the first leaf
// is seeded in_progress with a deadline, so Now is not empty.
func TestReplaceRoadmapTree_FirstLeafSeeded(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Gap WS 4", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, admin.ID, ws.ID, "Gap project 4", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	_, err = st.ReplaceRoadmapTree(ctx, admin.ID, proj.ID, nil, RoadmapPlanNode{Title: "Milestone", Children: []RoadmapPlanNode{
		{Title: "First step"},
		{Title: "Second step"},
	}})
	if err != nil {
		t.Fatalf("replace tree: %v", err)
	}
	nodes, err := st.ListRoadmap(ctx, admin.ID, proj.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var seeded int
	for _, n := range nodes {
		if n.Status == "in_progress" && n.Deadline != nil {
			seeded++
		}
	}
	if seeded != 1 {
		t.Fatalf("seeded in_progress leaves with deadline = %d, want exactly 1", seeded)
	}
}
