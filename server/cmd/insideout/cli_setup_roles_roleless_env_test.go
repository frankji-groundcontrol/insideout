//go:build roleless

package main

// What a stopped own-URL setup leaves in .env. The DSN setup is given becomes
// the runtime DATABASE_URL only once the role gate has passed; a stop that
// left an admin DSN there would let `serve` run as a role that bypasses RLS.
// Harness: cli_setup_roles_roleless_cluster_test.go.

import "testing"

// The skeleton value newRlRun's .env starts with.
const rlSkeletonURL = "change_me_db"

// databaseURL reads DATABASE_URL back from the run's .env without ever
// printing it: a failure message must not carry a DSN.
func (r *rlRun) databaseURL() string { return readEnvValues(r.env)["DATABASE_URL"] }

func TestRolelessStopLeavesDatabaseURLAlone(t *testing.T) {
	t.Run("piped without consent", func(t *testing.T) {
		c := newRolelessCluster(t)
		db := c.newDatabase(t, "")
		r := newRlRun(t)

		if code, out := r.run(t, c.dsn(db, "", "")); code != 1 {
			t.Fatalf("setup exit = %d, want 1\n%s", code, out)
		}
		if r.databaseURL() != rlSkeletonURL {
			t.Error("a stopped setup left the DSN it was given in .env as DATABASE_URL")
		}
	})

	t.Run("a role that cannot create the roles", func(t *testing.T) {
		c := newRolelessCluster(t)
		user, pw := c.newLoginRole(t, "")
		db := c.newDatabase(t, user)
		r := newRlRun(t)

		if code, out := r.run(t, c.dsn(db, user, pw), "--create-roles"); code != 1 {
			t.Fatalf("setup exit = %d, want 1\n%s", code, out)
		}
		if r.databaseURL() != rlSkeletonURL {
			t.Error("a refused provisioning left the DSN it was given in .env as DATABASE_URL")
		}
	})
}

// The other side of the same rule: once the gate passes, the supplied DSN is
// what .env carries (roles already there), or the app URL setup just wired.
func TestRolelessDatabaseURLIsWrittenOnceTheGatePasses(t *testing.T) {
	t.Run("roles already present", func(t *testing.T) {
		c := newRolelessCluster(t)
		db := c.newDatabase(t, "")
		c.operatorProvision(t, db, "ownerpw0001", "apppw0001")
		r := newRlRun(t)
		dsn := c.dsn(db, "", "")

		if code, out := r.run(t, dsn); code != 0 {
			t.Fatalf("setup exit = %d, want 0\n%s", code, out)
		}
		if r.databaseURL() != dsn {
			t.Error("the DSN setup was given is not the DATABASE_URL in .env")
		}
	})

	t.Run("roles provisioned", func(t *testing.T) {
		c := newRolelessCluster(t)
		db := c.newDatabase(t, "")
		r := newRlRun(t)
		superDSN := c.dsn(db, "", "")

		if code, out := r.run(t, superDSN, "--create-roles"); code != 0 {
			t.Fatalf("setup exit = %d, want 0\n%s", code, out)
		}
		got := r.databaseURL()
		if got == superDSN || got == rlSkeletonURL || got == "" {
			t.Error("DATABASE_URL is not the app role URL setup generated")
		}
	})
}
