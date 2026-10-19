package main

import (
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
)

// rv4-FR-F65: the promote-admin tests (dbEnv, openTestStore) and the saved-login
// refresh test opened DATABASE_URL without asking testlock, so
// INSIDEOUT_TEST_STRICT did not stop them from writing users, sessions and an
// admin account to a database on another machine. The parent runs each in a
// child process aimed at a DSN that never resolves and needs the guard's
// refusal. The real tests are children as they stand; the probe is dbEnv alone.
func TestRvGuardsStrictModeRefusesARemoteDatabase(t *testing.T) {
	testlock.AssertRefuses(t,
		"TestPromoteAdminCreatesTheFirstAdmin",
		"TestSavedLoginOutlivesItsAccessTokenTwice",
		"TestRvGuardsProbeOwnerURL")
}

// TestRvGuardsProbeOwnerURL keeps the app DSN on this machine, so only the
// owner DSN the parent hands the child is on another one: dbEnv guards it on
// its own, not just through DATABASE_URL.
func TestRvGuardsProbeOwnerURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://app:secret@127.0.0.1:1/app?sslmode=disable")
	dbEnv(t)
}
