package store

import (
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
)

// rv4-FR-F195, rv4-FR-F201: INSIDEOUT_TEST_STRICT refused a database on another
// machine only in the tests that called testlock; newTestStore, the door the
// store tests share, and writeWalkStore, the RLS write walks' door, never did.
// Each probe is one helper and nothing else; the parent runs it in a child
// process aimed at a DSN that never resolves and needs the guard's refusal.
func TestRvGuardsStrictModeRefusesARemoteDatabase(t *testing.T) {
	testlock.AssertRefuses(t, "TestRvGuardsProbeNewTestStore", "TestRvGuardsProbeWriteWalkStore")
}

func TestRvGuardsProbeNewTestStore(t *testing.T)   { newTestStore(t) }
func TestRvGuardsProbeWriteWalkStore(t *testing.T) { writeWalkStore(t) }
