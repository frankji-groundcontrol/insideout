package testlock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
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

// TEST-06: INSIDEOUT_TEST_STRICT turns "run against whatever DATABASE_URL
// holds" into "only against a database on this machine", so a DSN that points
// at a shared instance cannot be written to by a test run by accident.

func TestStrictModeRefusesAHostThatIsNotThisMachine(t *testing.T) {
	for _, tc := range []struct {
		name   string
		strict bool
		hosts  []string
		refuse bool
	}{
		{"off, a remote host", false, []string{"db.example.org"}, false},
		{"off, no hosts", false, nil, false},
		{"localhost", true, []string{"localhost"}, false},
		{"localhost in capitals", true, []string{"LocalHost"}, false},
		{"loopback v4", true, []string{"127.0.0.1"}, false},
		{"anywhere in 127/8", true, []string{"127.8.9.10"}, false},
		{"loopback v6", true, []string{"::1"}, false},
		{"unix socket directory", true, []string{"/var/run/postgresql"}, false},
		{"no host (the default socket)", true, []string{""}, false},
		{"a remote name", true, []string{"db.example.org"}, true},
		{"a private address", true, []string{"10.0.0.5"}, true},
		{"a documentation address", true, []string{"203.0.113.9"}, true},
		{"the unspecified address", true, []string{"0.0.0.0"}, true},
		{"a name that only starts with localhost", true, []string{"localhost.example.org"}, true},
		{"a remote fallback behind a local host", true, []string{"localhost", "db.example.org"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := localOnly(tc.strict, tc.hosts...); (err != nil) != tc.refuse {
				t.Fatalf("localOnly(strict=%v, %q) = %v, want refused=%v", tc.strict, tc.hosts, err, tc.refuse)
			}
		})
	}
}

func TestTheRefusalNeverPrintsTheHost(t *testing.T) {
	err := localOnly(true, "db-prod-1.example.org")
	if err == nil {
		t.Fatal("a remote host was accepted in strict mode")
	}
	if strings.Contains(err.Error(), "example.org") || !strings.Contains(err.Error(), strictEnv) {
		t.Fatalf("refusal = %q, want it to name %s and keep the host out of the log", err, strictEnv)
	}
}

func TestStrictIsReadFromTheEnvironment(t *testing.T) {
	for value, want := range map[string]bool{
		"": false, "0": false, "false": false, "FALSE": false, "off": false, "no": false,
		"1": true, "true": true, "yes": true, "anything else": true,
	} {
		t.Setenv(strictEnv, value)
		if got := strict(); got != want {
			t.Errorf("%s=%q: strict() = %v, want %v", strictEnv, value, got, want)
		}
	}
}

// recordTB captures a Fatalf instead of failing the real test, so a helper
// that is supposed to fail a test can be tested.
type recordTB struct {
	testing.TB
	msg string
}

func (r *recordTB) Fatalf(format string, args ...any) {
	r.msg = fmt.Sprintf(format, args...)
	runtime.Goexit()
}

// Acquire is the door every lock-taking test goes through; in strict mode a
// pool aimed at another machine must be refused there, before any connection.
func TestAcquireRefusesARemotePoolInStrictMode(t *testing.T) {
	t.Setenv(strictEnv, "1")
	// .invalid never resolves, so even a broken guard cannot reach a real host.
	pool, err := pgxpool.New(context.Background(), "postgres://app:secret@db.example.invalid:5432/app?sslmode=disable")
	if err != nil {
		t.Fatalf("build the pool (lazy: it does not connect): %v", err)
	}
	t.Cleanup(pool.Close)

	rec := &recordTB{TB: t}
	done := make(chan struct{})
	go func() {
		defer close(done)
		Acquire(rec, pool, freshName())
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Acquire went on to connect to the remote host instead of refusing it")
	}
	if !strings.Contains(rec.msg, strictEnv) || strings.Contains(rec.msg, "example.invalid") {
		t.Fatalf("Acquire refused with %q, want a message naming %s and not the host", rec.msg, strictEnv)
	}
}
