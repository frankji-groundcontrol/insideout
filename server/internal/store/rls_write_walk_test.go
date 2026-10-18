package store

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// rv4-DATA-02: TestRLSWalk (rls_walk_test.go) only reads, and a policy that lets
// the wrong INSERT or UPDATE through is invisible to its SELECT counts. That is
// how workspace_memberships_insert (rv4-DATA-01) and the users columns
// (rv4-DATA-03) stayed open. This walk attacks the write half: every statement
// runs as insideout_app, as the actor named, in a transaction that is always
// rolled back, and the case says what the database has to answer:
//
//  1. membership INSERTs: an admin row only for the workspace creator, a member
//     row only with the join code, always for yourself;
//  2. users INSERT and UPDATE: only the columns the app really writes, column
//     by column, with a no-actor UPDATE still matching no row (promote-admin's
//     DATABASE_OWNER_URL hint reads that "no row" answer);
//  3. UPDATE and DELETE on every RLS table (from the catalog, like the read
//     walk): an outsider and an actor-less session change no row.

// asApp runs one statement as insideout_app. actor sets app.user_id (uuid.Nil
// leaves it unset: the pre-auth context) and code sets app.join_code when it
// is not empty. The transaction is rolled back whatever the statement did.
func asApp(t *testing.T, st *Store, actor uuid.UUID, code, sql string, args ...any) (pgconn.CommandTag, error) {
	t.Helper()
	ctx := context.Background()
	tx, err := st.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)
	if actor != uuid.Nil {
		if _, err := tx.Exec(ctx, `SELECT set_config('app.user_id', $1, true)`, actor.String()); err != nil {
			t.Fatalf("set app.user_id: %v", err)
		}
	}
	if code != "" {
		if _, err := tx.Exec(ctx, `SELECT set_config('app.join_code', $1, true)`, code); err != nil {
			t.Fatalf("set app.join_code: %v", err)
		}
	}
	return tx.Exec(ctx, sql, args...)
}

// denied reports a refusal by the database itself: SQLSTATE 42501, whether it
// comes from a policy's WITH CHECK, a missing column privilege or a pinning
// trigger.
func denied(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42501"
}

// writeWalkStore is newTestStore plus the walk's precondition: the pool role
// must be one RLS applies to, or every "refused" below proves nothing.
func writeWalkStore(t *testing.T) *Store {
	t.Helper()
	st := newTestStore(t)
	var exempt bool
	if err := st.Pool.QueryRow(context.Background(), `SELECT rolbypassrls OR rolsuper FROM pg_roles WHERE rolname = current_user`).Scan(&exempt); err != nil || exempt {
		t.Fatalf("DATABASE_URL must connect as a role subject to RLS (exempt=%v, err=%v)", exempt, err)
	}
	return st
}

func TestRLSWriteWalk_MembershipInsert(t *testing.T) {
	st := writeWalkStore(t)
	ctx := context.Background()
	owner, attacker, plain, friend := mkUser(t, st), mkUser(t, st), mkUser(t, st), mkUser(t, st)

	// createWorkspaceTx is the legitimate admin insert: the creator of a
	// workspace becomes its first admin. attacker runs a workspace of their
	// own, which is no claim on the victim's.
	victim, err := st.CreateWorkspace(ctx, owner.ID, "RLS write walk", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.CreateWorkspace(ctx, attacker.ID, "RLS write walk attacker", ""); err != nil {
		t.Fatalf("create attacker workspace: %v", err)
	}

	const insert = `INSERT INTO insideout.workspace_memberships (workspace_id, user_id, role) VALUES ($1, $2, $3)`
	for _, c := range []struct {
		name  string
		actor uuid.UUID
		code  string
		who   uuid.UUID
		role  string
		deny  bool
	}{
		{"bare self-insert as admin", plain.ID, "", plain.ID, "admin", true},
		{"bare self-insert as member", plain.ID, "", plain.ID, "member", true},
		{"self-insert as member, wrong join code", plain.ID, "not-the-code", plain.ID, "member", true},
		{"self-insert as admin, right join code", plain.ID, victim.Code, plain.ID, "admin", true},
		{"admin of another workspace, bare, as admin", attacker.ID, "", attacker.ID, "admin", true},
		{"adding someone else as member, right join code", plain.ID, victim.Code, friend.ID, "member", true},
		{"the creator adding someone else as admin", owner.ID, "", friend.ID, "admin", true},
		{"self-insert as member, right join code", plain.ID, victim.Code, plain.ID, "member", false},
	} {
		_, err := asApp(t, st, c.actor, c.code, insert, victim.ID, c.who, c.role)
		switch {
		case c.deny && !denied(err):
			t.Errorf("%s: want a refusal (SQLSTATE 42501), got %v", c.name, err)
		case !c.deny && err != nil:
			t.Errorf("%s: want it allowed, got %v", c.name, err)
		}
	}

	// The refused inserts changed nothing, and the real join path still works.
	if _, err := st.JoinWorkspace(ctx, friend.ID, victim.Code); err != nil {
		t.Errorf("JoinWorkspace with the right code: %v", err)
	}
	if m, err := st.GetMembership(ctx, victim.ID, plain.ID); err == nil {
		t.Errorf("plain has a %s membership in the victim workspace after only rolled-back inserts", m.Role)
	}
}

// userColumnWrites lists every users column the app may write or may not, with
// an assignment that really changes the stored value (a no-op would pass for a
// refusal). updated_at is left out: set_updated_at maintains it.
var userColumnWrites = map[string]struct {
	set string
	ok  bool
}{
	"username":          {`username = username || '!'`, true},
	"bio":               {`bio = bio || '!'`, true},
	"keywords":          {`keywords = ARRAY['walk']`, true},
	"avatar_url":        {`avatar_url = 'https://example.com/walk.png'`, true},
	"password_hash":     {`password_hash = password_hash || '!'`, true},
	"id":                {`id = gen_random_uuid()`, false},
	"email":             {`email = 'rewritten-' || email`, false},
	"role":              {`role = CASE role WHEN 'admin' THEN 'user' ELSE 'admin' END`, false},
	"system_role":       {`system_role = CASE system_role WHEN 'admin' THEN 'member' ELSE 'admin' END`, false},
	"email_verified_at": {`email_verified_at = now()`, false},
	"meta":              {`meta = '{"walk":1}'::jsonb`, false},
	"created_at":        {`created_at = created_at - interval '1 day'`, false},
}

// userInsertExtras are columns an INSERT must not be able to set: CreateUser
// writes email, password_hash and username, and nothing else.
var userInsertExtras = map[string]string{
	"id": `gen_random_uuid()`, "role": `'admin'`, "system_role": `'admin'`,
	"email_verified_at": `now()`, "meta": `'{"walk":1}'::jsonb`, "created_at": `now()`,
}

func TestRLSWriteWalk_UserColumns(t *testing.T) {
	st := writeWalkStore(t)
	ctx := context.Background()
	me, other := mkUser(t, st), mkUser(t, st)

	// A new users column is red until someone decides whether the app writes it.
	rows, err := st.Pool.Query(ctx, `SELECT column_name FROM information_schema.columns WHERE table_schema = 'insideout' AND table_name = 'users'`)
	if err != nil {
		t.Fatalf("list users columns: %v", err)
	}
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if _, ok := userColumnWrites[col]; !ok && col != "updated_at" {
			t.Errorf("users.%s is not in userColumnWrites: decide whether insideout_app may write it", col)
		}
	}
	rows.Close()

	for col, w := range userColumnWrites {
		tag, err := asApp(t, st, me.ID, "", `UPDATE insideout.users SET `+w.set+` WHERE id = $1`, me.ID)
		switch {
		case w.ok && (err != nil || tag.RowsAffected() != 1):
			t.Errorf("UPDATE of own %s: want it allowed, got rows=%d err=%v", col, tag.RowsAffected(), err)
		case !w.ok && !denied(err):
			t.Errorf("UPDATE of own %s: want a refusal (SQLSTATE 42501), got %v", col, err)
		}
	}

	// Another user's row, and a session with no actor at all, match no row:
	// an answer, not an error, because promote-admin without the owner pool
	// turns "no row" into its set-DATABASE_OWNER_URL hint.
	for who, actor := range map[string]uuid.UUID{"another user": me.ID, "no actor": uuid.Nil} {
		tag, err := asApp(t, st, actor, "", `UPDATE insideout.users SET system_role = 'admin', updated_at = now() WHERE email = $1`, other.Email)
		if err != nil || tag.RowsAffected() != 0 {
			t.Errorf("system_role UPDATE of someone else's row as %s: want 0 rows and no error, got rows=%d err=%v", who, tag.RowsAffected(), err)
		}
	}

	email := func() string { return "walk-" + uuid.NewString() + "@example.com" }
	for col, val := range userInsertExtras {
		_, err := asApp(t, st, uuid.Nil, "", `INSERT INTO insideout.users (email, password_hash, username, `+col+`) VALUES ($1, 'x', 'walk', `+val+`)`, email())
		if !denied(err) {
			t.Errorf("INSERT of a user with %s set: want a refusal (SQLSTATE 42501), got %v", col, err)
		}
	}
	if _, err := asApp(t, st, uuid.Nil, "", `INSERT INTO insideout.users (email, password_hash, username) VALUES ($1, 'x', 'walk')`, email()); err != nil {
		t.Errorf("INSERT of a user with email, password_hash and username (CreateUser): %v", err)
	}
}

// TestRLSWriteWalk_NoRowChanges is the write twin of TestRLSWalk's zero-row
// checks: on every RLS table an outsider and an actor-less session change no
// row. A member's run of the same statements is the control that they can
// change rows at all.
func TestRLSWriteWalk_NoRowChanges(t *testing.T) {
	st := writeWalkStore(t)
	tables := rlsCatalog(t, st)
	member, outsider := seedRLSWalk(t, st)

	first := map[string]string{} // any column to assign to itself; proposal_decisions has no id
	rows, err := st.Pool.Query(context.Background(), `
		SELECT DISTINCT ON (table_name) table_name, column_name FROM information_schema.columns
		WHERE table_schema = 'insideout' ORDER BY table_name, ordinal_position`)
	if err != nil {
		t.Fatalf("list columns: %v", err)
	}
	for rows.Next() {
		var table, col string
		if err := rows.Scan(&table, &col); err != nil {
			t.Fatalf("scan: %v", err)
		}
		first[table] = col
	}
	rows.Close()

	changed := map[string]int{} // rows the member's statements changed, per verb
	for _, name := range tables {
		table, col := pgx.Identifier{"insideout", name}.Sanitize(), pgx.Identifier{first[name]}.Sanitize()
		for verb, sql := range map[string]string{
			"UPDATE": `UPDATE ` + table + ` SET ` + col + ` = ` + col,
			"DELETE": `DELETE FROM ` + table,
		} {
			if tag, err := asApp(t, st, member.ID, "", sql); err == nil {
				changed[verb] += int(tag.RowsAffected())
			}
			for who, actor := range map[string]uuid.UUID{"an outsider": outsider.ID, "no actor": uuid.Nil} {
				switch {
				case name == "ai_runs" && actor == uuid.Nil:
					continue // the stale-run reaper's deliberate actor-less policy
				case name == "users" && verb == "UPDATE" && actor != uuid.Nil:
					continue // an outsider may edit their own row
				}
				tag, err := asApp(t, st, actor, "", sql)
				if err != nil && !denied(err) {
					t.Errorf("%s on %s as %s: want 0 rows or a refusal, got %v", verb, name, who, err)
				}
				if err == nil && tag.RowsAffected() != 0 {
					t.Errorf("%s on %s as %s changed %d rows, want 0", verb, name, who, tag.RowsAffected())
				}
			}
		}
	}
	for _, verb := range []string{"UPDATE", "DELETE"} {
		if changed[verb] == 0 {
			t.Errorf("control: a member's %s changed no row anywhere, so the zero-row checks above prove nothing", verb)
		}
	}
}
