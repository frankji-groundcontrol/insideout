package testlock

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The lock is the contract, so these run against real PostgreSQL like every
// store-backed test here; without DATABASE_URL they skip. Each test names its
// lock after a fresh uuid so it never queues behind (or in front of) a real
// signup_mode holder running in a sibling package.

// testPool opens a pool whose sessions carry a unique application name, so
// waiting() can tell this test's parked session from anyone else's.
func testPool(t *testing.T, params map[string]string) (*pgxpool.Pool, string) {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	app := "testlock-test-" + uuid.NewString()
	cfg.ConnConfig.RuntimeParams["application_name"] = app
	for k, v := range params {
		cfg.ConnConfig.RuntimeParams[k] = v
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool, app
}

func freshName() string { return "test-" + uuid.NewString() }

// held reports whether someone else holds the lock: it tries for at most d
// and, if it gets the lock, lets go of it again at once. Failing to get it for
// any reason but running out of time is a broken test, not a held lock.
func held(t *testing.T, pool *pgxpool.Pool, name string, d time.Duration) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	release, err := hold(ctx, pool, name)
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if err != nil {
		t.Fatalf("probe the lock: %v", err)
	}
	release()
	return false
}

func TestASecondHolderWaitsUntilTheFirstReleases(t *testing.T) {
	pool, app := testPool(t, nil)
	name := freshName()
	ctx := context.Background()

	first, err := hold(ctx, pool, name)
	if err != nil {
		t.Fatalf("first hold: %v", err)
	}
	type result struct {
		release func()
		err     error
	}
	second := make(chan result, 1)
	go func() {
		release, err := hold(ctx, pool, name)
		second <- result{release, err}
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		if parked, err := waiting(ctx, pool, name, app); err != nil {
			t.Fatalf("look for the waiter: %v", err)
		} else if parked {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the second holder never showed up waiting in pg_locks")
		}
		time.Sleep(20 * time.Millisecond)
	}
	select {
	case r := <-second:
		t.Fatalf("the second holder got the lock while the first still held it (err %v)", r.err)
	case <-time.After(200 * time.Millisecond):
	}

	first()
	select {
	case r := <-second:
		if r.err != nil {
			t.Fatalf("second hold after release: %v", r.err)
		}
		r.release()
	case <-time.After(5 * time.Second):
		t.Fatal("the second holder never got the lock after the first released it")
	}
}

func TestReleaseFreesTheLockAndMayRunTwice(t *testing.T) {
	pool, _ := testPool(t, nil)
	name := freshName()

	release, err := hold(context.Background(), pool, name)
	if err != nil {
		t.Fatalf("hold: %v", err)
	}
	if !held(t, pool, name, 300*time.Millisecond) {
		t.Fatal("a second session took the lock while it was held")
	}
	release()
	release() // a second release, as Acquire's cleanup does after an early one
	if held(t, pool, name, 5*time.Second) {
		t.Fatal("the lock was still held after release")
	}
}

func TestDifferentNamesDoNotExcludeEachOther(t *testing.T) {
	pool, _ := testPool(t, nil)
	a, b := freshName(), freshName()

	release, err := hold(context.Background(), pool, a)
	if err != nil {
		t.Fatalf("hold %s: %v", a, err)
	}
	t.Cleanup(release)
	if held(t, pool, b, 5*time.Second) {
		t.Fatal("holding one lock kept a differently named lock from being taken")
	}
}

// The app pool arms a server-side statement_timeout (store.Open uses 15s); a
// wait longer than that must not be cut short by it.
func TestWaitingOutlivesTheStatementTimeout(t *testing.T) {
	pool, _ := testPool(t, map[string]string{"statement_timeout": "200"})
	name := freshName()

	first, err := hold(context.Background(), pool, name)
	if err != nil {
		t.Fatalf("first hold: %v", err)
	}
	time.AfterFunc(700*time.Millisecond, first)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	second, err := hold(ctx, pool, name)
	if err != nil {
		t.Fatalf("a wait of 700ms under a 200ms statement_timeout failed: %v", err)
	}
	second()
}

func TestAcquireHoldsForTheTestAndReleasesAtItsEnd(t *testing.T) {
	pool, _ := testPool(t, nil)
	name := freshName()

	t.Run("holder", func(t *testing.T) {
		Acquire(t, pool, name)
		if !held(t, pool, name, 300*time.Millisecond) {
			t.Fatal("Acquire returned without holding the lock")
		}
	})
	if held(t, pool, name, 5*time.Second) {
		t.Fatal("the lock outlived the test that acquired it")
	}
}
