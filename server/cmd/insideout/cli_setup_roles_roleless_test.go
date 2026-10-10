//go:build roleless

package main

// End-to-end scenarios for the own-URL role precondition
// (docs/issues/2026-10-07-own-url-fresh-db-bootstrap.md), driven through
// runSetupCommand against a real PostgreSQL cluster that has NOT been
// provisioned. They create and drop their own throwaway databases and
// login roles — and insideout_owner / insideout_app themselves, which are
// cluster-global names — so they cannot share a cluster with other work.
// The build tag excludes them from the normal suite instead of a t.Skip;
// once selected they refuse to run (Fatal) against anything but a
// superuser connection to a cluster that has neither insideout role.
//
//	go test -tags roleless ./cmd/insideout/ -run Roleless -count=1 -v
//
// INSIDEOUT_ROLELESS_URL names the superuser DSN of a throwaway cluster.
// It defaults to the private scratch cluster of the practice runbook
// (127.0.0.1:5612, trust auth, no password).

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Scenario (a): fresh cluster, superuser DSN, no roles. With consent given
// by flag (stdin is the DSN, so there is no prompt), setup provisions the
// roles per the init script, wires .env, migrates as the new owner, and
// prints no password or DSN.
func TestRolelessProvisionsPerInitScriptAndMigrates(t *testing.T) {
	c := newRolelessCluster(t)
	db := c.newDatabase(t, "")
	r := newRlRun(t)

	code, out := r.run(t, c.dsn(db, "", ""), "--create-roles")
	if code != 0 {
		t.Fatalf("setup exit = %d, want 0\n%s", code, out)
	}

	ctx := context.Background()
	for _, tt := range []struct {
		role   string
		bypass bool
	}{{"insideout_owner", true}, {"insideout_app", false}} {
		var super, login, bypass bool
		if err := c.admin.QueryRow(ctx, `SELECT rolsuper, rolcanlogin, rolbypassrls FROM pg_roles WHERE rolname = $1`, tt.role).Scan(&super, &login, &bypass); err != nil {
			t.Fatalf("%s missing after setup: %v", tt.role, err)
		}
		if super || !login || bypass != tt.bypass {
			t.Errorf("%s: super=%v login=%v bypassrls=%v, want false/true/%v", tt.role, super, login, bypass, tt.bypass)
		}
	}
	var dbOwner string
	var appConnect, appCreate bool
	if err := c.admin.QueryRow(ctx, `SELECT pg_get_userbyid(datdba), has_database_privilege('insideout_app', $1::text, 'CONNECT'), has_database_privilege('insideout_app', $1::text, 'CREATE') FROM pg_database WHERE datname = $1::text`, db).Scan(&dbOwner, &appConnect, &appCreate); err != nil {
		t.Fatal(err)
	}
	if dbOwner != "insideout_owner" || !appConnect || appCreate {
		t.Errorf("database owner=%q app CONNECT=%v CREATE=%v, want insideout_owner/true/false", dbOwner, appConnect, appCreate)
	}

	vals := readEnvValues(r.env)
	ownerPW, appPW := vals["POSTGRES_OWNER_PASSWORD"], vals["POSTGRES_APP_PASSWORD"]
	if len(ownerPW) < 32 || len(appPW) < 32 || ownerPW == appPW {
		t.Fatalf("generated passwords not stored as distinct strong values (%d, %d chars)", len(ownerPW), len(appPW))
	}
	for key, want := range map[string]struct{ user, pw string }{
		"DATABASE_URL":       {"insideout_app", appPW},
		"DATABASE_OWNER_URL": {"insideout_owner", ownerPW},
	} {
		u, err := url.Parse(vals[key])
		if err != nil {
			t.Fatalf("%s is not a URL", key)
		}
		pw, _ := u.User.Password()
		if u.User.Username() != want.user || pw != want.pw || strings.TrimPrefix(u.Path, "/") != db {
			t.Errorf("%s: user=%q db=%q (password matches=%v), want %s on %s", key, u.User.Username(), u.Path, pw == want.pw, want.user, db)
		}
	}
	rlNoSecrets(t, out, ownerPW, appPW, vals["DATABASE_URL"], vals["DATABASE_OWNER_URL"])

	// The migrations really ran, as the provisioned owner.
	owner, err := pgxpool.New(ctx, vals["DATABASE_OWNER_URL"])
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	var applied int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM insideout.schema_migrations`).Scan(&applied); err != nil || applied == 0 {
		t.Fatalf("no migrations recorded (n=%d, err=%v)", applied, err)
	}
	var tableOwner string
	if err := owner.QueryRow(ctx, `SELECT tableowner FROM pg_tables WHERE schemaname = 'insideout' AND tablename = 'workspace_memberships'`).Scan(&tableOwner); err != nil || tableOwner != "insideout_owner" {
		t.Fatalf("workspace_memberships owner = %q (err=%v), want insideout_owner", tableOwner, err)
	}
}

// A piped run without --create-roles must neither prompt nor provision: it
// stops before migrating and prints the SQL with placeholders only.
func TestRolelessPipedWithoutConsentStopsWithSQL(t *testing.T) {
	c := newRolelessCluster(t)
	db := c.newDatabase(t, "")
	r := newRlRun(t)

	code, out := r.run(t, c.dsn(db, "", ""))
	if code != 1 {
		t.Fatalf("setup exit = %d, want 1\n%s", code, out)
	}
	for _, want := range []string{"CREATE ROLE insideout_owner", "<owner-password>", "<app-password>", "ALTER DATABASE \"" + db + "\" OWNER TO insideout_owner", "--create-roles"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if c.roleExists(t, "insideout_owner") || c.roleExists(t, "insideout_app") {
		t.Error("roles were created without consent")
	}
	if c.schemaExists(t, db) {
		t.Error("migrations ran although the roles were missing")
	}
	if vals := readEnvValues(r.env); vals["POSTGRES_OWNER_PASSWORD"] != "" || vals["DATABASE_OWNER_URL"] != "" {
		t.Error("credentials were written although nothing was provisioned")
	}
}

// Scenario (b): roles absent and the connecting role cannot create them.
// Setup stops before migrating, prints the SQL, and changes nothing — also
// for a role that looks capable (CREATEROLE, CREATEDB, BYPASSRLS, owns the
// database) but cannot hand out ownership: the attempt is rolled back whole.
func TestRolelessInsufficientPrivilegeStopsBeforeMigrating(t *testing.T) {
	for _, tt := range []struct{ name, attrs string }{
		{"ordinary role", ""},
		{"capable-looking role", "CREATEROLE CREATEDB BYPASSRLS"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := newRolelessCluster(t)
			user, pw := c.newLoginRole(t, tt.attrs)
			db := c.newDatabase(t, user)
			r := newRlRun(t)

			code, out := r.run(t, c.dsn(db, user, pw), "--create-roles")
			if code != 1 {
				t.Fatalf("setup exit = %d, want 1\n%s", code, out)
			}
			for _, want := range []string{"CREATE ROLE insideout_owner", "<owner-password>", "superuser"} {
				if !strings.Contains(out, want) {
					t.Errorf("output lacks %q:\n%s", want, out)
				}
			}
			rlNoSecrets(t, out, pw)
			if c.roleExists(t, "insideout_owner") || c.roleExists(t, "insideout_app") {
				t.Error("a failed provisioning attempt left roles behind")
			}
			if c.schemaExists(t, db) {
				t.Error("migrations ran although the roles were missing")
			}
		})
	}
}

// Scenario (c): the roles already exist (an operator provisioned them) —
// behaviour is unchanged: no provisioning, migrations proceed.
func TestRolelessRolesPresentIsUnchanged(t *testing.T) {
	c := newRolelessCluster(t)
	db := c.newDatabase(t, "")
	c.operatorProvision(t, db, "ownerpw0001", "apppw0001")

	t.Run("superuser dsn migrates through SET LOCAL ROLE", func(t *testing.T) {
		r := newRlRun(t)
		code, out := r.run(t, c.dsn(db, "", ""))
		if code != 0 {
			t.Fatalf("setup exit = %d, want 0\n%s", code, out)
		}
		if vals := readEnvValues(r.env); vals["POSTGRES_OWNER_PASSWORD"] != "" || vals["DATABASE_OWNER_URL"] != "" {
			t.Error("setup wrote generated credentials although the roles were already present")
		}
	})

	t.Run("--create-roles is a no-op when the roles exist", func(t *testing.T) {
		r := newRlRun(t)
		code, out := r.run(t, c.dsn(db, "", ""), "--create-roles")
		if code != 0 {
			t.Fatalf("setup exit = %d, want 0\n%s", code, out)
		}
		if vals := readEnvValues(r.env); vals["POSTGRES_OWNER_PASSWORD"] != "" || vals["DATABASE_OWNER_URL"] != "" {
			t.Error("setup rewrote credentials although the roles were already present")
		}
	})

	t.Run("app dsn with the owner dsn already in .env", func(t *testing.T) {
		r := newRlRun(t)
		ownerURL := c.dsn(db, "insideout_owner", "ownerpw0001")
		if err := os.WriteFile(r.env, []byte("DATABASE_URL=change_me_db\nINSIDEOUT_JWT_SECRET=change_me_to_a_long_random_string\nDATABASE_OWNER_URL="+ownerURL+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		code, out := r.run(t, c.dsn(db, "insideout_app", "apppw0001"))
		if code != 0 {
			t.Fatalf("setup exit = %d, want 0\n%s", code, out)
		}
		rlNoSecrets(t, out, "ownerpw0001", "apppw0001")
		if !strings.Contains(out, "migrations applied (") {
			t.Errorf("migrations did not run:\n%s", out)
		}
	})
}

// Only one of the two roles exists: its password is unknown to setup, so it
// must not guess — stop before migrating, point at the operator script.
func TestRolelessPartialRolesStopBeforeMigrating(t *testing.T) {
	c := newRolelessCluster(t)
	db := c.newDatabase(t, "")
	c.exec(t, `CREATE ROLE insideout_app LOGIN NOSUPERUSER PASSWORD 'legacy'`)
	r := newRlRun(t)

	code, out := r.run(t, c.dsn(db, "", ""), "--create-roles")
	if code != 1 {
		t.Fatalf("setup exit = %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "provision_roles.sql") {
		t.Errorf("output should point at server/db/provision_roles.sql:\n%s", out)
	}
	if c.roleExists(t, "insideout_owner") || c.schemaExists(t, db) {
		t.Error("setup changed the database although one role already existed")
	}
}

// The probe against a real catalog for each kind of connecting role: which
// ones may attempt provisioning (canTry) is the only thing that varies. The
// catalog check itself is shared with TestProbeRolesReadsTheCatalog, which
// runs it on DATABASE_URL.
func TestRolelessProbeMatchesCatalog(t *testing.T) {
	c := newRolelessCluster(t)
	ctx := context.Background()
	got, err := probeRoles(ctx, c.admin, ownerRole, appRole)
	if want := (rolesProbe{canTry: true, database: "postgres", schemaMissing: true, dbOwner: "postgres"}); err != nil || got != want {
		t.Fatalf("superuser on a roleless cluster: probeRoles = %+v, %v; want %+v", got, err, want)
	}
	checkProbeRoles(t, c.admin)
	for _, tt := range []struct {
		attrs  string
		canTry bool
	}{
		{"", false},
		{"CREATEROLE", false},
		{"CREATEROLE BYPASSRLS", true},
	} {
		name, pw := c.newLoginRole(t, tt.attrs)
		pool, err := pgxpool.New(ctx, c.dsn("postgres", name, pw))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(pool.Close)
		got, err := probeRoles(ctx, pool, ownerRole, appRole)
		if err != nil || got.canTry != tt.canTry || got.owner || got.app {
			t.Errorf("role with %q: probeRoles = %+v, %v; want canTry=%v and no roles found", tt.attrs, got, err, tt.canTry)
		}
		checkProbeRoles(t, pool)
	}
}
