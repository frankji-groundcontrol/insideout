package store

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// rv4-FR-F194: the owner a project is given to has to be a member of its
// workspace. UpdateProject let an admin give a project to an existing user
// outside the workspace (a project owned by someone who cannot open it) and to
// a user id that does not exist (the foreign key failed, a raw 23503).

func TestRvHandoverANewOwnerMustBeAWorkspaceMember(t *testing.T) {
	w := newRvHandoverWorld(t)
	outsider := mkUser(t, w.st)
	for name, to := range map[string]uuid.UUID{"an existing user outside the workspace": outsider.ID, "a user id that does not exist": uuid.New()} {
		t.Run(name, func(t *testing.T) {
			p, err := w.st.UpdateProject(context.Background(), w.admin.ID, w.project.ID, rvHandoverFields(w.project, &to))
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("giving the project to %s = %+v, %v; want ErrValidation", name, p, err)
			}
			rvHandoverUnchanged(t, w)
		})
	}
}

// Who may change the owner is judged before whom it is given to, so a non-admin
// owner naming a non-member is told ErrHandoverNeedsAdmin, not that the member
// is unknown. This passes without the member check.
func TestRvHandoverTheOwnersStandingIsJudgedBeforeTheNewOwner(t *testing.T) {
	w := newRvHandoverWorld(t)
	outsider := mkUser(t, w.st)
	_, err := w.st.UpdateProject(context.Background(), w.owner.ID, w.project.ID, rvHandoverFields(w.project, &outsider.ID))
	if !errors.Is(err, ErrHandoverNeedsAdmin) {
		t.Fatalf("a non-admin owner naming a non-member = %v; want ErrHandoverNeedsAdmin", err)
	}
	rvHandoverUnchanged(t, w)
}

// Only a change of owner is judged: with the owner gone from the workspace, an
// admin's edit that keeps that owner still applies. This passes without the
// member check.
func TestRvHandoverAnOwnerWhoLeftTheWorkspaceStaysWhileTheOwnerIsUnchanged(t *testing.T) {
	w := newRvHandoverWorld(t)
	ctx := context.Background()
	if err := w.st.RemoveMember(ctx, w.admin.ID, w.project.WorkspaceID, w.owner.ID); err != nil {
		t.Fatalf("the owner leaves the workspace: %v", err)
	}
	f := rvHandoverFields(w.project, &w.owner.ID)
	f.Title = "Renamed by an admin"
	p, err := w.st.UpdateProject(ctx, w.admin.ID, w.project.ID, f)
	if err != nil || p.Title != "Renamed by an admin" || p.OwnerID == nil || *p.OwnerID != w.owner.ID {
		t.Fatalf("the admin's edit with the departed owner kept = %+v, %v; want it applied, owner unchanged", p, err)
	}
}
