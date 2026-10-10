//go:build roleless

package main

// Both roles exist on the cluster (a first database was provisioned earlier),
// but the database under test is not that one. The migrations need exactly one
// database-level right — CREATE, so that insideout_owner can create the
// insideout schema, and only while the schema is missing — so setup must stop
// before migrating exactly when that right is missing, and must not stop the
// shared-instance layout (docs/architecture/database-and-rls.md), where an
// admin pre-creates the schema in a database somebody else owns.

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	rlOwnerPW = "ownerpw0003"
	rlAppPW   = "apppw0003"
)

// rolesFromFirstDatabase provisions the roles the way the bundled init script
// does, on a throwaway first database, so they exist cluster-wide before the
// database under test is touched.
func (c *rlCluster) rolesFromFirstDatabase(t *testing.T) {
	t.Helper()
	c.operatorProvision(t, c.newDatabase(t, ""), rlOwnerPW, rlAppPW)
}

// execIn runs statements as the cluster superuser connected to db.
func (c *rlCluster) execIn(t *testing.T, db string, stmts ...string) {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), c.dsn(db, "", ""))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, s := range stmts {
		if _, err := pool.Exec(context.Background(), s); err != nil {
			t.Fatalf("%s: %v", strings.SplitN(s, " ", 3)[0], err)
		}
	}
}

func (c *rlCluster) scalar(t *testing.T, sql string, args ...any) string {
	t.Helper()
	var s string
	if err := c.admin.QueryRow(context.Background(), sql, args...).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func (c *rlCluster) databaseOwner(t *testing.T, db string) string {
	t.Helper()
	return c.scalar(t, `SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname = $1`, db)
}

// printedSQL returns the statements a stop message printed: two-space-indented
// lines ending in ';', without the ';'.
func printedSQL(out string) []string {
	var stmts []string
	for _, line := range strings.Split(out, "\n") {
		if s, ok := strings.CutPrefix(line, "  "); ok && !strings.HasPrefix(s, " ") && strings.HasSuffix(s, ";") {
			stmts = append(stmts, strings.TrimSuffix(s, ";"))
		}
	}
	return stmts
}

// splitFixes separates the two fixes a stop message offers: the statements
// that hand the database to the roles, and the one that pre-creates the schema.
func splitFixes(out string) (dedicated, schema []string) {
	for _, s := range printedSQL(out) {
		if strings.HasPrefix(s, "CREATE SCHEMA") {
			schema = append(schema, s)
		} else {
			dedicated = append(dedicated, s)
		}
	}
	return dedicated, schema
}

// Roles present, but insideout_owner can neither own nor create in this
// database and the schema is missing: the migration would die with "permission
// denied for database". Setup must stop before migrating, say whose database
// it is, print the fix, and leave the database untouched — even with
// --create-roles, which is consent to create roles, not to change owners. The
// printed statements must then actually fix it.
func TestRolelessNotReadyDatabaseStopsBeforeMigrating(t *testing.T) {
	c := newRolelessCluster(t)
	c.rolesFromFirstDatabase(t)
	super := c.scalar(t, `SELECT current_user`)
	other, _ := c.newLoginRole(t, "")
	for _, tt := range []struct {
		name, dbOwner, wantOwner, as string // as: the connecting role, "" = the superuser
	}{
		{"database made by a superuser", "", super, ""},
		{"database made by another role", other, other, ""},
		{"connecting as insideout_owner", "", super, "insideout_owner"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db := c.newDatabase(t, tt.dbOwner)
			dsn := c.dsn(db, tt.as, rlOwnerPW)
			r := newRlRun(t)

			var printed string
			for _, args := range [][]string{nil, {"--create-roles"}} {
				code, out := r.run(t, dsn, args...)
				if code != 1 {
					t.Fatalf("setup %v exit = %d, want 1\n%s", args, code, out)
				}
				for _, want := range []string{
					`is owned by "` + tt.wantOwner + `"`,
					`ALTER DATABASE "` + db + `" OWNER TO insideout_owner`,
					`GRANT CONNECT ON DATABASE "` + db + `" TO insideout_app`,
					`CREATE SCHEMA insideout AUTHORIZATION insideout_owner`,
				} {
					if !strings.Contains(out, want) {
						t.Errorf("setup %v output lacks %q:\n%s", args, want, out)
					}
				}
				for _, bad := range []string{"CREATE ROLE", "PASSWORD"} {
					if strings.Contains(out, bad) {
						t.Errorf("setup %v printed %q although the roles exist:\n%s", args, bad, out)
					}
				}
				rlNoSecrets(t, out, rlOwnerPW, rlAppPW)
				if c.schemaExists(t, db) {
					t.Fatalf("setup %v migrated although the database was not ready", args)
				}
				if got := c.databaseOwner(t, db); got != tt.wantOwner {
					t.Fatalf("setup %v changed the database owner to %q", args, got)
				}
				printed = out
			}

			dedicated, _ := splitFixes(printed)
			if len(dedicated) != 4 {
				t.Fatalf("want the 4 database statements of the init script, got %q", dedicated)
			}
			c.execIn(t, db, dedicated...)
			code, out := r.run(t, dsn)
			if code != 0 || !strings.Contains(out, "migrations applied (") {
				t.Fatalf("after the printed SQL: exit = %d, want a clean migration\n%s", code, out)
			}
			if got := c.databaseOwner(t, db); got != "insideout_owner" {
				t.Errorf("database owner = %q after the fix, want insideout_owner", got)
			}
		})
	}
}

// The shared-instance fix: a database that must keep its owner gets the schema
// pre-created instead, and setup then migrates without changing the owner.
func TestRolelessNotReadyDatabaseSchemaFixKeepsTheOwner(t *testing.T) {
	c := newRolelessCluster(t)
	c.rolesFromFirstDatabase(t)
	super := c.scalar(t, `SELECT current_user`)
	db := c.newDatabase(t, "")
	r := newRlRun(t)

	code, out := r.run(t, c.dsn(db, "", ""))
	if code != 1 {
		t.Fatalf("setup exit = %d, want 1\n%s", code, out)
	}
	_, schema := splitFixes(out)
	if len(schema) != 1 {
		t.Fatalf("want one schema statement, got %q\n%s", schema, out)
	}
	c.execIn(t, db, schema...)
	code, out = r.run(t, c.dsn(db, "", ""))
	if code != 0 || !strings.Contains(out, "migrations applied (") {
		t.Fatalf("after the printed schema statement: exit = %d, want a clean migration\n%s", code, out)
	}
	if got := c.databaseOwner(t, db); got != super {
		t.Errorf("database owner = %q, want it unchanged (%q)", got, super)
	}
}

// What must NOT stop. The dedicated layout, including a second database made
// for the roles; and the databases insideout_owner does not own but can still
// migrate: the shared instance, where an admin pre-created the schema, and a
// database where CREATE was granted to it. Stopping these would break working
// deployments.
func TestRolelessReadyDatabasesAreNotStopped(t *testing.T) {
	c := newRolelessCluster(t)
	c.rolesFromFirstDatabase(t)
	super := c.scalar(t, `SELECT current_user`)
	for _, tt := range []struct {
		name, dbOwner, wantOwner string
		prep                     func(db string) string // run as the superuser before setup
	}{
		{"second database owned by insideout_owner", "insideout_owner", "insideout_owner", nil},
		{"shared instance: schema pre-created, owner is someone else", "", super,
			func(string) string { return `CREATE SCHEMA insideout AUTHORIZATION insideout_owner` }},
		{"CREATE granted to insideout_owner, owner is someone else", "", super,
			func(db string) string { return `GRANT CREATE ON DATABASE "` + db + `" TO insideout_owner` }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db := c.newDatabase(t, tt.dbOwner)
			if tt.prep != nil {
				c.execIn(t, db, tt.prep(db))
			}
			code, out := newRlRun(t).run(t, c.dsn(db, "", ""))
			if code != 0 || !strings.Contains(out, "migrations applied (") {
				t.Fatalf("setup exit = %d, want a clean migration\n%s", code, out)
			}
			if !strings.Contains(out, "present") || strings.Contains(out, "is owned by") {
				t.Errorf("output should report the roles present and no ownership problem:\n%s", out)
			}
			if got := c.databaseOwner(t, db); got != tt.wantOwner {
				t.Errorf("database owner = %q, want %q", got, tt.wantOwner)
			}
		})
	}
}
