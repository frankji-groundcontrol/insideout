package store

import (
	"context"
	"errors"
	"os"
	"testing"
)

// The Go layer must refuse on its own, not lean on RLS. Run the same store code
// as insideout_owner (BYPASSRLS), where no policy can hide the row: only the
// explicit membership checks stand between a removed member and the PRDs,
// project updates and snapshots they wrote. UpdateSections, UpdatePrdStatus,
// CreateRevision, CreatePrdCommit, UpdateProjectUpdate and DeleteProjectUpdate
// each let authorship through without asking whether the actor still belongs.
func TestAuthoredExMember_GoLayerDenies(t *testing.T) {
	ownerURL := os.Getenv("DATABASE_OWNER_URL")
	if ownerURL == "" {
		t.Skip("DATABASE_OWNER_URL not set, skipping the RLS-bypassing check of the Go layer")
	}
	ctx := context.Background()
	app := newTestStore(t)
	owner, err := Open(ctx, ownerURL)
	if err != nil {
		t.Fatalf("open owner store: %v", err)
	}
	t.Cleanup(owner.Close)
	var bypass bool
	if err := owner.Pool.QueryRow(ctx, `SELECT rolbypassrls OR rolsuper FROM pg_roles WHERE rolname = current_user`).Scan(&bypass); err != nil || !bypass {
		t.Fatalf("DATABASE_OWNER_URL role must bypass RLS for this test (bypass=%v, err=%v)", bypass, err)
	}

	f := newAuthoredFixture(t, app)
	f.remove(t, app)

	wantForbidden := func(call string, err error) {
		t.Helper()
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("removed member %s: want ErrForbidden, got %v", call, err)
		}
	}
	_, err = owner.UpdateSections(ctx, f.gone.ID, f.prd.ID, nil, map[string]string{"background": "hijacked"}, nil)
	wantForbidden("UpdateSections", err)
	_, err = owner.UpdatePrdStatus(ctx, f.gone.ID, f.prd.ID, "reviewing")
	wantForbidden("UpdatePrdStatus", err)
	_, err = owner.CreateRevision(ctx, f.gone.ID, f.prd.ID, nil)
	wantForbidden("CreateRevision", err)
	_, err = owner.CreatePrdCommit(ctx, f.gone.ID, f.prd.ID, "hijacked", "decision", "", nil, "")
	wantForbidden("CreatePrdCommit", err)
	_, err = owner.UpdateProjectUpdate(ctx, f.gone.ID, f.edit.ID, "hijacked")
	wantForbidden("UpdateProjectUpdate", err)
	wantForbidden("DeleteProjectUpdate", owner.DeleteProjectUpdate(ctx, f.gone.ID, f.del.ID))

	f.wantUntouched(t, owner)
}
