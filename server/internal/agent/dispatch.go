package agent

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// dispatcher gates concurrent access to the LLM provider: at most one
// in-flight turn per conversation (convLock), and a global cap on
// concurrent provider calls (permits) — see
// docs/plans/2026-07-21-prd-agent-harness/plan.md §5.1.
//
// ponytail: both primitives are in-process/single-instance. Move
// convLock to pg_advisory_lock(conversation_id) and permits to a
// DB-backed counter when a second server instance exists.
type dispatcher struct {
	mu        sync.Mutex
	locks     map[uuid.UUID]*sync.Mutex
	permits   chan struct{}
	queueWait time.Duration
}

func newDispatcher(maxConcurrent int, queueWait time.Duration) *dispatcher {
	return &dispatcher{
		locks:     make(map[uuid.UUID]*sync.Mutex),
		permits:   make(chan struct{}, maxConcurrent),
		queueWait: queueWait,
	}
}

// tryLockConversation returns (unlock, true) if no other turn is in
// flight for this conversation, or (nil, false) if one already is — the
// caller should respond 409 CONVERSATION_BUSY immediately, no queuing.
func (d *dispatcher) tryLockConversation(conversationID uuid.UUID) (unlock func(), ok bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	l, exists := d.locks[conversationID]
	if !exists {
		l = &sync.Mutex{}
		d.locks[conversationID] = l
	}
	// Take the conversation lock while still holding d.mu. TryLock never
	// blocks, so this is safe, and it closes the TOCTOU that existed when
	// acquire/release ran outside d.mu: a releaser can no longer delete
	// the map entry between a new acquirer reading it and TryLocking it.
	// Every holder of a pointer to l now holds the lock, so the entry is
	// only reclaimed when no pointer escapes unreleased.
	if !l.TryLock() {
		return nil, false
	}
	return func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		l.Unlock()
		// Drop the entry only if the map still points at our instance and
		// nobody is waiting on it, so the map doesn't grow forever.
		if d.locks[conversationID] == l && l.TryLock() {
			l.Unlock()
			delete(d.locks, conversationID)
		}
	}, true
}

// acquirePermit reserves one of the global concurrent-provider-call
// slots, waiting up to queueWait. waiting reports whether the acquire had
// to queue at all (>0 other callers already held permits) — used as the
// degradation-ladder trigger for skipping the critic pass, since queue
// pressure right now is a much better signal than yesterday's average.
func (d *dispatcher) acquirePermit() (release func(), waited bool, ok bool) {
	select {
	case d.permits <- struct{}{}:
		return func() { <-d.permits }, false, true
	default:
	}
	timer := time.NewTimer(d.queueWait)
	defer timer.Stop()
	select {
	case d.permits <- struct{}{}:
		return func() { <-d.permits }, true, true
	case <-timer.C:
		return nil, true, false
	}
}
