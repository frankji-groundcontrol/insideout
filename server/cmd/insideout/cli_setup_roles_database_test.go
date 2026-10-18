package main

import (
	"strings"
	"testing"
)

// The database-readiness row of the decision table (cli_setup_roles.go). With
// both roles present, setup stops only when the migration would certainly die
// on this database: the insideout schema is missing and insideout_owner cannot
// create it. Ownership alone is not the test — a database somebody else owns
// migrates fine once an admin pre-created the schema (the shared-instance
// layout of docs/architecture/database-and-rls.md) — and the zero value of
// the facts means "no problem known", so unknown never stops.
func TestDecideRolesDatabaseReadiness(t *testing.T) {
	present := rolesProbe{owner: true, app: true, ownerBypassRLS: true, dbOwner: "alice"}
	with := func(p rolesProbe, schemaMissing, ownerCantCreate bool) rolesProbe {
		p.schemaMissing, p.ownerCantCreate = schemaMissing, ownerCantCreate
		return p
	}
	for _, tt := range []struct {
		name             string
		p                rolesProbe
		terminal, create bool
		want             rolesAction
	}{
		{"nothing known against it proceeds", present, false, false, rolesProceed},
		{"schema missing but the owner can create it", with(present, true, false), false, false, rolesProceed},
		{"shared instance: schema pre-created, owner cannot create", with(present, false, true), false, false, rolesProceed},
		{"schema exists and the owner can create", with(present, false, false), false, false, rolesProceed},
		{"schema missing and the owner cannot create it", with(present, true, true), false, false, rolesNotReady},
		{"a terminal does not turn the stop into a prompt", with(present, true, true), true, false, rolesNotReady},
		{"--create-roles does not change a database owner", with(present, true, true), true, true, rolesNotReady},
		{"the stop needs no privilege to explain", with(rolesProbe{owner: true, app: true, ownerBypassRLS: true}, true, true), false, false, rolesNotReady},
		{"a lone role still stops the old way", with(rolesProbe{owner: true, canTry: true}, true, true), true, true, rolesStop},
		{"missing roles still go the provisioning way", with(rolesProbe{canTry: true}, true, true), false, true, rolesCreate},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := decideRoles(tt.p, tt.terminal, tt.create); got != tt.want {
				t.Fatalf("decideRoles(%+v, terminal=%v, create=%v) = %d, want %d", tt.p, tt.terminal, tt.create, got, tt.want)
			}
		})
	}
}

// What the user reads when the database is not ready: whose it is, both ways
// to fix it as SQL they can paste, and no secret of any kind.
func TestNotReadyMessage(t *testing.T) {
	out := captureStderr(t, func() {
		explainNotReady(rolesProbe{owner: true, app: true, schemaMissing: true, ownerCantCreate: true, database: "appdb", dbOwner: "alice"})
	})
	for _, want := range []string{
		`is owned by "alice"`,
		"superuser",
		"Nothing in the database was changed",
		`  ALTER DATABASE "appdb" OWNER TO insideout_owner;`,
		`  GRANT CONNECT ON DATABASE "appdb" TO insideout_app;`,
		`  REVOKE CREATE ON SCHEMA public FROM PUBLIC;`,
		`  REVOKE ALL ON SCHEMA public FROM insideout_app;`,
		`  CREATE SCHEMA insideout AUTHORIZATION insideout_owner;`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	for _, bad := range []string{"CREATE ROLE", "PASSWORD", "postgres://", "<owner-password>"} {
		if strings.Contains(out, bad) {
			t.Errorf("output must not mention %q (the roles exist; no credential is involved):\n%s", bad, out)
		}
	}

	out = captureStderr(t, func() { explainNotReady(rolesProbe{database: `we"ird db`, dbOwner: "alice"}) })
	if !strings.Contains(out, `ALTER DATABASE "we""ird db" OWNER TO insideout_owner;`) {
		t.Errorf("database name not quoted as an identifier:\n%s", out)
	}
}
