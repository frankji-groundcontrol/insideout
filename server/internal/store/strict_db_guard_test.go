package store

import (
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
)

// rv4-FR-F195, rv4-FR-F201: INSIDEOUT_TEST_STRICT refused a database on another
// machine only in the tests that called testlock. newTestStore and writeWalkStore
// ask first; the two tests below still open a DSN themselves. Each child is aimed
// at a DSN that never resolves and must fail with the guard's refusal.
func TestRvGuardsStrictModeRefusesARemoteDatabase(t *testing.T) {
	testlock.AssertRefuses(t,
		"TestRvGuardsProbeNewTestStore",
		"TestRvGuardsProbeWriteWalkStore",
		"TestServeRoleRefusesABypassRLSPool",
		"TestCircuitFunctions_TakeARowID")
}

func TestRvGuardsProbeNewTestStore(t *testing.T)   { newTestStore(t) }
func TestRvGuardsProbeWriteWalkStore(t *testing.T) { writeWalkStore(t) }
