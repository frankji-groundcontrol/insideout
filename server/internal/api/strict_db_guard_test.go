package api

import (
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
)

// rv4-FR-F195: newRouteWorld, newStreamFixture and newGateFixture opened
// DATABASE_URL without asking testlock, so INSIDEOUT_TEST_STRICT did not stop
// the api package from writing to a database on another machine. Each probe is
// one helper; testlock.AssertRefuses runs it in a child aimed at a DSN that
// never resolves and needs the guard's refusal. Deleting the call turns that
// child red.
func TestRvGuardsStrictModeRefusesARemoteDatabase(t *testing.T) {
	testlock.AssertRefuses(t,
		"TestRvGuardsProbeNewRouteWorld",
		"TestRvGuardsProbeNewStreamFixture",
		"TestRvGuardsProbeNewGateFixture")
}

func TestRvGuardsProbeNewRouteWorld(t *testing.T)    { newRouteWorld(t) }
func TestRvGuardsProbeNewStreamFixture(t *testing.T) { newStreamFixture(t) }
func TestRvGuardsProbeNewGateFixture(t *testing.T)   { newGateFixture(t) }
