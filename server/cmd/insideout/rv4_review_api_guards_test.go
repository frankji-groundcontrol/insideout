package main

import (
	"os"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
)

// remoteFixtureDSN is the unresolvable database testlock.AssertRefuses points a
// child at. A probe that keeps one DSN on this machine skips unless that child
// aimed the other one here, so a normal run never dials.
const remoteFixtureDSN = "postgres://app:secret@db.example.invalid:5432/app?sslmode=disable"

// rv4-FR-F65, rv4-FR-F195: the promote-admin tests (dbEnv, openTestStore), the
// saved-login refresh test, and the opens that still dial DATABASE_URL or
// DATABASE_OWNER_URL themselves. The parent runs each in a child aimed at a DSN
// that never resolves and needs the guard's refusal. The shutdown probes keep
// one DSN on this machine so deleting either check is the one that still dials.
func TestRvGuardsStrictModeRefusesARemoteDatabase(t *testing.T) {
	testlock.AssertRefuses(t,
		"TestPromoteAdminCreatesTheFirstAdmin",
		"TestSavedLoginOutlivesItsAccessTokenTwice",
		"TestRvGuardsProbeAppURL",
		"TestRvGuardsProbeOwnerURL",
		"TestIdeaConvertStoresTheLocaleOverTheRealServer",
		"TestProbeRolesReadsTheCatalog",
		"TestCLIAndMCPAgainstTheRealAPI",
		"TestRunPromoteAdmin_LogsTheFactNotTheAddress",
		"TestRvGuardsProbePromoteAdminLogAppURL",
		"TestRvGuardsProbePromoteAdminLogOwnerURL",
		"TestSeedDemo_RerunDoesNotDuplicateTheDemoWorkspace",
		"TestRunServe_ShutdownEndsAnOpenProgressStream",
		"TestRvGuardsProbeShutdownAppURL",
		"TestRvGuardsProbeShutdownOwnerURL")
}

func TestRvGuardsProbeAppURL(t *testing.T) {
	t.Setenv("DATABASE_OWNER_URL", "postgres://owner:secret@127.0.0.1:1/app?sslmode=disable")
	dbEnv(t)
}

func TestRvGuardsProbeOwnerURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://app:secret@127.0.0.1:1/app?sslmode=disable")
	dbEnv(t)
}

func TestRvGuardsProbeShutdownAppURL(t *testing.T) {
	if os.Getenv("DATABASE_URL") != remoteFixtureDSN {
		t.Skip("app-DSN probe runs in the strict-refusal child")
	}
	t.Setenv("DATABASE_OWNER_URL", "postgres://owner:secret@127.0.0.1:1/app?sslmode=disable")
	startShutdownRig(t, "")
}

func TestRvGuardsProbeShutdownOwnerURL(t *testing.T) {
	if os.Getenv("DATABASE_OWNER_URL") != remoteFixtureDSN {
		t.Skip("owner-DSN probe runs in the strict-refusal child")
	}
	t.Setenv("DATABASE_URL", "postgres://app:secret@127.0.0.1:1/app?sslmode=disable")
	startShutdownRig(t, "")
}

func TestRvGuardsProbePromoteAdminLogAppURL(t *testing.T) {
	if os.Getenv("DATABASE_URL") != remoteFixtureDSN {
		t.Skip("app-DSN probe runs in the strict-refusal child")
	}
	t.Setenv("DATABASE_OWNER_URL", "postgres://owner:secret@127.0.0.1:1/app?sslmode=disable")
	TestRunPromoteAdmin_LogsTheFactNotTheAddress(t)
}

func TestRvGuardsProbePromoteAdminLogOwnerURL(t *testing.T) {
	if os.Getenv("DATABASE_OWNER_URL") != remoteFixtureDSN {
		t.Skip("owner-DSN probe runs in the strict-refusal child")
	}
	t.Setenv("DATABASE_URL", "postgres://app:secret@127.0.0.1:1/app?sslmode=disable")
	TestRunPromoteAdmin_LogsTheFactNotTheAddress(t)
}
