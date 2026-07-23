package agent

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDispatcher_ConversationLock_RejectsConcurrent(t *testing.T) {
	d := newDispatcher(4, 50*time.Millisecond)
	convID := uuid.New()

	unlock, ok := d.tryLockConversation(convID)
	if !ok {
		t.Fatal("first lock should succeed")
	}
	if _, ok := d.tryLockConversation(convID); ok {
		t.Fatal("second concurrent lock should be rejected (409 CONVERSATION_BUSY)")
	}

	unlock()
	unlock2, ok := d.tryLockConversation(convID)
	if !ok {
		t.Fatal("lock should succeed again after unlock")
	}
	unlock2()

	// A different conversation is never blocked by another's lock.
	other := uuid.New()
	unlockOther, ok := d.tryLockConversation(other)
	if !ok {
		t.Fatal("a different conversation's lock should never be blocked")
	}
	unlockOther()
}

func TestDispatcher_ConversationLock_MapDoesNotLeakOnUnlock(t *testing.T) {
	d := newDispatcher(4, 50*time.Millisecond)
	convID := uuid.New()

	unlock, _ := d.tryLockConversation(convID)
	unlock()

	d.mu.Lock()
	_, exists := d.locks[convID]
	d.mu.Unlock()
	if exists {
		t.Fatal("uncontended unlock should delete the map entry, not leak it")
	}
}

func TestDispatcher_Permits_ImmediateWhenUnderCap(t *testing.T) {
	d := newDispatcher(2, 50*time.Millisecond)
	release, waited, ok := d.acquirePermit()
	if !ok || waited {
		t.Fatalf("first acquire under cap: ok=%v waited=%v, want ok=true waited=false", ok, waited)
	}
	release()
}

func TestDispatcher_Permits_QueuesThenTimesOut(t *testing.T) {
	d := newDispatcher(1, 30*time.Millisecond)
	release1, _, ok := d.acquirePermit()
	if !ok {
		t.Fatal("first acquire should succeed")
	}
	defer release1()

	_, waited, ok := d.acquirePermit()
	if ok {
		t.Fatal("second acquire against a full semaphore should time out, not succeed")
	}
	if !waited {
		t.Fatal("a failed acquire still queued, so waited should be true")
	}
}

func TestDispatcher_Permits_QueuedAcquireSucceedsOnRelease(t *testing.T) {
	d := newDispatcher(1, 200*time.Millisecond)
	release1, _, _ := d.acquirePermit()

	done := make(chan bool, 1)
	go func() {
		release2, waited, ok := d.acquirePermit()
		done <- ok && waited
		if ok {
			release2()
		}
	}()

	time.Sleep(20 * time.Millisecond)
	release1()

	select {
	case ok := <-done:
		if !ok {
			t.Fatal("queued acquire should succeed once a permit frees up, with waited=true")
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("queued acquire never returned")
	}
}
