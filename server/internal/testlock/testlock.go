// Package testlock holds the guard rails the DATABASE_URL-gated tests share
// when they all run against ONE real database.
//
// `go test ./...` starts every package's test binary as its own process, in
// parallel, so nothing inside a process (t.Parallel, a mutex) can order two
// tests that live in different packages. A test that reads or flips a global
// row races its sibling packages: store's TestSignupModeSetting sets
// signup_mode to "open" while api's TestInvitationRoundTrip needs it to be
// "invitation_only" (review-4 TEST-02). The database is the one thing every
// process shares, so it does the ordering: a session-level advisory lock.
//
// The lock lives on a connection taken out of the pool (the way the api
// server's single-replica guard does it), so closing the pool never waits on
// a holder and a test binary that dies mid-test frees the lock when its
// session ends.
package testlock

import (
	"bytes"
	"context"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SignupModeLock names the lock every test that reads or writes the global
// signup_mode setting (the insideout.settings row) takes.
const SignupModeLock = "signup_mode"

// BreakerLock names the lock every test that opens, fills or asserts on the
// production circuit-breaker row ('anthropic') takes (rv4-CLOSE-08).
const BreakerLock = "breaker"

// waitLimit bounds how long a test waits for a lock another process holds. A
// holder keeps it for one test (seconds), so reaching this means a holder is
// wedged; failing beats hanging until the package timeout.
const waitLimit = 3 * time.Minute

// The two-int advisory key form keeps these locks apart from every bigint key
// the server itself takes (insideout:singleton, insideout:migrate).
const (
	lockSQL    = `SELECT pg_advisory_lock(hashtext('insideout:testlock'), hashtext($1))`
	unlockSQL  = `SELECT pg_advisory_unlock(hashtext('insideout:testlock'), hashtext($1))`
	waitingSQL = `SELECT count(*) FROM pg_locks l JOIN pg_stat_activity a ON a.pid = l.pid
		WHERE l.locktype = 'advisory' AND NOT l.granted AND l.objsubid = 2
		  AND l.classid = hashtext('insideout:testlock')::oid AND l.objid = hashtext($1)::oid
		  AND a.application_name = $2`
)

// strictEnv, when set, makes RequireLocalHost refuse a database on another
// machine. It is opt-in: CI and the shared-instance workflows leave it unset.
const strictEnv = "INSIDEOUT_TEST_STRICT"

// strict reports whether INSIDEOUT_TEST_STRICT asks for the local-only guard.
// Empty, 0, false, no and off mean off; anything else means on.
func strict() bool {
	switch strings.ToLower(os.Getenv(strictEnv)) {
	case "", "0", "false", "no", "off":
		return false
	}
	return true
}

// RequireLocalHost fails the test when INSIDEOUT_TEST_STRICT is set and the
// pool is aimed at a database that is not on this machine (review-4 TEST-06:
// the integration tests write rows, flip global settings and assume a seeded
// database, and DATABASE_URL may hold a shared instance). Acquire calls it;
// tests that open a pool of their own call it too. It checks the pool's
// configuration, so it never connects, and it never prints the host.
func RequireLocalHost(t testing.TB, pool *pgxpool.Pool) {
	t.Helper()
	cfg := pool.Config().ConnConfig
	hosts := []string{cfg.Host}
	for _, fb := range cfg.Fallbacks {
		hosts = append(hosts, fb.Host)
	}
	if err := localOnly(strict(), hosts...); err != nil {
		t.Fatalf("%v", err)
	}
}

// localOnly is RequireLocalHost's decision apart from the pool and the
// environment: with strict off everything passes, with it on every host must be
// this machine. The error leaves the host out so a log never carries it.
func localOnly(strict bool, hosts ...string) error {
	if !strict {
		return nil
	}
	for _, h := range hosts {
		if !isLocal(h) {
			return fmt.Errorf("%s is set: refusing to run against a database that is not on this machine (allowed: localhost, a loopback address, a unix socket); the host is not printed", strictEnv)
		}
	}
	return nil
}

// isLocal is true for the hosts a database on this machine answers on. pgx
// keeps a unix socket's directory in Host, and an empty Host means the default
// socket.
func isLocal(host string) bool {
	if host == "" || strings.HasPrefix(host, "/") || strings.EqualFold(host, "localhost") {
		return true
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.IsLoopback()
}

// SignupMode takes the signup_mode lock for the rest of the test. Call it
// before the test reads or saves the setting, so the restore a test registers
// with t.Cleanup (which runs first, last-registered-first) happens while the
// lock is still held.
func SignupMode(t testing.TB, pool *pgxpool.Pool) {
	t.Helper()
	Acquire(t, pool, SignupModeLock)
}

// Acquire blocks until this process holds the named lock, then keeps it until
// the returned release runs or the test ends, whichever is first. Release is
// safe to call more than once.
func Acquire(t testing.TB, pool *pgxpool.Pool, name string) (release func()) {
	t.Helper()
	RequireLocalHost(t, pool)
	ctx, cancel := context.WithTimeout(context.Background(), waitLimit)
	defer cancel()
	release, err := hold(ctx, pool, name)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Cleanup(release)
	return release
}

// hold takes the lock on a connection of its own. It is the part of Acquire
// that can run off the test goroutine, so it returns an error instead of
// failing a test.
func hold(ctx context.Context, pool *pgxpool.Pool, name string) (func(), error) {
	pc, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("testlock %q: take a connection: %w", name, err)
	}
	conn := pc.Hijack()
	// The app pool arms a server-side statement_timeout (store.Open); waiting
	// for a lock another process holds must be allowed to outlast it.
	if _, err := conn.Exec(ctx, `SET statement_timeout = 0`); err != nil {
		conn.Close(context.Background())
		return nil, fmt.Errorf("testlock %q: lift the statement timeout: %w", name, err)
	}
	if _, err := conn.Exec(ctx, lockSQL, name); err != nil {
		conn.Close(context.Background())
		return nil, fmt.Errorf("testlock %q: not acquired (another test run may be wedged holding it; Acquire waits up to %s): %w", name, waitLimit, err)
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			// Unlock explicitly before closing so the next waiter does not
			// depend on how fast the server tears the session down.
			conn.Exec(ctx, unlockSQL, name)
			conn.Close(ctx)
		})
	}, nil
}

// waiting reports whether a session of the application app is parked on the
// named lock. Filtering on the application name keeps another process that
// happens to wait for the same lock from vouching for the one under test.
func waiting(ctx context.Context, pool *pgxpool.Pool, name, app string) (bool, error) {
	var n int
	if err := pool.QueryRow(ctx, waitingSQL, name, app).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

// AssertWaits proves that each named test of the calling test binary takes
// the named lock. For every test it holds the lock itself, starts the binary
// again to run only that test, and requires the child to park on the lock
// (seen in pg_locks, by a per-child application name) rather than run to the
// end; once the lock is released the child must pass. A test that never
// calls testlock fails here at once: the child finishes without waiting.
//
// pool must reach the same database the child opens (DATABASE_URL).
func AssertWaits(t *testing.T, pool *pgxpool.Pool, name string, tests ...string) {
	t.Helper()
	for _, test := range tests {
		t.Run(test, func(t *testing.T) { assertWaits(t, pool, name, test) })
	}
}

func assertWaits(t *testing.T, pool *pgxpool.Pool, name, test string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("locate the test binary: %v", err)
	}
	release := Acquire(t, pool, name)

	app := "testlock-" + uuid.NewString()
	var out bytes.Buffer
	cmd := exec.Command(exe, "-test.run=^"+test+"$", "-test.count=1", "-test.v", "-test.timeout=2m")
	cmd.Env = append(os.Environ(), "PGAPPNAME="+app) // pgx reads it for every connection the child opens
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Start(); err != nil {
		t.Fatalf("start the test binary for %s: %v", test, err)
	}
	t.Cleanup(func() { cmd.Process.Kill() }) // no-op once it has exited
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	// gone ends the child and waits for it, after which its output is safe to read.
	gone := func() { cmd.Process.Kill(); <-done }
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for parked := false; !parked; {
		select {
		case err := <-done:
			t.Fatalf("%s ran to the end (%v) while another process held the %q lock: it does not call testlock\n%s", test, err, name, tail(&out))
		case <-ctx.Done():
			gone()
			t.Fatalf("%s neither finished nor waited for the %q lock within 30s\n%s", test, name, tail(&out))
		case <-time.After(50 * time.Millisecond):
		}
		var err error
		if parked, err = waiting(ctx, pool, name, app); err != nil {
			gone()
			t.Fatalf("look for %s in pg_locks: %v\n%s", test, err, tail(&out))
		}
	}

	release()
	select {
	case err := <-done:
		if err != nil || !strings.Contains(out.String(), "--- PASS: "+test) {
			t.Fatalf("%s did not pass once the %q lock was free (%v)\n%s", test, name, err, tail(&out))
		}
	case <-time.After(2 * time.Minute):
		gone()
		t.Fatalf("%s still running 2m after the %q lock was released\n%s", test, name, tail(&out))
	}
}

// tail is the last of a child's output, enough to see why it failed.
func tail(b *bytes.Buffer) string {
	s := b.String()
	if len(s) > 2000 {
		s = "..." + s[len(s)-2000:]
	}
	return s
}
