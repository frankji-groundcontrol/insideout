package store

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// rv4-SEC-01: a burst of login-code guesses must be capped at
// LoginCodeMaxAttempts, not merely charged after the fact. The two-statement
// VerifyLoginCode compared the hash in one statement and charged the miss in
// a second, so every in-flight guess was evaluated for free before the first
// charge landed. These tests force that worst interleaving on a real
// Postgres: a one-connection pool serves statements strictly FIFO (puddle
// waits on a FIFO semaphore) and the connection is held while the whole
// burst parks, so "every compare first, every charge later" is the order the
// old code sees, deterministically rather than by scheduler luck.

// openSingleConnStore opens a second store on DATABASE_URL whose pool holds
// exactly one connection. Callers have already skipped via newTestStore when
// the variable is unset.
func openSingleConnStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	switch {
	case !strings.HasPrefix(url, "postgres://") && !strings.HasPrefix(url, "postgresql://"):
		url += " pool_max_conns=1" // key=value DSN
	case strings.Contains(url, "?"):
		url += "&pool_max_conns=1"
	default:
		url += "?pool_max_conns=1"
	}
	st, err := Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open single-connection store: %v", err)
	}
	t.Cleanup(st.Close)
	return st
}

// waitParkedGuesses blocks until n VerifyLoginCode calls are parked waiting
// for the held pool connection, so the next launch is provably queued behind
// them. It reads goroutine stacks instead of sleeping: a fixed delay would
// make the queue order, and with it the verdict, depend on scheduler timing.
func waitParkedGuesses(t *testing.T, n int) {
	t.Helper()
	buf := make([]byte, 1<<22)
	deadline := time.Now().Add(10 * time.Second)
	for {
		dump := string(buf[:runtime.Stack(buf, true)])
		parked := 0
		for _, g := range strings.Split(dump, "\n\n") {
			if strings.Contains(g, "(*Store).VerifyLoginCode") && strings.Contains(g, "semaphore.(*Weighted).Acquire") {
				parked++
			}
		}
		if parked >= n {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("only %d of %d guesses reached the pool queue", parked, n)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// burstGuesses queues one VerifyLoginCode per hash behind the held
// connection, in order, then releases it and returns every result in launch
// order. Each launch waits until the previous guesses are parked, so launch
// order is queue order.
func burstGuesses(t *testing.T, st *Store, userID uuid.UUID, hashes []string) []error {
	t.Helper()
	ctx := context.Background()
	hold, err := st.Pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("hold the only pool connection: %v", err)
	}
	defer hold.Release()

	errs := make([]error, len(hashes))
	var wg sync.WaitGroup
	for i, h := range hashes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = st.VerifyLoginCode(ctx, userID, h)
		}()
		waitParkedGuesses(t, i+1)
	}
	hold.Release()
	wg.Wait()
	return errs
}

func TestLoginCodeBurstCappedAtMaxAttempts(t *testing.T) {
	setup := newTestStore(t)
	ctx := context.Background()
	u := mkUser(t, setup)
	good := freshHash(t)
	mustCreateAuthToken(t, setup, u.ID, AuthTokenLoginCode, good, 10*time.Minute)

	// 20 wrong guesses, then the correct code last: guess number 21 against
	// a code that must have retired after LoginCodeMaxAttempts.
	const wrong = 20
	hashes := make([]string, 0, wrong+1)
	for i := 0; i < wrong; i++ {
		hashes = append(hashes, freshHash(t))
	}
	hashes = append(hashes, good)

	errs := burstGuesses(t, openSingleConnStore(t), u.ID, hashes)

	for i, err := range errs[:wrong] {
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("wrong guess %d: err = %v, want ErrNotFound", i+1, err)
		}
	}
	if err := errs[wrong]; !errors.Is(err, ErrNotFound) {
		t.Errorf("correct code as guess %d: err = %v; a burst got %d free evaluations, the cap is %d",
			wrong+1, err, wrong+1, LoginCodeMaxAttempts)
	}

	var attempts int
	var retired bool
	if err := setup.Pool.QueryRow(ctx, `
		SELECT attempts, used_at IS NOT NULL
		FROM insideout.auth_tokens WHERE user_id = $1 AND kind = $2`,
		u.ID, AuthTokenLoginCode).Scan(&attempts, &retired); err != nil {
		t.Fatalf("read token row: %v", err)
	}
	if attempts != LoginCodeMaxAttempts || !retired {
		t.Errorf("after the burst attempts = %d retired = %v, want attempts = %d retired = true",
			attempts, retired, LoginCodeMaxAttempts)
	}
}

// The cap counts guesses, not luck: the correct code queued as the
// LoginCodeMaxAttempts-th guess still verifies inside a burst, and every
// guess queued after it is rejected because that guess consumed the code.
func TestLoginCodeBurstLastAllowedGuessStillWins(t *testing.T) {
	setup := newTestStore(t)
	ctx := context.Background()
	u := mkUser(t, setup)
	good := freshHash(t)
	mustCreateAuthToken(t, setup, u.ID, AuthTokenLoginCode, good, 10*time.Minute)

	hashes := make([]string, 0, LoginCodeMaxAttempts+10)
	for i := 0; i < LoginCodeMaxAttempts-1; i++ {
		hashes = append(hashes, freshHash(t))
	}
	hashes = append(hashes, good)
	for i := 0; i < 10; i++ {
		hashes = append(hashes, freshHash(t))
	}

	errs := burstGuesses(t, openSingleConnStore(t), u.ID, hashes)

	for i, err := range errs {
		switch {
		case i == LoginCodeMaxAttempts-1 && err != nil:
			t.Errorf("correct code as guess %d (inside the cap): err = %v, want nil", i+1, err)
		case i != LoginCodeMaxAttempts-1 && !errors.Is(err, ErrNotFound):
			t.Errorf("guess %d: err = %v, want ErrNotFound", i+1, err)
		}
	}

	var attempts int
	var retired bool
	if err := setup.Pool.QueryRow(ctx, `
		SELECT attempts, used_at IS NOT NULL
		FROM insideout.auth_tokens WHERE user_id = $1 AND kind = $2`,
		u.ID, AuthTokenLoginCode).Scan(&attempts, &retired); err != nil {
		t.Fatalf("read token row: %v", err)
	}
	if attempts != LoginCodeMaxAttempts || !retired {
		t.Errorf("attempts = %d retired = %v, want attempts = %d retired = true",
			attempts, retired, LoginCodeMaxAttempts)
	}
}

// Single-use survives the one-statement rewrite under real contention:
// racing submissions of the correct code on the full-size pool yield exactly
// one success.
func TestLoginCodeConcurrentCorrectCodeSingleUse(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u := mkUser(t, st)
	good := freshHash(t)
	mustCreateAuthToken(t, st, u.ID, AuthTokenLoginCode, good, 10*time.Minute)

	errs := make([]error, 8)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs[i] = st.VerifyLoginCode(ctx, u.ID, good)
		}()
	}
	close(start)
	wg.Wait()

	wins := 0
	for i, err := range errs {
		switch {
		case err == nil:
			wins++
		case !errors.Is(err, ErrNotFound):
			t.Errorf("racer %d: err = %v, want nil or ErrNotFound", i, err)
		}
	}
	if wins != 1 {
		t.Errorf("%d of %d concurrent submissions of the correct code verified, want exactly 1", wins, len(errs))
	}
}
