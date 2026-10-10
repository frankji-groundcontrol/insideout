package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// SEC-08: SharesWorkspaceWith must answer true for two members of the same
// workspace. It queries on the bare pool (no withUserContext), so the
// workspace_memberships SELECT policy — which only shows rows to
// insideout.current_user_id(), NULL on the bare pool — hides every row and
// the function answers false no matter what.
func TestSharesWorkspaceWith_TrueForCoMembers(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	member := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Shares WS "+uuid.NewString()[:8], "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, member.ID, ws.Code); err != nil {
		t.Fatalf("join workspace: %v", err)
	}

	shared, err := st.SharesWorkspaceWith(ctx, admin.ID, member.ID)
	if err != nil {
		t.Fatalf("shares workspace: %v", err)
	}
	if !shared {
		t.Fatal("SharesWorkspaceWith(admin, member) = false for two members of the same workspace: the bare-pool SELECT is hidden by the workspace_memberships RLS policy — fail-closed, but wrong")
	}
}
