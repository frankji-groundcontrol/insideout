package store

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// rv4-DATA-03 (read half): the column-level guard for the one secret users
// carries. insideout_app may SELECT every users column except password_hash.
// A column added later without its own GRANT is red here, in CI, instead of a
// permission-denied error for every reader in production; and password_hash
// cannot quietly become readable again through a table-wide GRANT. The columns
// come from pg_attribute, not information_schema.columns: that view hides a
// column the role holds no privilege on, which is exactly the ungranted column
// this test exists to catch.
func TestUsersReadGrant_Columns(t *testing.T) {
	st := writeWalkStore(t)
	rows, err := st.Pool.Query(context.Background(), `
		SELECT a.attname, has_column_privilege('insideout_app', a.attrelid, a.attnum, 'SELECT')
		FROM pg_attribute a
		WHERE a.attrelid = 'insideout.users'::regclass AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY a.attnum`)
	if err != nil {
		t.Fatalf("list users columns: %v", err)
	}
	defer rows.Close()
	sawSecret := false
	for rows.Next() {
		var col string
		var can bool
		if err := rows.Scan(&col, &can); err != nil {
			t.Fatalf("scan: %v", err)
		}
		switch {
		case col == "password_hash":
			sawSecret = true
			if can {
				t.Errorf("insideout_app can SELECT users.password_hash: revoke the table-wide SELECT and grant the other columns by name (20261018200000_users_read_grants.sql)")
			}
		case !can:
			t.Errorf("insideout_app cannot SELECT users.%s: add it to the GRANT SELECT (...) column list; a users column without its own grant is a permission-denied error for every reader", col)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("list users columns: %v", err)
	}
	if !sawSecret {
		t.Error("control: users.password_hash is not in pg_attribute, so this guard checks nothing")
	}
}

// The definer function is the only door to the credential, so its shape is
// part of the guard: SECURITY DEFINER (it reads what its caller cannot), owned
// by the role that owns users, search_path pinned so nothing a caller creates
// can shadow a name inside it, and EXECUTE for insideout_app and the owner
// only. PUBLIC holds EXECUTE on every new function by default.
func TestUsersReadGrant_CredentialFunction(t *testing.T) {
	st := writeWalkStore(t)
	var secdef, sameOwner, appExec bool
	var owner string
	var cfg, grantees []string
	err := st.Pool.QueryRow(context.Background(), `
		SELECT p.prosecdef, p.proowner = c.relowner, pg_get_userbyid(p.proowner),
		       has_function_privilege('insideout_app', p.oid, 'EXECUTE'),
		       COALESCE(p.proconfig, ARRAY[]::text[]),
		       array(SELECT DISTINCT CASE a.grantee WHEN 0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END AS g
		               FROM aclexplode(COALESCE(p.proacl, acldefault('f', p.proowner))) a
		              WHERE a.privilege_type = 'EXECUTE' ORDER BY 1)
		FROM pg_proc p, pg_class c
		WHERE p.oid = to_regprocedure('insideout._credential_for_email(text)') AND c.oid = 'insideout.users'::regclass`,
	).Scan(&secdef, &sameOwner, &owner, &appExec, &cfg, &grantees)
	if errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("insideout._credential_for_email(text) does not exist: login has no way to read the credential once password_hash is not selectable")
	}
	if err != nil {
		t.Fatalf("inspect insideout._credential_for_email: %v", err)
	}
	if !secdef {
		t.Error("the function must be SECURITY DEFINER: it reads the column its caller cannot")
	}
	if !sameOwner {
		t.Errorf("the function is owned by %s, not by the role that owns insideout.users: its definer rights would not reach the column", owner)
	}
	if !slices.Contains(cfg, "search_path=insideout, pg_temp") {
		t.Errorf("the function must pin search_path to insideout, pg_temp, got %v", cfg)
	}
	if !appExec {
		t.Error("insideout_app cannot EXECUTE the function, so login cannot read the credential")
	}
	for _, g := range grantees {
		if g != "insideout_app" && g != owner {
			t.Errorf("EXECUTE is also granted to %s: REVOKE ALL ON FUNCTION ... FROM PUBLIC, then grant insideout_app only", g)
		}
	}
}

// Callers moved to the definer function: login's reader (GetUserByEmail) is
// the only store method that carries the stored credential. Every other users
// reader, and every RETURNING, leaves PasswordHash empty.
func TestUsersReadGrant_OnlyLoginCarriesCredential(t *testing.T) {
	st := writeWalkStore(t)
	ctx := context.Background()
	hash := "grant-hash-" + uuid.NewString()
	created, err := st.CreateUser(ctx, "grant-"+uuid.NewString()+"@example.com", hash, "grant")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	byEmail, err := st.GetUserByEmail(ctx, created.Email)
	if err != nil || byEmail.PasswordHash != hash {
		t.Errorf("GetUserByEmail: want the stored credential, got err=%v matches=%v", err, err == nil && byEmail.PasswordHash == hash)
	}
	byID, err := st.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	updated, err := st.UpdateProfile(ctx, created.ID, "grant", "bio", []string{}, nil)
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	for name, u := range map[string]*User{"CreateUser": created, "GetUserByID": byID, "UpdateProfile": updated} {
		if u.PasswordHash != "" {
			t.Errorf("%s returned the stored credential: only GetUserByEmail may carry it", name)
		}
	}
}
