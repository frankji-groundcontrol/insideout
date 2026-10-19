package agent

import (
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
)

// rv4-FR-F195, rv4-FR-F201: the agent's newTestStore (gates_test.go) opened
// DATABASE_URL without asking testlock, so INSIDEOUT_TEST_STRICT did not stop a
// gate test from writing to a database on another machine. The probe is the
// helper and nothing else; the parent runs it in a child process aimed at a DSN
// that never resolves and needs the guard's refusal.
func TestRvGuardsStrictModeRefusesARemoteDatabase(t *testing.T) {
	testlock.AssertRefuses(t, "TestRvGuardsProbeNewTestStore")
}

func TestRvGuardsProbeNewTestStore(t *testing.T) { newTestStore(t) }
