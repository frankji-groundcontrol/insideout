package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

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
	if err := st.UpdateMemberRole(ctx, alice.ID, ws.ID, bob.ID, "admin"); err != nil {
		t.Fatalf("promote bob: %v", err)
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
