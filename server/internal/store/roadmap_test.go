package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// byID indexes a node slice by its ID for relationship assertions.
func byID(nodes []RoadmapNode) map[string]RoadmapNode {
	m := make(map[string]RoadmapNode, len(nodes))
	for _, n := range nodes {
		m[n.ID.String()] = n
	}
	return m
}

// sp returns a *string for building partial RoadmapNodeFields in tests.
func sp(s string) *string { return &s }

// ip returns an *int for the expected-count precondition in tests.
func ip(i int) *int { return &i }

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
	if _, err := st.UpdateRoadmapNode(ctx, member.ID, branchA.ID, RoadmapNodeFields{Title: sp("Core MVP"), Description: sp(""), Status: sp("in_progress")}); err != nil {
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

// A1: a partial PATCH writes only the fields it names and leaves the rest
// alone — the lost-update fix. The D1 case is a status-only change on a node
// whose description is populated: the description must survive.
func TestRoadmap_PartialUpdate(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	admin := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Partial WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, admin.ID, ws.ID, "Partial", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	n, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, nil, "Original", "full description")
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	// Status-only: title + populated description preserved (D1).
	got, err := st.UpdateRoadmapNode(ctx, admin.ID, n.ID, RoadmapNodeFields{Status: sp("in_progress")})
	if err != nil {
		t.Fatalf("status-only update: %v", err)
	}
	if got.Title != "Original" || got.Description != "full description" || got.Status != "in_progress" {
		t.Fatalf("status-only: want title/desc preserved, got %+v", got)
	}

	// Title-only: status + description preserved.
	got, err = st.UpdateRoadmapNode(ctx, admin.ID, n.ID, RoadmapNodeFields{Title: sp("Renamed")})
	if err != nil {
		t.Fatalf("title-only update: %v", err)
	}
	if got.Title != "Renamed" || got.Description != "full description" || got.Status != "in_progress" {
		t.Fatalf("title-only: want status/desc preserved, got %+v", got)
	}

	// Description-only, clearing to "" — a present empty string is a real write.
	got, err = st.UpdateRoadmapNode(ctx, admin.ID, n.ID, RoadmapNodeFields{Description: sp("")})
	if err != nil {
		t.Fatalf("description-clear update: %v", err)
	}
	if got.Title != "Renamed" || got.Description != "" || got.Status != "in_progress" {
		t.Fatalf("description-clear: want desc cleared, rest preserved, got %+v", got)
	}

	// All three at once still works (parity with the old full write).
	got, err = st.UpdateRoadmapNode(ctx, admin.ID, n.ID, RoadmapNodeFields{Title: sp("T"), Description: sp("D"), Status: sp("done")})
	if err != nil {
		t.Fatalf("full update: %v", err)
	}
	if got.Title != "T" || got.Description != "D" || got.Status != "done" {
		t.Fatalf("full update: got %+v", got)
	}
}

// A4: ReplaceRoadmapTree refuses to silently wipe a non-empty roadmap unless
// the caller confirms the exact live count; empty roadmaps need no confirm.
func TestRoadmap_ReplaceTreeGuard(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	admin := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Guard WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, admin.ID, ws.ID, "Guard", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	tree := RoadmapPlanNode{Title: "Root", Children: []RoadmapPlanNode{{Title: "c1"}, {Title: "c2"}}}

	// Empty roadmap replaces with no expectedCount.
	count, err := st.ReplaceRoadmapTree(ctx, admin.ID, proj.ID, nil, tree)
	if err != nil {
		t.Fatalf("replace empty: %v", err)
	}
	if count != 3 {
		t.Fatalf("replace empty: want 3 nodes, got %d", count)
	}

	// Non-empty, no confirm → ReplaceConflictError carrying the live count.
	_, err = st.ReplaceRoadmapTree(ctx, admin.ID, proj.ID, nil, tree)
	var rc *ReplaceConflictError
	if !errors.As(err, &rc) {
		t.Fatalf("no-confirm: want ReplaceConflictError, got %v", err)
	}
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("no-confirm: want to match ErrConflict, got %v", err)
	}
	if rc.LiveCount != 3 {
		t.Fatalf("no-confirm: want LiveCount 3, got %d", rc.LiveCount)
	}

	// Wrong count → conflict, tree untouched.
	if _, err = st.ReplaceRoadmapTree(ctx, admin.ID, proj.ID, ip(99), tree); !errors.Is(err, ErrConflict) {
		t.Fatalf("wrong-count: want ErrConflict, got %v", err)
	}
	if nodes, _ := st.ListRoadmap(ctx, admin.ID, proj.ID); len(nodes) != 3 {
		t.Fatalf("wrong-count: tree must be untouched, got %d nodes", len(nodes))
	}

	// Correct live count → replace succeeds.
	if _, err = st.ReplaceRoadmapTree(ctx, admin.ID, proj.ID, ip(3), tree); err != nil {
		t.Fatalf("confirmed replace: %v", err)
	}
}

// A5: ExpandRoadmapNode is atomic — a mid-insert failure rolls the whole
// expansion back, leaving zero partial children behind.
func TestRoadmap_ExpandAtomic(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	admin := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Expand WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, admin.ID, ws.ID, "Expand", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	parent, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, nil, "Parent", "")
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	children := []RoadmapPlanNode{{Title: "k1"}, {Title: "k2"}, {Title: "k3"}}

	// Happy path: all children land, positions 0..n-1.
	created, err := st.ExpandRoadmapNode(ctx, admin.ID, parent.ID, children)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if len(created) != 3 {
		t.Fatalf("expand: want 3 children, got %d", len(created))
	}
	for i, c := range created {
		if c.Position != i || c.ParentID == nil || *c.ParentID != parent.ID {
			t.Fatalf("child %d: want position %d under parent, got %+v", i, i, c)
		}
	}

	// Fault injection: fail the 2nd insert → whole tx rolls back, nothing persists.
	expandFailAt = 2
	defer func() { expandFailAt = 0 }()
	more := []RoadmapPlanNode{{Title: "x1"}, {Title: "x2"}, {Title: "x3"}}
	if _, err := st.ExpandRoadmapNode(ctx, admin.ID, parent.ID, more); err == nil {
		t.Fatal("injected failure: want error, got nil")
	}
	nodes, err := st.ListRoadmap(ctx, admin.ID, proj.ID)
	if err != nil {
		t.Fatalf("list after rollback: %v", err)
	}
	// Still just the parent + the 3 happy-path children — no partial x* nodes.
	if len(nodes) != 4 {
		t.Fatalf("rollback: want 4 nodes (no partial expansion), got %d", len(nodes))
	}
}

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

// mkNamedUser is mkUser but with a caller-chosen username, so attribution tests
// can tell the creator apart from the editor (mkUser names everyone the same).
func mkNamedUser(t *testing.T, st *Store, username string) *User {
	t.Helper()
	u, err := st.CreateUser(context.Background(), username+"-"+uuid.NewString()+"@test.local", "x", username)
	if err != nil {
		t.Fatalf("create user %q: %v", username, err)
	}
	return u
}

// strPtr dereferences a *string for assertions, failing the test on nil.
func strPtr(t *testing.T, s *string, what string) string {
	t.Helper()
	if s == nil {
		t.Fatalf("%s: want a name, got nil", what)
	}
	return *s
}

// B3: every mutation records who did it — created_by on create/expand/replace,
// updated_by on every write — and ListRoadmap resolves those ids to display
// names via a LEFT JOIN. The visible mark is the LAST editor (D10); the creator
// is preserved separately.
func TestRoadmap_Attribution(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	alice := mkNamedUser(t, st, "alice-attr")
	bob := mkNamedUser(t, st, "bob-attr")

	ws, err := st.CreateWorkspace(ctx, alice.ID, "Attr WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, bob.ID, ws.Code); err != nil {
		t.Fatalf("bob join: %v", err)
	}
	proj, err := st.CreateProject(ctx, alice.ID, ws.ID, "Attr", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	// replace: the whole generated tree is attributed to the caller (create path).
	tree := RoadmapPlanNode{Title: "Root", Children: []RoadmapPlanNode{{Title: "c1"}, {Title: "c2"}}}
	if _, err := st.ReplaceRoadmapTree(ctx, alice.ID, proj.ID, nil, tree); err != nil {
		t.Fatalf("replace: %v", err)
	}
	nodes, err := st.ListRoadmap(ctx, bob.ID, proj.ID)
	if err != nil || len(nodes) != 3 {
		t.Fatalf("list after replace: %d nodes, err=%v", len(nodes), err)
	}
	idx := byID(nodes)
	var rootID uuid.UUID
	for _, n := range nodes {
		if n.ParentID == nil {
			rootID = n.ID
		}
		if n.CreatedBy == nil || *n.CreatedBy != alice.ID || n.UpdatedBy == nil || *n.UpdatedBy != alice.ID {
			t.Fatalf("replace attribution: node %s want created/updated by alice, got %+v", n.ID, n)
		}
		if got := strPtr(t, n.CreatorName, "replace CreatorName"); got != "alice-attr" {
			t.Fatalf("replace CreatorName: want alice-attr, got %q", got)
		}
	}

	// update: bob flips the root's status → updated_by becomes bob, created_by
	// stays alice (the visible mark now differs from the creator — the D10 case).
	root, err := st.UpdateRoadmapNode(ctx, bob.ID, rootID, RoadmapNodeFields{Status: sp("in_progress")})
	if err != nil {
		t.Fatalf("bob update: %v", err)
	}
	if root.CreatedBy == nil || *root.CreatedBy != alice.ID || root.UpdatedBy == nil || *root.UpdatedBy != bob.ID {
		t.Fatalf("update attribution: want created=alice updated=bob, got %+v", root)
	}
	nodes, _ = st.ListRoadmap(ctx, bob.ID, proj.ID)
	idx = byID(nodes)
	rn := idx[rootID.String()]
	if got := strPtr(t, rn.CreatorName, "root CreatorName"); got != "alice-attr" {
		t.Fatalf("root CreatorName: want alice-attr, got %q", got)
	}
	if got := strPtr(t, rn.EditorName, "root EditorName"); got != "bob-attr" {
		t.Fatalf("root EditorName: want bob-attr (last editor), got %q", got)
	}

	// move: alice reparents c1 under c2 → c1's updated_by becomes alice.
	var c1, c2 RoadmapNode
	for _, n := range nodes {
		if n.Title == "c1" {
			c1 = n
		}
		if n.Title == "c2" {
			c2 = n
		}
	}
	moved, err := st.MoveRoadmapNode(ctx, alice.ID, c1.ID, &c2.ID, 0)
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	if moved.UpdatedBy == nil || *moved.UpdatedBy != alice.ID {
		t.Fatalf("move attribution: want updated_by=alice, got %+v", moved)
	}

	// expand: bob breaks c2 into children → each child is created+updated by bob.
	created, err := st.ExpandRoadmapNode(ctx, bob.ID, c2.ID, []RoadmapPlanNode{{Title: "k1"}, {Title: "k2"}})
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	for _, ch := range created {
		if ch.CreatedBy == nil || *ch.CreatedBy != bob.ID || ch.UpdatedBy == nil || *ch.UpdatedBy != bob.ID {
			t.Fatalf("expand attribution: child %s want created/updated by bob, got %+v", ch.ID, ch)
		}
	}

	// D6: a removed author must not drop the node. ON DELETE SET NULL leaves
	// created_by/updated_by NULL; simulate that directly and confirm the LEFT
	// JOIN still returns the node with nil names (rendered "unknown" upstream).
	if err := st.withUserContext(ctx, alice.ID, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx, `UPDATE insideout.roadmap_nodes SET created_by = NULL, updated_by = NULL WHERE id = $1`, rootID)
		return e
	}); err != nil {
		t.Fatalf("null out attribution: %v", err)
	}
	nodes, err = st.ListRoadmap(ctx, alice.ID, proj.ID)
	if err != nil {
		t.Fatalf("list after null-out: %v", err)
	}
	idx = byID(nodes)
	orphan, ok := idx[rootID.String()]
	if !ok {
		t.Fatalf("removed-author node dropped from the tree — LEFT JOIN must keep it")
	}
	if orphan.CreatorName != nil || orphan.EditorName != nil {
		t.Fatalf("removed-author names: want nil/unknown, got creator=%v editor=%v", orphan.CreatorName, orphan.EditorName)
	}

	// D7 backstop: an insert claiming a created_by that isn't the authenticated
	// caller is rejected by the tightened RLS policy. bob is a member, so the
	// membership check passes — only the created_by mismatch fails, isolating D7.
	err = st.withUserContext(ctx, alice.ID, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx, `INSERT INTO insideout.roadmap_nodes (project_id, title, created_by, updated_by) VALUES ($1, 'spoof', $2, $2)`, proj.ID, bob.ID)
		return e
	})
	if err == nil {
		t.Fatal("spoofed created_by: want RLS rejection, got nil")
	}
}
