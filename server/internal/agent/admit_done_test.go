package agent

import (
	"testing"
	"time"
)

// The roadmap handlers defer done (see internal/api/roadmap_ai.go), so a
// planner that panics unwinds into done being called late or twice: the
// permit must come back exactly once, never block the caller.
func TestAdmit_DoneIsIdempotent(t *testing.T) {
	t.Setenv("INSIDEOUT_AI_MAX_CONCURRENT", "1")
	t.Setenv("INSIDEOUT_AI_QUEUE_WAIT", "50ms")
	c := New(newTestStore(t), nil, discardLog())
	userID, workspaceID := admitFixture(t, c)

	_, done, ok := admit(c, userID, workspaceID)
	if !ok {
		t.Fatal("the first admit was refused")
	}
	done(nil)

	returned := make(chan struct{})
	go func() {
		done(nil)
		close(returned)
	}()
	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("a second done call blocked: releasing the permit is not idempotent")
	}

	// Exactly one release: the slot is usable again by the next caller.
	_, again, ok := admit(c, userID, workspaceID)
	if !ok {
		t.Fatal("admit after a double done was refused: the permit was not handed back exactly once")
	}
	again(nil)
}
