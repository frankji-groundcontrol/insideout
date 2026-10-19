package store

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// rv4 whole-change review, cluster w4-rv-store-db. Real PostgreSQL, no mocks.
//
// Root A (rv4-FR-F37, F22, F34, F23, F33): DATA-01 closed the membership
// INSERT, but the same invariant ("a membership row needs a reason") was left
// open on UPDATE (an admin could re-point user_id or workspace_id of an
// existing row) and the admin branch of the INSERT rule trusted
// workspaces.creator_id forever, a column that survives removal from the
// workspace and that any admin could rewrite. Every statement below runs as
// insideout_app in a transaction that is always rolled back.

// rvStoreDbSession is one rolled-back transaction on the app pool in which the
// acting user can change between statements.
type rvStoreDbSession struct {
	t   *testing.T
	ctx context.Context
	tx  pgx.Tx
}

// rvStoreDbRun runs fn in a transaction that is rolled back whatever fn did.
func rvStoreDbRun(t *testing.T, st *Store, fn func(s *rvStoreDbSession)) {
	t.Helper()
	ctx := context.Background()
	tx, err := st.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)
	fn(&rvStoreDbSession{t: t, ctx: ctx, tx: tx})
}

// as makes actor the acting user (app.user_id) for the statements that follow.
func (s *rvStoreDbSession) as(actor uuid.UUID) {
	s.t.Helper()
	if _, err := s.tx.Exec(s.ctx, `SELECT set_config('app.user_id', $1, true)`, actor.String()); err != nil {
		s.t.Fatalf("set app.user_id: %v", err)
	}
}

// try runs one statement under a savepoint, so a refusal leaves the session
// usable for the next step.
func (s *rvStoreDbSession) try(sql string, args ...any) (pgconn.CommandTag, error) {
	s.t.Helper()
	if _, err := s.tx.Exec(s.ctx, `SAVEPOINT rv_store_db_try`); err != nil {
		s.t.Fatalf("savepoint: %v", err)
	}
	tag, err := s.tx.Exec(s.ctx, sql, args...)
	if err != nil {
		if _, rerr := s.tx.Exec(s.ctx, `ROLLBACK TO SAVEPOINT rv_store_db_try`); rerr != nil {
			s.t.Fatalf("rollback to savepoint: %v", rerr)
		}
		return tag, err
	}
	if _, err := s.tx.Exec(s.ctx, `RELEASE SAVEPOINT rv_store_db_try`); err != nil {
		s.t.Fatalf("release savepoint: %v", err)
	}
	return tag, nil
}

// rvStoreDbOwnerStore opens DATABASE_OWNER_URL (insideout_owner, which RLS does
// not apply to). It skips without the variable, like the other owner-side
// checks in this package.
func rvStoreDbOwnerStore(t *testing.T) *Store {
	t.Helper()
	ownerURL := os.Getenv("DATABASE_OWNER_URL")
	if ownerURL == "" {
		t.Skip("DATABASE_OWNER_URL not set, skipping the owner-side check")
	}
	owner, err := Open(context.Background(), ownerURL)
	if err != nil {
		t.Fatalf("open owner store: %v", err)
	}
	t.Cleanup(owner.Close)
	var bypass bool
	if err := owner.Pool.QueryRow(context.Background(), `SELECT rolbypassrls OR rolsuper FROM pg_roles WHERE rolname = current_user`).Scan(&bypass); err != nil || !bypass {
		t.Fatalf("DATABASE_OWNER_URL role must bypass RLS (bypass=%v, err=%v)", bypass, err)
	}
	return owner
}

// rvStoreDbCrew is a workspace whose creator has promoted a second admin.
type rvStoreDbCrew struct {
	creator, second *User
	ws              *Workspace
}

func rvStoreDbNewCrew(t *testing.T, st *Store) rvStoreDbCrew {
	t.Helper()
	ctx := context.Background()
	c := rvStoreDbCrew{creator: mkUser(t, st), second: mkUser(t, st)}
	var err error
	if c.ws, err = st.CreateWorkspace(ctx, c.creator.ID, "RV4 creator trust", ""); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, c.second.ID, c.ws.Code); err != nil {
		t.Fatalf("second joins: %v", err)
	}
	if err := st.UpdateMemberRole(ctx, c.creator.ID, c.ws.ID, c.second.ID, "admin"); err != nil {
		t.Fatalf("promote second: %v", err)
	}
	return c
}

const rvStoreDbInsertAdmin = `INSERT INTO insideout.workspace_memberships (workspace_id, user_id, role) VALUES ($1, $2, 'admin')`

// F37, F22, F34: an admin cannot turn an existing membership row into someone
// else's, in another workspace, or under another id; the role is the one thing
// an UPDATE may change.
func TestRvStoreDbMembershipUpdate_CannotRepointARow(t *testing.T) {
	st := writeWalkStore(t)
	ctx := context.Background()
	admin, member, outsider := mkUser(t, st), mkUser(t, st), mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "RV4 membership update", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	other, err := st.CreateWorkspace(ctx, admin.ID, "RV4 membership update, second", "")
	if err != nil {
		t.Fatalf("create second workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, member.ID, ws.Code); err != nil {
		t.Fatalf("member joins: %v", err)
	}
	row, err := st.GetMembership(ctx, ws.ID, member.ID)
	if err != nil {
		t.Fatalf("read the member's row: %v", err)
	}

	const upd = `UPDATE insideout.workspace_memberships SET `
	for _, c := range []struct {
		name  string
		actor uuid.UUID
		sql   string
		args  []any
		deny  bool
		rows  int64 // rows a permitted statement changes
	}{
		{"admin re-points a member row at someone who never joined", admin.ID,
			upd + `user_id = $3 WHERE workspace_id = $1 AND user_id = $2`, []any{ws.ID, member.ID, outsider.ID}, true, 0},
		{"admin re-points and promotes in one statement", admin.ID,
			upd + `user_id = $2, role = 'admin' WHERE id = $1`, []any{row.ID, outsider.ID}, true, 0},
		{"admin moves a member row into another workspace they administer", admin.ID,
			upd + `workspace_id = $3 WHERE workspace_id = $1 AND user_id = $2`, []any{ws.ID, member.ID, other.ID}, true, 0},
		{"admin rewrites the row id", admin.ID,
			upd + `id = gen_random_uuid() WHERE id = $1`, []any{row.ID}, true, 0},
		{"admin rewrites created_at", admin.ID,
			upd + `created_at = created_at - interval '1 day' WHERE id = $1`, []any{row.ID}, true, 0},
		{"admin changes the role (the one thing an UPDATE may change)", admin.ID,
			upd + `role = 'admin' WHERE id = $1`, []any{row.ID}, false, 1},
		{"admin writes the same values back", admin.ID,
			upd + `user_id = user_id, workspace_id = workspace_id WHERE id = $1`, []any{row.ID}, false, 1},
		{"member promotes themselves: the row is not theirs to change", member.ID,
			upd + `role = 'admin' WHERE id = $1`, []any{row.ID}, false, 0},
		{"member re-points their own row", member.ID,
			upd + `user_id = $2 WHERE id = $1`, []any{row.ID, outsider.ID}, false, 0},
	} {
		tag, err := asApp(t, st, c.actor, "", c.sql, c.args...)
		switch {
		case c.deny && !denied(err):
			t.Errorf("%s: want a refusal (SQLSTATE 42501), got %v", c.name, err)
		case !c.deny && err != nil:
			t.Errorf("%s: want it allowed, got %v", c.name, err)
		case !c.deny && tag.RowsAffected() != c.rows:
			t.Errorf("%s: changed %d rows, want %d", c.name, tag.RowsAffected(), c.rows)
		}
	}

	// The attempts were rolled back; the roster is what the real calls made.
	list, err := st.ListMembers(ctx, admin.ID, ws.ID)
	if err != nil || len(list) != 2 {
		t.Fatalf("roster after the attempts = %d rows (err %v), want the admin and the member", len(list), err)
	}
	if _, err := st.GetMembership(ctx, ws.ID, outsider.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("outsider has a membership after only rolled-back statements: %v", err)
	}
	// The real path for a role change keeps working in both directions.
	for _, role := range []string{"admin", "member"} {
		if err := st.UpdateMemberRole(ctx, admin.ID, ws.ID, member.ID, role); err != nil {
			t.Errorf("UpdateMemberRole to %s: %v", role, err)
		}
	}
}

// F23: workspaces.creator_id outlives the creator's membership, so the admin
// branch of the INSERT rule must not read it as "this person may come back as
// admin". The admin row is for the first member of a workspace (what
// createWorkspaceTx writes), never for a workspace that already has people.
func TestRvStoreDbCreatorTrust_RemovedCreatorCannotReAdmin(t *testing.T) {
	st := writeWalkStore(t)
	ctx := context.Background()
	for _, way := range []string{"removed by the other admin", "leaves on their own"} {
		t.Run(way, func(t *testing.T) {
			c := rvStoreDbNewCrew(t, st)
			actor := c.second.ID
			if way == "leaves on their own" {
				actor = c.creator.ID
			}
			if err := st.RemoveMember(ctx, actor, c.ws.ID, c.creator.ID); err != nil {
				t.Fatalf("creator out of the workspace: %v", err)
			}
			if _, err := st.GetMembership(ctx, c.ws.ID, c.creator.ID); !errors.Is(err, ErrNotFound) {
				t.Fatalf("creator still belongs: %v", err)
			}
			if _, err := asApp(t, st, c.creator.ID, "", rvStoreDbInsertAdmin, c.ws.ID, c.creator.ID); !denied(err) {
				t.Errorf("a creator who is no longer a member inserting themselves as admin: want a refusal (SQLSTATE 42501), got %v", err)
			}
		})
	}
}

// F33, F23: any admin could rewrite workspaces.creator_id (workspaces_update
// allows creator or admin and has no column limit) and so become the one
// person the INSERT rule and the creator-only delete policy trust.
func TestRvStoreDbCreatorTrust_AdminCannotBecomeCreator(t *testing.T) {
	st := writeWalkStore(t)
	c := rvStoreDbNewCrew(t, st)

	const setCreator = `UPDATE insideout.workspaces SET creator_id = $2 WHERE id = $1`
	if _, err := asApp(t, st, c.second.ID, "", setCreator, c.ws.ID, c.second.ID); !denied(err) {
		t.Errorf("an admin who is not the creator naming themselves creator: want a refusal (SQLSTATE 42501), got %v", err)
	}
	if _, err := asApp(t, st, c.creator.ID, "", setCreator, c.ws.ID, c.second.ID); !denied(err) {
		t.Errorf("the creator handing the creator role to another admin: want a refusal (SQLSTATE 42501), got %v", err)
	}

	// The whole chain in one session: take over creator_id, be removed by the
	// creator, come back as admin. Every step that matters is checked.
	rvStoreDbRun(t, st, func(s *rvStoreDbSession) {
		s.as(c.second.ID)
		if _, err := s.try(setCreator, c.ws.ID, c.second.ID); !denied(err) {
			t.Errorf("chain, step 1 (rewrite creator_id): want a refusal, got %v", err)
		}
		s.as(c.creator.ID)
		if _, err := s.try(`DELETE FROM insideout.workspace_memberships WHERE workspace_id = $1 AND user_id = $2`, c.ws.ID, c.second.ID); err != nil {
			t.Fatalf("chain, step 2 (the creator removes the other admin): %v", err)
		}
		s.as(c.second.ID)
		if _, err := s.try(rvStoreDbInsertAdmin, c.ws.ID, c.second.ID); !denied(err) {
			t.Errorf("chain, step 3 (come back as admin): want a refusal (SQLSTATE 42501), got %v", err)
		}
	})

	// What an admin may still do to the workspace row, including writing the
	// creator back unchanged, which a pin must not mistake for a change.
	if _, err := asApp(t, st, c.second.ID, "",
		`UPDATE insideout.workspaces SET title = title || '!', creator_id = creator_id WHERE id = $1`, c.ws.ID); err != nil {
		t.Errorf("an admin editing the workspace title: %v", err)
	}
}
