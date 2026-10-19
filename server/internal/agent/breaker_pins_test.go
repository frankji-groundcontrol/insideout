package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
)

// Pins for the circuit-breaker WIRING that surrounds an LLM call: which
// errors feed the breaker and what Admit's done records. The SQL rules
// themselves (open verdict, failed half-open probe reopening, row-id
// isolation) are pinned on a dedicated test row in
// internal/store/circuit_row_id_test.go — the functions take a row id
// since 20261011100000 — so this file no longer needs to pin them against
// the shared row. The wiring still runs on the one production row
// ('anthropic', global by design), so this file is deliberately sequential
// (no t.Parallel) and the api package's parallel resets remain the reason
// these assert DELTAS, never absolutes: those resets only ever zero the
// counts — they never touch last_failure_at — so a failure recording is
// witnessed by that timestamp moving, and "records nothing" is witnessed by
// counts that stay put around code that must not record.

// breakerState reads the shared row's failure count and the epoch of its last
// recorded failure (0 when there was none).
func breakerState(t *testing.T, c *Coach) (failures int, lastFailure float64) {
	t.Helper()
	if err := c.store.Pool.QueryRow(context.Background(),
		`SELECT failure_count, coalesce(extract(epoch FROM last_failure_at), 0) FROM insideout.ai_circuit_breaker WHERE id = 'anthropic'`).
		Scan(&failures, &lastFailure); err != nil {
		t.Fatalf("read the circuit breaker row: %v", err)
	}
	return failures, lastFailure
}

// resetBreaker closes the shared row now and again at cleanup. The restore is
// a direct UPDATE, not RecordCircuitResult(true): a success cannot close an
// open circuit, and later tests must not inherit this test's failures.
func resetBreaker(t *testing.T, c *Coach) {
	t.Helper()
	reset := func() {
		if _, err := c.store.Pool.Exec(context.Background(),
			`UPDATE insideout.ai_circuit_breaker SET state = 'closed', failure_count = 0, success_count = 0, opened_at = NULL, last_failure_at = NULL WHERE id = 'anthropic'`); err != nil {
			t.Fatalf("reset the circuit breaker row: %v", err)
		}
	}
	reset()
	t.Cleanup(reset)
}

// rv4-CLOSE-08: the wiring tests below open and fill the one production row
// while store's TestCircuitFunctions_TakeARowID, in a sibling test process of
// the same `go test ./...`, asserts that row is not open. Every test that goes
// through resetBreaker must therefore hold the cross-process "breaker" lock.
// The test that reads the row in the other package takes the same lock.
func TestBreakerWiringTestsWaitForTheCrossProcessLock(t *testing.T) {
	st := newTestStore(t)
	testlock.AssertWaits(t, st.Pool, "breaker",
		"TestFailTurn_FeedsOnlyProviderHealthToTheBreaker",
		"TestAdmitDone_ReportsTheBreakerVerdicts",
		"TestAdmit_RefusedWhileTheCircuitIsOpen",
		"TestHandleMessage_TotalTurnDeadlineCutsATricklingTurn")
}

// failTurn feeds the breaker only failures that say something about the
// provider: our own loop failing (ErrLoopInternal) records nothing, an
// unclassified provider error records a failure.
func TestFailTurn_FeedsOnlyProviderHealthToTheBreaker(t *testing.T) {
	st := newTestStore(t)
	_, ex := seedConversation(t, st)
	run := startRun(t, st, ex)
	c := New(st, fakeStreamer{}, discardLog())
	sse, ok := newSSEWriter(httptest.NewRecorder())
	if !ok {
		t.Fatal("the test recorder must accept an SSE writer")
	}
	resetBreaker(t, c)

	failures0, last0 := breakerState(t, c)
	c.failTurn(context.Background(), ex.actorID, run.ID,
		fmt.Errorf("%w: exceeded max tool iterations", ErrLoopInternal), sse)
	failures1, last1 := breakerState(t, c)
	if failures1 != failures0 || last1 != last0 {
		t.Fatalf("an ErrLoopInternal turn moved the breaker: failures %d->%d, last_failure epoch %v->%v", failures0, failures1, last0, last1)
	}

	_, last0 = breakerState(t, c)
	c.failTurn(context.Background(), ex.actorID, run.ID,
		errors.New("agent: anthropic returned 503"), sse)
	_, last1 = breakerState(t, c)
	if last1 <= last0 {
		t.Fatalf("an unclassified provider failure did not feed the breaker: last_failure epoch %v -> %v", last0, last1)
	}
}

// Admit's done reports the breaker verdicts: a success resets the failure
// count, only rate-limited/transient/stalled provider errors record a
// failure, and a botched plan or a client that went away records nothing.
func TestAdmitDone_ReportsTheBreakerVerdicts(t *testing.T) {
	c := New(newTestStore(t), nil, discardLog())
	userID, workspaceID := admitFixture(t, c)

	admitDone := func(t *testing.T, callErr error) {
		t.Helper()
		_, done, ok := admit(c, userID, workspaceID)
		if !ok {
			t.Fatalf("admit was refused while reporting %v", callErr)
		}
		done(callErr)
	}

	t.Run("a success clears the failure count", func(t *testing.T) {
		resetBreaker(t, c)
		admitDone(t, fmt.Errorf("%w: anthropic returned 429", ErrProviderRateLimited))
		admitDone(t, nil)
		if failures, _ := breakerState(t, c); failures != 0 {
			t.Fatalf("failure_count = %d after a successful call, want the reset to 0", failures)
		}
	})

	for _, tc := range []struct {
		name string
		err  error
	}{
		{"rate limited records a failure", fmt.Errorf("%w: anthropic returned 429", ErrProviderRateLimited)},
		{"transient records a failure", fmt.Errorf("%w: anthropic returned 503", ErrProviderTransient)},
		{"an upstream stall records a failure", ErrUpstreamStall},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetBreaker(t, c)
			_, last0 := breakerState(t, c)
			admitDone(t, tc.err)
			_, last1 := breakerState(t, c)
			if last1 <= last0 {
				t.Fatalf("done(%v) did not record a breaker failure: last_failure epoch %v -> %v", tc.err, last0, last1)
			}
		})
	}

	for _, tc := range []struct {
		name string
		err  error
	}{
		{"a plan the model botched records nothing", errors.New("planner returned no nodes")},
		{"a client that went away records nothing", context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetBreaker(t, c)
			failures0, last0 := breakerState(t, c)
			admitDone(t, tc.err)
			failures1, last1 := breakerState(t, c)
			if failures1 != failures0 || last1 != last0 {
				t.Fatalf("done(%v) moved the breaker: failures %d->%d, last_failure epoch %v->%v", tc.err, failures0, failures1, last0, last1)
			}
		})
	}
}

// While the breaker is open, Admit refuses with the coach's own 503
// CIRCUIT_OPEN before any permit, run row or provider call is spent.
func TestAdmit_RefusedWhileTheCircuitIsOpen(t *testing.T) {
	c := New(newTestStore(t), nil, discardLog())
	userID, workspaceID := admitFixture(t, c)
	resetBreaker(t, c)
	if _, err := c.store.Pool.Exec(context.Background(),
		`UPDATE insideout.ai_circuit_breaker SET state = 'open', failure_count = 5, opened_at = now() WHERE id = 'anthropic'`); err != nil {
		t.Fatalf("open the circuit breaker: %v", err)
	}

	rec, done, ok := admit(c, userID, workspaceID)
	if ok || done != nil {
		t.Fatal("admit was allowed while the circuit was open")
	}
	if rec.Code != http.StatusServiceUnavailable ||
		!strings.Contains(rec.Body.String(), `"code":"CIRCUIT_OPEN"`) ||
		!strings.Contains(rec.Body.String(), `"circuit_state":"open"`) {
		t.Fatalf("open-circuit refusal = %d %s, want a 503 CIRCUIT_OPEN naming the open state", rec.Code, rec.Body.String())
	}

	var runs int
	if err := c.store.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM insideout.ai_runs WHERE user_id = $1`, userID).Scan(&runs); err != nil {
		t.Fatalf("count runs: %v", err)
	}
	if runs != 0 {
		t.Fatalf("%d run rows after an open-circuit refusal, want 0", runs)
	}
}
