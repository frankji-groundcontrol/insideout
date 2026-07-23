package store

import (
	"context"
	"errors"
	"testing"
)

// byID indexes a node slice by its ID for relationship assertions.
func byID(nodes []RoadmapNode) map[string]RoadmapNode {
	m := make(map[string]RoadmapNode, len(nodes))
	for _, n := range nodes {
		m[n.ID.String()] = n
	}
	return m
}

func TestRoadmap_TreeLifecycle(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	member := mkUser(t, st)
	stranger := mkUser(t, st)

	ws, err := st.CreateWorkspace(ctx, admin.ID, "Roadmap WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, member.ID, ws.Code); err != nil {
		t.Fatalf("member join: %v", err)
	}
	proj, err := st.CreateProject(ctx, admin.ID, ws.ID, "Roadmap Project", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	pid := proj.ID

	// Deny: a stranger cannot even read the tree (RLS hides the project, so
	// existence is not disclosed — ErrNotFound, not ErrForbidden).
	if _, err := st.ListRoadmap(ctx, stranger.ID, pid); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stranger ListRoadmap: want ErrNotFound, got %v", err)
	}
	if _, err := st.CreateRoadmapNode(ctx, stranger.ID, pid, nil, "x", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stranger CreateRoadmapNode: want ErrNotFound, got %v", err)
	}

	// Build a branched tree: root with two parallel branches, one grandchild.
	root, err := st.CreateRoadmapNode(ctx, admin.ID, pid, nil, "MVP", "")
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	if root.ParentID != nil || root.Position != 0 {
		t.Fatalf("root: want nil parent + position 0, got parent=%v pos=%d", root.ParentID, root.Position)
	}
	branchA, err := st.CreateRoadmapNode(ctx, admin.ID, pid, &root.ID, "Core MVP", "")
	if err != nil {
		t.Fatalf("create branchA: %v", err)
	}
	branchB, err := st.CreateRoadmapNode(ctx, member.ID, pid, &root.ID, "Launch", "")
	if err != nil {
		t.Fatalf("create branchB (member): %v", err)
	}
	if branchA.Position != 0 || branchB.Position != 1 {
		t.Fatalf("sibling positions: want 0,1 got %d,%d", branchA.Position, branchB.Position)
	}
	leaf, err := st.CreateRoadmapNode(ctx, admin.ID, pid, &branchA.ID, "Auth flow", "")
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}

	// Cross-project parent is rejected.
	otherProj, err := st.CreateProject(ctx, admin.ID, ws.ID, "Other", "")
	if err != nil {
		t.Fatalf("create other project: %v", err)
	}
	if _, err := st.CreateRoadmapNode(ctx, admin.ID, otherProj.ID, &root.ID, "bad", ""); !errors.Is(err, ErrConflict) {
		t.Fatalf("cross-project parent: want ErrConflict, got %v", err)
	}

	// List returns all four nodes with correct parentage.
	nodes, err := st.ListRoadmap(ctx, member.ID, pid)
	if err != nil {
		t.Fatalf("member ListRoadmap: %v", err)
	}
	if len(nodes) != 4 {
		t.Fatalf("list: want 4 nodes, got %d", len(nodes))
	}
	idx := byID(nodes)
	if *idx[branchA.ID.String()].ParentID != root.ID || *idx[leaf.ID.String()].ParentID != branchA.ID {
		t.Fatalf("parentage wrong: %+v", idx)
	}

	// Collaborative: a plain member can update status.
	if _, err := st.UpdateRoadmapNode(ctx, member.ID, branchA.ID, RoadmapNodeFields{Title: "Core MVP", Description: "", Status: "in_progress"}); err != nil {
		t.Fatalf("member UpdateRoadmapNode: %v", err)
	}

	// Reparent branchB under branchA.
	moved, err := st.MoveRoadmapNode(ctx, admin.ID, branchB.ID, &branchA.ID, 1)
	if err != nil {
		t.Fatalf("move branchB under branchA: %v", err)
	}
	if moved.ParentID == nil || *moved.ParentID != branchA.ID {
		t.Fatalf("move: want parent branchA, got %v", moved.ParentID)
	}

	// Cycle guard: cannot move the root under its own descendant.
	if _, err := st.MoveRoadmapNode(ctx, admin.ID, root.ID, &leaf.ID, 0); !errors.Is(err, ErrConflict) {
		t.Fatalf("move root under descendant: want ErrConflict, got %v", err)
	}
	// Cycle guard: cannot move a node under itself.
	if _, err := st.MoveRoadmapNode(ctx, admin.ID, branchA.ID, &branchA.ID, 0); !errors.Is(err, ErrConflict) {
		t.Fatalf("move under self: want ErrConflict, got %v", err)
	}

	// Delete branchA — its whole subtree (leaf + the moved branchB) cascades.
	if err := st.DeleteRoadmapNode(ctx, admin.ID, branchA.ID); err != nil {
		t.Fatalf("delete branchA: %v", err)
	}
	nodes, err = st.ListRoadmap(ctx, admin.ID, pid)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(nodes) != 1 || nodes[0].ID != root.ID {
		t.Fatalf("after cascade: want only root left, got %d nodes", len(nodes))
	}
}
