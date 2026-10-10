package store

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// A6: the cycle-guard recursive CTE must terminate even when the data already
// contains a cycle. With UNION ALL the walk looped forever; UNION de-dups and
// returns, so the move completes instead of hanging.
func TestRoadmap_MoveWithExistingCycle(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	admin := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Cycle WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, admin.ID, ws.ID, "Cycle", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	a, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, nil, "A", "")
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	b, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, &a.ID, "B", "")
	if err != nil {
		t.Fatalf("create B: %v", err)
	}
	c, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, nil, "C", "")
	if err != nil {
		t.Fatalf("create C: %v", err)
	}

	// Force a pre-existing A↔B cycle (A.parent=B, B.parent=A) — data the public
	// API would never create, written inside a user-context tx so RLS allows it.
	if err := st.withUserContext(ctx, admin.ID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE insideout.roadmap_nodes SET parent_id = $2 WHERE id = $1`, a.ID, b.ID)
		return err
	}); err != nil {
		t.Fatalf("force cycle: %v", err)
	}

	// Move A (whose subtree now contains the cycle) under C. descendants(A)
	// walks A→B→A…; UNION terminates at {A,B}, so this returns instead of
	// hanging. The timeout wraps only the move so slow shared-DB setup latency
	// can't eat the budget — a regression to UNION ALL hangs the CTE until the
	// deadline cancels it and the test fails.
	moveCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := st.MoveRoadmapNode(moveCtx, admin.ID, a.ID, &c.ID, 0); err != nil {
		t.Fatalf("move with existing cycle: want completion, got %v", err)
	}
}

// Moving a node onto a position a sibling already holds must shift the
// others, so the list never shows two siblings at the same position
// (BUG-017).
func TestRoadmap_MoveShiftsSiblings(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	admin := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Move WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, admin.ID, ws.ID, "Move", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	a, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, nil, "A", "")
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	b, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, nil, "B", "")
	if err != nil {
		t.Fatalf("create B: %v", err)
	}
	c, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, nil, "C", "")
	if err != nil {
		t.Fatalf("create C: %v", err)
	}

	if _, err := st.MoveRoadmapNode(ctx, admin.ID, c.ID, nil, 0); err != nil {
		t.Fatalf("move C to position 0: %v", err)
	}
	nodes, err := st.ListRoadmap(ctx, admin.ID, proj.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	got := make([]string, 0, len(nodes))
	seen := map[int]bool{}
	for _, n := range nodes {
		if n.ParentID != nil {
			continue
		}
		if seen[n.Position] {
			t.Fatalf("two roots share position %d: %+v", n.Position, nodes)
		}
		seen[n.Position] = true
		got = append(got, n.Title)
	}
	want := []string{"C", "A", "B"}
	if len(got) != len(want) {
		t.Fatalf("roots: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order: got %v, want %v (ids a=%s b=%s c=%s)", got, want, a.ID, b.ID, c.ID)
		}
	}
}
