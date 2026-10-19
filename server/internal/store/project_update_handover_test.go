package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// rv4-TEST-03: a project's owner who is not a workspace admin may edit the
// project but may not give it away. The projects_update policy has a USING
// clause and no WITH CHECK, so Postgres judges the new row by USING as well and
// refuses such an owner with SQLSTATE 42501 — which used to reach the caller as
// a raw error (a 500 at the API). UpdateProject now says ErrForbidden before it
// writes. Real PostgreSQL and its row-level security throughout.

// rvHandoverWorld: admin made the workspace and the project and then gave the
// project to owner, a plain member; other is a third member.
type rvHandoverWorld struct {
	st                  *Store
	admin, owner, other *User
	project             *Project
}

func newRvHandoverWorld(t *testing.T) *rvHandoverWorld {
	t.Helper()
	st := newTestStore(t)
	ctx := context.Background()
	w := &rvHandoverWorld{st: st, admin: mkUser(t, st), owner: mkUser(t, st), other: mkUser(t, st)}
	ws, err := st.CreateWorkspace(ctx, w.admin.ID, "Handover", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	for _, u := range []*User{w.owner, w.other} {
		if _, err := st.JoinWorkspace(ctx, u.ID, ws.Code); err != nil {
			t.Fatalf("join workspace: %v", err)
		}
	}
	p, err := st.CreateProject(ctx, w.admin.ID, ws.ID, "Handover", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if w.project, err = st.UpdateProject(ctx, w.admin.ID, p.ID, rvHandoverFields(p, &w.owner.ID)); err != nil {
		t.Fatalf("the admin gives the project to a member: %v", err)
	}
	return w
}

// rvHandoverFields is the project's record as it stands, owned by owner.
func rvHandoverFields(p *Project, owner *uuid.UUID) ProjectUpdateFields {
	return ProjectUpdateFields{Title: p.Title, Description: p.Description, Status: p.Status, OwnerID: owner}
}

// rvHandoverUnchanged fails the test unless the project is still the one the
// world made: same title, still owned by the plain member.
func rvHandoverUnchanged(t *testing.T, w *rvHandoverWorld) {
	t.Helper()
	got, err := w.st.GetProjectForMember(context.Background(), w.project.ID, w.admin.ID)
	if err != nil {
		t.Fatalf("read the project back: %v", err)
	}
	if got.Title != w.project.Title || got.OwnerID == nil || *got.OwnerID != w.owner.ID {
		t.Fatalf("project after a refused edit = title %q owner %v, want %q owned by %s", got.Title, got.OwnerID, w.project.Title, w.owner.ID)
	}
}

func TestRvHandoverANonAdminOwnerCannotGiveTheProjectAway(t *testing.T) {
	w := newRvHandoverWorld(t)
	for name, to := range map[string]*uuid.UUID{"to another member": &w.other.ID, "to nobody": nil} {
		t.Run(name, func(t *testing.T) {
			f := rvHandoverFields(w.project, to)
			f.Title = "Sneaked"
			p, err := w.st.UpdateProject(context.Background(), w.owner.ID, w.project.ID, f)
			if !errors.Is(err, ErrHandoverNeedsAdmin) || !errors.Is(err, ErrForbidden) {
				t.Fatalf("hand-over %s = %+v, %v; want ErrHandoverNeedsAdmin (an ErrForbidden)", name, p, err)
			}
			rvHandoverUnchanged(t, w)
		})
	}
}

// Only the project's owner is told it was a hand-over they may not make; a
// member who owns nothing here keeps the plain refusal, whatever they send.
func TestRvHandoverAMemberWhoDoesNotOwnTheProjectKeepsThePlainRefusal(t *testing.T) {
	w := newRvHandoverWorld(t)
	for name, to := range map[string]*uuid.UUID{"keeping the owner": &w.owner.ID, "to themselves": &w.other.ID, "to nobody": nil} {
		t.Run(name, func(t *testing.T) {
			_, err := w.st.UpdateProject(context.Background(), w.other.ID, w.project.ID, rvHandoverFields(w.project, to))
			if !errors.Is(err, ErrForbidden) || errors.Is(err, ErrHandoverNeedsAdmin) {
				t.Fatalf("a non-owner's edit %s = %v; want the plain ErrForbidden", name, err)
			}
			rvHandoverUnchanged(t, w)
		})
	}
}

// What the policy keeps allowing keeps working: the owner's edit that keeps the
// owner, and an admin's hand-over or clearing. These pass without the fix.
func TestRvHandoverPlainEditsAndAdminHandOversStayAllowed(t *testing.T) {
	w := newRvHandoverWorld(t)
	ctx := context.Background()

	f := rvHandoverFields(w.project, &w.owner.ID)
	f.Title, f.Status = "Edited by its owner", "paused"
	p, err := w.st.UpdateProject(ctx, w.owner.ID, w.project.ID, f)
	if err != nil || p.Title != "Edited by its owner" || p.Status != "paused" || p.OwnerID == nil || *p.OwnerID != w.owner.ID {
		t.Fatalf("the owner's plain edit = %+v, %v; want it applied, owner unchanged", p, err)
	}

	p, err = w.st.UpdateProject(ctx, w.admin.ID, w.project.ID, rvHandoverFields(p, &w.other.ID))
	if err != nil || p.OwnerID == nil || *p.OwnerID != w.other.ID {
		t.Fatalf("the admin's hand-over = %+v, %v; want the other member to own it", p, err)
	}
	p, err = w.st.UpdateProject(ctx, w.admin.ID, w.project.ID, rvHandoverFields(p, nil))
	if err != nil || p.OwnerID != nil {
		t.Fatalf("the admin clearing the owner = %+v, %v; want it ownerless", p, err)
	}
}

// The refusal is decided before the UPDATE is tried. Holding the project row's
// lock makes that visible: a call that reached the UPDATE waits for the lock
// and runs out its deadline, a refused one answers at once. The admin's
// hand-over in the same spot is the control: it must reach the UPDATE, so it
// waits — which shows the lock does hold the UPDATE up.
func TestRvHandoverTheRefusalComesBeforeTheUpdate(t *testing.T) {
	w := newRvHandoverWorld(t)
	ctx := context.Background()

	holder, err := w.st.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin the lock holder: %v", err)
	}
	t.Cleanup(func() { _ = holder.Rollback(context.Background()) })
	if _, err := holder.Exec(ctx, `SELECT set_config('app.user_id', $1, true)`, w.admin.ID.String()); err != nil {
		t.Fatalf("act as the admin: %v", err)
	}
	if tag, err := holder.Exec(ctx, `UPDATE insideout.projects SET title = title WHERE id = $1`, w.project.ID); err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("lock the project row: %v, %d rows", err, tag.RowsAffected())
	}

	within := func(d time.Duration) (context.Context, context.CancelFunc) { return context.WithTimeout(ctx, d) }
	toOther := rvHandoverFields(w.project, &w.other.ID)

	waitCtx, cancel := within(1500 * time.Millisecond)
	defer cancel()
	began := time.Now()
	_, err = w.st.UpdateProject(waitCtx, w.admin.ID, w.project.ID, toOther)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(began) < time.Second {
		t.Fatalf("the admin's hand-over under the held lock = %v after %s; want it to wait out its deadline (the control)", err, time.Since(began))
	}

	refuseCtx, cancelRefuse := within(5 * time.Second)
	defer cancelRefuse()
	began = time.Now()
	_, err = w.st.UpdateProject(refuseCtx, w.owner.ID, w.project.ID, toOther)
	if !errors.Is(err, ErrForbidden) || time.Since(began) > time.Second {
		t.Fatalf("the owner's hand-over under the held lock = %v after %s; want ErrForbidden at once, before the UPDATE", err, time.Since(began))
	}
}

// The backstop. A caller the pre-check let through could still meet the policy
// (the role changing after the check, say); the UPDATE then fails with 42501
// and must come out as ErrForbidden, not as a raw policy error. updateProjectRow
// is the UPDATE UpdateProject runs, driven here without the pre-check and with
// a real refusal from real Postgres.
func TestRvHandoverARefusedUpdateIsForbiddenNotARawPolicyError(t *testing.T) {
	w := newRvHandoverWorld(t)
	ctx := context.Background()

	// The premise: the policy itself refuses a non-admin owner's hand-over, 42501.
	err := w.st.withUserContext(ctx, w.owner.ID, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx, `UPDATE insideout.projects SET owner_id = $2 WHERE id = $1`, w.project.ID, w.other.ID)
		return e
	})
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Fatalf("premise: the bare UPDATE answered %v, want a 42501 policy refusal", err)
	}

	err = w.st.withUserContext(ctx, w.owner.ID, func(tx pgx.Tx) error {
		_, e := updateProjectRow(ctx, tx, w.project.ID, rvHandoverFields(w.project, &w.other.ID))
		return e
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("updateProjectRow under the policy's refusal = %v, want ErrForbidden", err)
	}
	rvHandoverUnchanged(t, w)
}
