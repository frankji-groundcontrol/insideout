package store

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// rv4-DATA-03 (read half). The write walk (rls_write_walk_test.go) closed what
// the app may write to users; this walk closes what it may read. users_select
// shows a user every co-member's row and insideout_app held table-wide SELECT,
// so any co-member's password_hash was one SELECT away at the database layer.
// No API response carries it, which is why TestRLSWalk (it counts rows) and the
// HTTP tests never saw it. Every statement below runs as insideout_app, as the
// actor named, in a transaction that is always rolled back, and every
// read-shaped way to the column must be refused by the database itself
// (SQLSTATE 42501): naming it, SELECT *, a whole-row reference, a join, a
// subquery, a WHERE oracle, RETURNING, and an UPDATE that copies it into a
// column anyone can read. The controls keep the refusal honest: it is about
// the column, so the other columns and the workspace scoping of rows are
// unchanged, and the stored credential is reachable through one named helper.

func TestRLSReadWalk_UsersCredential(t *testing.T) {
	st := writeWalkStore(t)
	ctx := context.Background()
	viewer, target, outsider := mkUser(t, st), mkUser(t, st), mkUser(t, st)

	// viewer and target share a workspace, so users_select shows each the
	// other's row; outsider shares none.
	ws, err := st.CreateWorkspace(ctx, viewer.ID, "RLS read walk", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, target.ID, ws.Code); err != nil {
		t.Fatalf("join workspace: %v", err)
	}

	const insertUser = `INSERT INTO insideout.users (email, password_hash, username) VALUES ($1, 'x', 'walk')`
	fresh := "walk-" + uuid.NewString() + "@example.com"
	for _, c := range []struct {
		name  string
		actor uuid.UUID
		sql   string
		arg   any
	}{
		{"column read of a co-member", viewer.ID, `SELECT password_hash FROM insideout.users WHERE id = $1`, target.ID},
		{"column read of your own row", viewer.ID, `SELECT password_hash FROM insideout.users WHERE id = $1`, viewer.ID},
		{"column read with no actor (the pre-auth context)", uuid.Nil, `SELECT password_hash FROM insideout.users WHERE email = $1`, target.Email},
		{"SELECT *", viewer.ID, `SELECT * FROM insideout.users WHERE id = $1`, target.ID},
		{"whole-row reference", viewer.ID, `SELECT to_jsonb(u) FROM insideout.users u WHERE u.id = $1`, target.ID},
		{"join through memberships", viewer.ID, `SELECT m.workspace_id, u.password_hash FROM insideout.workspace_memberships m JOIN insideout.users u ON u.id = m.user_id WHERE u.id = $1`, target.ID},
		{"subquery", viewer.ID, `SELECT (SELECT password_hash FROM insideout.users WHERE id = $1)`, target.ID},
		{"WHERE oracle", viewer.ID, `SELECT 1 FROM insideout.users WHERE id = $1 AND password_hash LIKE '%'`, target.ID},
		{"INSERT ... RETURNING the column", uuid.Nil, insertUser + ` RETURNING password_hash`, fresh},
		{"INSERT ... RETURNING *", uuid.Nil, insertUser + ` RETURNING *`, fresh},
		{"UPDATE ... RETURNING the column", viewer.ID, `UPDATE insideout.users SET bio = bio WHERE id = $1 RETURNING password_hash`, viewer.ID},
		{"UPDATE copying the column into bio", viewer.ID, `UPDATE insideout.users SET bio = password_hash WHERE id = $1`, viewer.ID},
	} {
		tag, err := asApp(t, st, c.actor, "", c.sql, c.arg)
		if !denied(err) {
			t.Errorf("%s: want a refusal (SQLSTATE 42501), got rows=%d err=%v", c.name, tag.RowsAffected(), err)
		}
	}

	// Control: every other column of the target's row stays readable to a
	// co-member and stays invisible to an outsider, so the refusals above are
	// about password_hash and not about rows or the table.
	const readable = `SELECT id, email, username, avatar_url, bio, keywords, role, system_role, email_verified_at, meta, created_at, updated_at FROM insideout.users WHERE id = $1`
	for _, c := range []struct {
		who   string
		actor uuid.UUID
		rows  int64
	}{{"a co-member", viewer.ID, 1}, {"an outsider", outsider.ID, 0}} {
		tag, err := asApp(t, st, c.actor, "", readable, target.ID)
		if err != nil || tag.RowsAffected() != c.rows {
			t.Errorf("control: the non-secret columns of the target's row as %s: want %d rows and no error, got rows=%d err=%v", c.who, c.rows, tag.RowsAffected(), err)
		}
	}
}

// The credential still has to reach login, through exactly one named door:
// insideout._credential_for_email answers in the pre-auth context (no
// app.user_id) for a known email, and in no other.
func TestRLSReadWalk_CredentialHelper(t *testing.T) {
	st := writeWalkStore(t)
	ctx := context.Background()
	email, hash := "walk-"+uuid.NewString()+"@example.com", "walk-hash-"+uuid.NewString()
	u, err := st.CreateUser(ctx, email, hash, "walk")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	const call = `SELECT id, password_hash FROM insideout._credential_for_email($1)`
	var id uuid.UUID
	var got string
	if err := st.Pool.QueryRow(ctx, call, email).Scan(&id, &got); err != nil || id != u.ID || got != hash {
		t.Errorf("helper for a known email, no actor: want the user's id and stored credential, got id=%s matches=%v err=%v", id, got == hash, err)
	}
	if err := st.Pool.QueryRow(ctx, call, "nobody-"+uuid.NewString()+"@example.com").Scan(&id, &got); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("helper for an unknown email: want no row, got err=%v", err)
	}
	if tag, err := asApp(t, st, u.ID, "", call, email); err != nil || tag.RowsAffected() != 0 {
		t.Errorf("helper with an actor set (inside a request's own transaction): want no rows, got rows=%d err=%v", tag.RowsAffected(), err)
	}
}
