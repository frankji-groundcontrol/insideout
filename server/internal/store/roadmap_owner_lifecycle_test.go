package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// rv5 row 2.9 (TREEB-12, FEAT-05): ownership follows membership, and work in
// progress always has a human owner. Real PostgreSQL.

func ownerOf(t *testing.T, st *Store, viewer, node uuid.UUID) *uuid.UUID {
	t.Helper()
	n, err := st.GetRoadmapNode(context.Background(), viewer, node)
	if err != nil {
		t.Fatalf("read node: %v", err)
	}
	return n.OwnerID
}

func TestLeavingAWorkspaceClearsYourNodeOwnership(t *testing.T) {
	w := newDelegateWorld(t)
	st, ctx := w.st, context.Background()
	if _, err := st.UpdateRoadmapNode(ctx, w.owner.ID, w.shipLeaf.ID, RoadmapNodeFields{OwnerID: &w.other.ID}); err != nil {
		t.Fatalf("assign Ship leaf: %v", err)
	}
	// The member's own workspace is not touched by leaving this one.
	ws, err := st.CreateWorkspace(ctx, w.member.ID, "Member's own", "")
	if err != nil {
		t.Fatalf("create own workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, w.member.ID, ws.ID, "Own project", "")
	if err != nil {
		t.Fatalf("create own project: %v", err)
	}
	own, err := st.CreateRoadmapNode(ctx, w.member.ID, proj.ID, nil, "Own root", "")
	if err != nil {
		t.Fatalf("create own node: %v", err)
	}
	if _, err := st.UpdateRoadmapNode(ctx, w.member.ID, own.ID, RoadmapNodeFields{OwnerID: &w.member.ID}); err != nil {
		t.Fatalf("own the node: %v", err)
	}

	// An admin removes the member; a plain member leaves on their own.
	if err := st.RemoveMember(ctx, w.owner.ID, w.project.WorkspaceID, w.member.ID); err != nil {
		t.Fatalf("remove member: %v", err)
	}
	if err := st.RemoveMember(ctx, w.other.ID, w.project.WorkspaceID, w.other.ID); err != nil {
		t.Fatalf("other leaves: %v", err)
	}
	if o := ownerOf(t, st, w.owner.ID, w.payments.ID); o != nil {
		t.Errorf("Payments still owned by the removed member %s", o)
	}
	if o := ownerOf(t, st, w.owner.ID, w.shipLeaf.ID); o != nil {
		t.Errorf("Ship leaf still owned by the member who left %s", o)
	}
	if o := ownerOf(t, st, w.member.ID, own.ID); o == nil || *o != w.member.ID {
		t.Errorf("the member's node in their own workspace lost its owner: %v", o)
	}
}

func TestStartingAnOwnerlessLeafMakesYouItsOwner(t *testing.T) {
	w := newDelegateWorld(t)
	st, ctx := w.st, context.Background()
	start := RoadmapNodeFields{Status: sptr("in_progress")}

	// The delegate starts a leaf below their node: they own it now.
	if _, err := st.UpdateRoadmapNode(ctx, w.member.ID, w.card.ID, start); err != nil {
		t.Fatalf("member starts Card: %v", err)
	}
	if o := ownerOf(t, st, w.owner.ID, w.card.ID); o == nil || *o != w.member.ID {
		t.Errorf("Card owner = %v, want the member who started it", o)
	}
	// A leaf with an owner keeps that owner when someone else starts it.
	if _, err := st.UpdateRoadmapNode(ctx, w.owner.ID, w.refunds.ID, RoadmapNodeFields{OwnerID: &w.other.ID}); err != nil {
		t.Fatalf("assign Refunds: %v", err)
	}
	if _, err := st.UpdateRoadmapNode(ctx, w.owner.ID, w.refunds.ID, start); err != nil {
		t.Fatalf("owner starts Refunds: %v", err)
	}
	if o := ownerOf(t, st, w.owner.ID, w.refunds.ID); o == nil || *o != w.other.ID {
		t.Errorf("Refunds owner = %v, want the one already assigned", o)
	}
	// A branch is not an execution item: starting it assigns nobody.
	if _, err := st.UpdateRoadmapNode(ctx, w.owner.ID, w.shipping.ID, start); err != nil {
		t.Fatalf("owner starts Shipping: %v", err)
	}
	if o := ownerOf(t, st, w.owner.ID, w.shipping.ID); o != nil {
		t.Errorf("starting a branch assigned %s", o)
	}
}
