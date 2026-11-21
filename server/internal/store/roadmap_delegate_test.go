package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// rv5 row 2.2 (TREEB-02, TREEX-01, TREEB-07): the person a node is handed to
// may work inside that node's subtree and nowhere else, and the database
// enforces the same boundary as the store. Real PostgreSQL.

// delegateWorld is the review's probe: the project owner (a workspace admin)
// hands "Payments" to a plain member. Root has two branches, Payments and
// Shipping; Payments has two leaves, Card and Refunds.
type delegateWorld struct {
	st                       *Store
	owner, member, other     *User
	project                  *Project
	root, payments, shipping *RoadmapNode
	card, refunds, shipLeaf  *RoadmapNode
}

func newDelegateWorld(t *testing.T) *delegateWorld {
	t.Helper()
	st := newTestStore(t)
	ctx := context.Background()
	w := &delegateWorld{st: st, owner: mkUser(t, st), member: mkUser(t, st), other: mkUser(t, st)}
	ws, err := st.CreateWorkspace(ctx, w.owner.ID, "Delegate probe", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	for _, u := range []*User{w.member, w.other} {
		if _, err := st.JoinWorkspace(ctx, u.ID, ws.Code); err != nil {
			t.Fatalf("join: %v", err)
		}
	}
	if w.project, err = st.CreateProject(ctx, w.owner.ID, ws.ID, "Delegate probe", ""); err != nil {
		t.Fatalf("create project: %v", err)
	}
	add := func(parent *RoadmapNode, title string) *RoadmapNode {
		t.Helper()
		var pid *uuid.UUID
		if parent != nil {
			pid = &parent.ID
		}
		n, err := st.CreateRoadmapNode(ctx, w.owner.ID, w.project.ID, pid, title, "")
		if err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
		return n
	}
	w.root = add(nil, "Root")
	w.payments = add(w.root, "Payments")
	w.shipping = add(w.root, "Shipping")
	w.card = add(w.payments, "Card")
	w.refunds = add(w.payments, "Refunds")
	w.shipLeaf = add(w.shipping, "Ship leaf")
	if _, err := st.UpdateRoadmapNode(ctx, w.owner.ID, w.payments.ID, RoadmapNodeFields{OwnerID: &w.member.ID}); err != nil {
		t.Fatalf("assign Payments: %v", err)
	}
	return w
}

func TestADelegateWritesInsideTheirSubtree(t *testing.T) {
	w := newDelegateWorld(t)
	st, ctx, m := w.st, context.Background(), w.member.ID

	if _, err := st.UpdateRoadmapNode(ctx, m, w.payments.ID, RoadmapNodeFields{Title: sptr("Payments v2")}); err != nil {
		t.Errorf("rename the delegated node: %v", err)
	}
	if _, err := st.UpdateRoadmapNode(ctx, m, w.card.ID, RoadmapNodeFields{Status: sptr("in_progress")}); err != nil {
		t.Errorf("start a leaf under it: %v", err)
	}
	child, err := st.CreateRoadmapNode(ctx, m, w.project.ID, &w.payments.ID, "Wallets", "")
	if err != nil {
		t.Fatalf("add a child under it: %v", err)
	}
	if _, err := st.ExpandRoadmapNode(ctx, m, w.refunds.ID, []RoadmapPlanNode{{Title: "Partial refunds"}}); err != nil {
		t.Errorf("break a node down: %v", err)
	}
	if _, err := st.MoveRoadmapNode(ctx, m, w.card.ID, &w.refunds.ID, 0); err != nil {
		t.Errorf("move a leaf within the subtree: %v", err)
	}
	if err := st.AddValidationObservation(ctx, m, child.ID, "customers paid", "", time.Now()); err != nil {
		t.Errorf("add evidence: %v", err)
	}
	if _, err := st.UpdateRoadmapNode(ctx, m, child.ID, RoadmapNodeFields{OwnerID: &w.other.ID}); err != nil {
		t.Errorf("re-delegate a descendant: %v", err)
	}
	if err := st.DeleteRoadmapNode(ctx, m, child.ID); err != nil {
		t.Errorf("delete a node below the delegated root: %v", err)
	}
}

func TestADelegateIsRefusedOutsideTheirSubtree(t *testing.T) {
	w := newDelegateWorld(t)
	st, ctx, m := w.st, context.Background(), w.member.ID
	refused := func(what string, err error) {
		t.Helper()
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("%s: want ErrForbidden, got %v", what, err)
		}
	}

	_, err := st.MoveRoadmapNode(ctx, m, w.payments.ID, &w.shipping.ID, 0)
	refused("move the delegated root", err)
	refused("delete the delegated root", st.DeleteRoadmapNode(ctx, m, w.payments.ID))
	_, err = st.UpdateRoadmapNode(ctx, m, w.payments.ID, RoadmapNodeFields{OwnerID: &w.other.ID})
	refused("reassign the delegated root", err)
	_, err = st.UpdateRoadmapNode(ctx, m, w.payments.ID, RoadmapNodeFields{ClearOwner: true})
	refused("clear the delegated root's owner", err)
	_, err = st.UpdateRoadmapNode(ctx, m, w.shipLeaf.ID, RoadmapNodeFields{Title: sptr("mine now")})
	refused("edit a sibling branch", err)
	_, err = st.CreateRoadmapNode(ctx, m, w.project.ID, &w.shipping.ID, "intruder", "")
	refused("add under a sibling branch", err)
	_, err = st.UpdateRoadmapNode(ctx, m, w.root.ID, RoadmapNodeFields{Title: sptr("mine now")})
	refused("edit the parent", err)
	_, err = st.CreateRoadmapNode(ctx, m, w.project.ID, nil, "second root", "")
	refused("add a root", err)
	_, err = st.MoveRoadmapNode(ctx, m, w.card.ID, &w.shipping.ID, 0)
	refused("move a leaf out of the subtree", err)
	_, err = st.ExpandRoadmapNode(ctx, m, w.shipping.ID, []RoadmapPlanNode{{Title: "x"}})
	refused("break down a sibling branch", err)
	_, err = st.ValidateRoadmapNode(ctx, m, w.card.ID, "looks right")
	refused("validate", err)
	_, err = st.ReplaceRoadmapTree(ctx, m, w.project.ID, ReplaceOptions{Force: true}, RoadmapPlanNode{Title: "mine"})
	refused("replace the tree", err)
	refused("add evidence outside", st.AddValidationObservation(ctx, m, w.shipLeaf.ID, "seen", "", time.Now()))

	// Clearing the owner takes the right away again.
	if _, err := st.UpdateRoadmapNode(ctx, w.owner.ID, w.payments.ID, RoadmapNodeFields{ClearOwner: true}); err != nil {
		t.Fatalf("owner clears Payments: %v", err)
	}
	_, err = st.UpdateRoadmapNode(ctx, m, w.card.ID, RoadmapNodeFields{Title: sptr("still mine?")})
	refused("edit after the owner was cleared", err)
}

// The database enforces the same boundary: the member's own SQL, run as
// insideout_app under their identity, changes no node outside their subtree.
func TestRLSKeepsADelegateInsideTheirSubtree(t *testing.T) {
	w := newDelegateWorld(t)
	st, m := w.st, w.member.ID
	writeWalkStore(t)

	for _, c := range []struct {
		name string
		sql  string
		id   uuid.UUID
		want int64
	}{
		{"update a sibling-branch node", `UPDATE insideout.roadmap_nodes SET title = 'x' WHERE id = $1`, w.shipLeaf.ID, 0},
		{"update the parent", `UPDATE insideout.roadmap_nodes SET title = 'x' WHERE id = $1`, w.root.ID, 0},
		{"delete a sibling-branch node", `DELETE FROM insideout.roadmap_nodes WHERE id = $1`, w.shipLeaf.ID, 0},
		{"update inside the subtree", `UPDATE insideout.roadmap_nodes SET title = 'x' WHERE id = $1`, w.card.ID, 1},
		{"delete inside the subtree", `DELETE FROM insideout.roadmap_nodes WHERE id = $1`, w.card.ID, 1},
	} {
		tag, err := asApp(t, st, m, "", c.sql, c.id)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if tag.RowsAffected() != c.want {
			t.Errorf("%s: %d rows, want %d", c.name, tag.RowsAffected(), c.want)
		}
	}

	const insert = `INSERT INTO insideout.roadmap_nodes (project_id, parent_id, title, created_by) VALUES ($1, $2, 'x', $3)`
	if _, err := asApp(t, st, m, "", insert, w.project.ID, w.shipping.ID, m); !denied(err) {
		t.Errorf("insert under a sibling branch: want 42501, got %v", err)
	}
	if _, err := asApp(t, st, m, "", insert, w.project.ID, nil, m); !denied(err) {
		t.Errorf("insert a root: want 42501, got %v", err)
	}
	if _, err := asApp(t, st, m, "", insert, w.project.ID, w.payments.ID, m); err != nil {
		t.Errorf("insert under the delegated node: %v", err)
	}

	// A writable parent does not open another tenant's project: the row's
	// project must be the parent's project, on insert and on a raw reparent.
	ctx := context.Background()
	fws, err := st.CreateWorkspace(ctx, w.other.ID, "Foreign", "")
	if err != nil {
		t.Fatalf("foreign workspace: %v", err)
	}
	fp, err := st.CreateProject(ctx, w.other.ID, fws.ID, "Foreign", "")
	if err != nil {
		t.Fatalf("foreign project: %v", err)
	}
	froot, err := st.CreateRoadmapNode(ctx, w.other.ID, fp.ID, nil, "Foreign root", "")
	if err != nil {
		t.Fatalf("foreign root: %v", err)
	}
	if _, err := asApp(t, st, m, "", insert, fp.ID, w.payments.ID, m); !denied(err) {
		t.Errorf("insert into a foreign project under the delegated node: want 42501, got %v", err)
	}
	const reparent = `UPDATE insideout.roadmap_nodes SET parent_id = $2 WHERE id = $1`
	if _, err := asApp(t, st, m, "", reparent, w.refunds.ID, froot.ID); !denied(err) {
		t.Errorf("reparent under another project's node: want 42501, got %v", err)
	}
}
