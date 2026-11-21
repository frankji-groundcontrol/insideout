package store

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
)

// Roadmap node dependencies (2026-11-22 plan, "Dependency contract"): a node
// may wait on other nodes of its project, across branches, never on its own
// ancestor or descendant and never around a cycle. Real PostgreSQL, on the
// delegate probe: Root > Payments (member's) > Card, Refunds; Root > Shipping
// > Ship leaf.

func dependsOn(t *testing.T, w *delegateWorld, id uuid.UUID) []uuid.UUID {
	t.Helper()
	n, err := w.st.GetRoadmapNode(context.Background(), w.owner.ID, id)
	if err != nil {
		t.Fatalf("read node: %v", err)
	}
	return n.DependsOn
}

func TestANodeDependsOnANodeInAnotherBranch(t *testing.T) {
	w := newDelegateWorld(t)
	st, ctx := w.st, context.Background()

	n, err := st.AddRoadmapDependency(ctx, w.owner.ID, w.card.ID, w.shipLeaf.ID)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if !slices.Equal(n.DependsOn, []uuid.UUID{w.shipLeaf.ID}) {
		t.Fatalf("view dependsOn = %v, want [ship leaf]", n.DependsOn)
	}
	if _, err := st.AddRoadmapDependency(ctx, w.owner.ID, w.card.ID, w.shipLeaf.ID); err != nil {
		t.Fatalf("re-add is idempotent: %v", err)
	}
	// The member writes Card (they own Payments) and may wait on any node they see.
	if _, err := st.AddRoadmapDependency(ctx, w.member.ID, w.card.ID, w.shipping.ID); err != nil {
		t.Fatalf("delegate adds inside their subtree: %v", err)
	}
	if got := dependsOn(t, w, w.card.ID); !slices.Equal(got, []uuid.UUID{w.shipLeaf.ID, w.shipping.ID}) {
		t.Errorf("dependsOn = %v, want [ship leaf, shipping] in the order added", got)
	}
	list, err := st.ListRoadmap(ctx, w.other.ID, w.project.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, l := range list {
		if want := l.ID == w.card.ID; want != (len(l.DependsOn) == 2) {
			t.Errorf("list: %s dependsOn = %v", l.Title, l.DependsOn)
		}
	}

	if _, err := st.AddRoadmapDependency(ctx, w.member.ID, w.shipLeaf.ID, w.card.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("member on a node they do not write: %v, want ErrForbidden", err)
	}
	if _, err := st.AddRoadmapDependency(ctx, w.other.ID, w.refunds.ID, w.shipLeaf.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("plain member: %v, want ErrForbidden", err)
	}
}

func TestADependencyIsRefusedOnItselfInItsTreeOrAroundACycle(t *testing.T) {
	w := newDelegateWorld(t)
	st, ctx, o := w.st, context.Background(), w.owner.ID
	add := func(from, to *RoadmapNode) error {
		_, err := st.AddRoadmapDependency(ctx, o, from.ID, to.ID)
		return err
	}

	if err := add(w.card, w.card); !errors.Is(err, ErrDependencySelf) {
		t.Errorf("self: %v, want ErrDependencySelf", err)
	}
	for _, c := range []struct {
		name     string
		from, to *RoadmapNode
	}{{"on its parent", w.card, w.payments}, {"on its root", w.card, w.root}, {"on its child", w.payments, w.refunds}, {"root on a leaf", w.root, w.shipLeaf}} {
		if err := add(c.from, c.to); !errors.Is(err, ErrDependencyInTree) {
			t.Errorf("%s: %v, want ErrDependencyInTree", c.name, err)
		}
	}

	if err := add(w.card, w.shipLeaf); err != nil {
		t.Fatalf("card on ship leaf: %v", err)
	}
	if err := add(w.shipLeaf, w.card); !errors.Is(err, ErrDependencyCycle) {
		t.Errorf("direct cycle: %v, want ErrDependencyCycle", err)
	}
	if err := add(w.shipLeaf, w.refunds); err != nil {
		t.Fatalf("ship leaf on refunds: %v", err)
	}
	if err := add(w.refunds, w.card); !errors.Is(err, ErrDependencyCycle) {
		t.Errorf("three-node cycle: %v, want ErrDependencyCycle", err)
	}
	if !errors.Is(ErrDependencyCycle, ErrConflict) || !errors.Is(ErrDependencyInTree, ErrConflict) || !errors.Is(ErrDependencySelf, ErrValidation) {
		t.Error("dependency errors must wrap ErrConflict (cycle, in tree) and ErrValidation (self)")
	}
}

func TestADependencyStaysInsideOneVisibleProject(t *testing.T) {
	w := newDelegateWorld(t)
	st, ctx := w.st, context.Background()
	other, err := st.CreateProject(ctx, w.owner.ID, w.project.WorkspaceID, "Elsewhere", "")
	if err != nil {
		t.Fatalf("second project: %v", err)
	}
	far, err := st.CreateRoadmapNode(ctx, w.owner.ID, other.ID, nil, "Far", "")
	if err != nil {
		t.Fatalf("far node: %v", err)
	}
	if _, err := st.AddRoadmapDependency(ctx, w.owner.ID, w.card.ID, far.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("another project: %v, want ErrNotFound", err)
	}
	if _, err := st.AddRoadmapDependency(ctx, w.owner.ID, w.card.ID, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown node: %v, want ErrNotFound", err)
	}
	outsider := mkUser(t, st)
	if _, err := st.AddRoadmapDependency(ctx, outsider.ID, w.card.ID, w.shipLeaf.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("outsider: %v, want ErrNotFound", err)
	}

	// The database holds the same line for SQL the store never wrote.
	writeWalkStore(t)
	const insert = `INSERT INTO insideout.roadmap_dependencies (node_id, depends_on_id, created_by) VALUES ($1, $2, $3)`
	if _, err := asApp(t, st, w.owner.ID, "", insert, w.card.ID, far.ID, w.owner.ID); !denied(err) {
		t.Errorf("raw cross-project insert: want 42501, got %v", err)
	}
	if _, err := asApp(t, st, w.member.ID, "", insert, w.shipLeaf.ID, w.card.ID, w.member.ID); !denied(err) {
		t.Errorf("raw insert on a node the member does not write: want 42501, got %v", err)
	}
	if _, err := asApp(t, st, w.member.ID, "", insert, w.card.ID, w.shipLeaf.ID, w.member.ID); err != nil {
		t.Errorf("raw insert inside the member's subtree: %v", err)
	}
}

func TestRemovingADependencyAndItsCascades(t *testing.T) {
	w := newDelegateWorld(t)
	st, ctx, o := w.st, context.Background(), w.owner.ID
	for _, to := range []*RoadmapNode{w.shipLeaf, w.shipping} {
		if _, err := st.AddRoadmapDependency(ctx, o, w.card.ID, to.ID); err != nil {
			t.Fatalf("add: %v", err)
		}
	}
	if err := st.RemoveRoadmapDependency(ctx, w.other.ID, w.card.ID, w.shipping.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("plain member removes: %v, want ErrForbidden", err)
	}
	if err := st.RemoveRoadmapDependency(ctx, w.member.ID, w.card.ID, w.shipping.ID); err != nil {
		t.Fatalf("delegate removes: %v", err)
	}
	if err := st.RemoveRoadmapDependency(ctx, o, w.card.ID, w.shipping.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("remove a link that is gone: %v, want ErrNotFound", err)
	}

	// Deleting the node waited on drops the link.
	if err := st.DeleteRoadmapNode(ctx, o, w.shipLeaf.ID, false); err != nil {
		t.Fatalf("delete ship leaf: %v", err)
	}
	if got := dependsOn(t, w, w.card.ID); len(got) != 0 {
		t.Errorf("after deleting the node waited on, dependsOn = %v", got)
	}

	// Replacing the tree deletes nodes that are linked: without the cascade
	// the delete would fail on the foreign key.
	if _, err := st.AddRoadmapDependency(ctx, o, w.card.ID, w.shipping.ID); err != nil {
		t.Fatalf("re-add: %v", err)
	}
	count := 5
	if _, err := st.ReplaceRoadmapTree(ctx, o, w.project.ID, ReplaceOptions{ExpectedCount: &count}, RoadmapPlanNode{Title: "New"}); err != nil {
		t.Fatalf("replace a tree with a dependency: %v", err)
	}
}
