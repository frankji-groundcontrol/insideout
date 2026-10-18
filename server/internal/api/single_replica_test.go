package api

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// SEC-03 / REL-05 (review-3): the auth throttle, email resend cooldown and
// SSE registries are per-process maps, and nothing at runtime ever said so —
// a second replica (or a well-meaning scale-up) silently multiplied every
// one of those budgets with no log line announcing it. Building the handler
// (what startup does once per process in main.go runServe) must state the
// single-replica contract, and a second server process against the same
// database must announce itself loudly. Real Postgres like every other
// store-backed test in this package; skips without DATABASE_URL.

// newSingletonLogServer builds a Server whose logs land in logs, the way the
// cooldown end-to-end test captures the dev-mode mail sender.
func newSingletonLogServer(st *store.Store, logs io.Writer) *Server {
	return NewServer(st, nil, &config.Config{},
		slog.New(slog.NewTextHandler(logs, nil)), nil, nil)
}

func TestSingletonGuardStatesContractAtStartup(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	st, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)

	var logs bytes.Buffer
	s := newSingletonLogServer(st, &logs)
	t.Cleanup(s.stopSingletonGuard)
	_ = s.Handler() // startup builds the handler chain exactly once

	// Only presence is asserted here: other store-backed tests in this
	// package build handlers without releasing the guard, so this database
	// may legitimately show a holder when this test runs.
	if !strings.Contains(logs.String(), "single-replica contract") {
		t.Fatalf("startup must state the single-replica contract (per-process throttle/cooldown/turn-lock/SSE state); got logs:\n%s", logs.String())
	}
}

func TestSingletonGuardErrorsWhenASecondServerRuns(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	st, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)

	// The guard waits out a rolling deploy's overlap (rv4-REL-04) before it
	// calls a held lock a second replica; shorten that wait for the test.
	every, window := singletonRetryEvery, singletonRetryWindow
	singletonRetryEvery, singletonRetryWindow = 20*time.Millisecond, 200*time.Millisecond
	t.Cleanup(func() { singletonRetryEvery, singletonRetryWindow = every, window })

	// The incumbent replica: its guard takes the singleton advisory lock on
	// a connection of its own and holds it, exactly like a running process.
	var incumbentLog syncBuffer
	incumbent := newSingletonLogServer(st, &incumbentLog)
	t.Cleanup(incumbent.stopSingletonGuard)
	_ = incumbent.Handler()

	// The second replica against the same database must say so, loudly, once
	// the holder has outlasted the retry window.
	var secondLog syncBuffer
	second := newSingletonLogServer(st, &secondLog)
	t.Cleanup(second.stopSingletonGuard)
	_ = second.Handler()

	// Something holds the lock for the whole test (the incumbent, or a guard
	// an earlier test left running), so the ERROR is certain and deterministic.
	deadline := time.Now().Add(5 * time.Second)
	for {
		out := secondLog.String()
		if strings.Contains(out, "second replica") && strings.Contains(out, "level=ERROR") {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("a second server process must announce itself with an ERROR log once the holder outlasts the retry window (the review's 'loud instead of silent'); got logs:\n%s", out)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// syncBuffer is a log sink the guard's background retry may write while the
// test reads it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// releaseSingletonGuards frees the lock from every Server this process built:
// other tests leave their guards holding it for the life of the process.
func releaseSingletonGuards() {
	singletonGuards.Range(func(k, _ any) bool {
		k.(*Server).stopSingletonGuard()
		return true
	})
}

// singletonLockHeld asks the database whether some other session holds the
// singleton lock. A free lock is taken and given straight back.
func singletonLockHeld(t *testing.T, st *store.Store) bool {
	t.Helper()
	ctx := context.Background()
	conn, err := st.Pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire a connection: %v", err)
	}
	defer conn.Release()
	var free bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtext($1))`, singletonLockKey).Scan(&free); err != nil {
		t.Fatalf("try the singleton lock: %v", err)
	}
	if free {
		if _, err := conn.Exec(ctx, `SELECT pg_advisory_unlock(hashtext($1))`, singletonLockKey); err != nil {
			t.Fatalf("give the singleton lock back: %v", err)
		}
	}
	return !free
}

// REL-04: the guard tried the lock once, at construction, and never again. A
// rolling deploy starts the new process while the old one still holds the lock,
// so the new one logged the "second replica" ERROR on every deploy, then
// served for the rest of its life without the lock: the next stray replica
// saw no holder and the guard was silent. Finding the lock held must be a
// pause, not a verdict: no ERROR in the first moments, and the lock taken over
// once the holder exits. Real Postgres; skips without DATABASE_URL.
func TestSingletonGuardTakesOverWhenTheIncumbentExits(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	st, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	releaseSingletonGuards() // so the incumbent below is the only holder

	var incumbentLog, secondLog syncBuffer
	incumbent := newSingletonLogServer(st, &incumbentLog)
	t.Cleanup(incumbent.stopSingletonGuard)
	_ = incumbent.Handler()
	if !singletonLockHeld(t, st) {
		t.Fatalf("the incumbent did not take the singleton lock; logs:\n%s", incumbentLog.String())
	}

	second := newSingletonLogServer(st, &secondLog)
	t.Cleanup(second.stopSingletonGuard)
	_ = second.Handler() // the rolling deploy: the new process starts beside the old one
	if out := secondLog.String(); strings.Contains(out, "level=ERROR") {
		t.Errorf("the second process logged an ERROR in its first moments, but an overlap with the process it replaces is normal; got logs:\n%s", out)
	}

	incumbent.stopSingletonGuard() // the old process exits
	deadline := time.Now().Add(8 * time.Second)
	for !singletonLockHeld(t, st) {
		if time.Now().After(deadline) {
			t.Fatalf("8s after the incumbent exited nobody holds the singleton lock: the second process never retried, so a third replica would find no holder; logs:\n%s", secondLog.String())
		}
		time.Sleep(250 * time.Millisecond)
	}
}
