package store

import (
	"context"
	"os"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
)

// rv4-CLOSE-08: the test below ends by reading the production 'anthropic' row
// and failing if it is open. The agent package's wiring tests, in a sibling
// test process of the same `go test ./...`, open that very row for a moment,
// so this test spuriously failed whenever the two overlapped. It takes the
// "breaker" lock the wiring tests take.
func TestCircuitRowTestWaitsForTheCrossProcessLock(t *testing.T) {
	st := newTestStore(t)
	testlock.AssertWaits(t, st.Pool, "breaker", "TestCircuitFunctions_TakeARowID")
}

// TEST-06's core: the breaker rules live in SQL functions hardwired to the
// one shared 'anthropic' row, so no test can own its breaker state — the
// agent package's pins had to delta-assert against a row other packages'
// tests reset mid-flight. The functions must take a row id (defaulting to
// the production 'anthropic') so a test row is first-class.
func TestCircuitFunctions_TakeARowID(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	ctx := context.Background()
	st, err := Open(ctx, url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)

	const own = "circuit-own-row-test"
	if _, err := st.Pool.Exec(ctx,
		`INSERT INTO insideout.ai_circuit_breaker (id, state, failure_count, opened_at, last_failure_at)
		 VALUES ($1, 'open', 5, now(), now())
		 ON CONFLICT (id) DO UPDATE SET state = 'open', failure_count = 5, success_count = 0, opened_at = now(), last_failure_at = now()`,
		own); err != nil {
		t.Fatalf("seed an open test row: %v", err)
	}
	t.Cleanup(func() {
		_, _ = st.Pool.Exec(context.Background(), `DELETE FROM insideout.ai_circuit_breaker WHERE id = $1`, own)
	})

	var allowed bool
	var state string
	if err := st.Pool.QueryRow(ctx,
		`SELECT allowed, state FROM insideout.ai_check_circuit($1)`, own).Scan(&allowed, &state); err != nil {
		t.Fatalf("ai_check_circuit must accept a row id: %v", err)
	}
	if allowed || state != "open" {
		t.Fatalf("own open row: allowed=%v state=%q, want the open row's own verdict", allowed, state)
	}

	// A half-open probe that fails reopens immediately — on the OWN row,
	// without touching the production row.
	if _, err := st.Pool.Exec(ctx, `SELECT insideout.ai_record_circuit_result(false, $1)`, own); err != nil {
		t.Fatalf("ai_record_circuit_result must accept a row id: %v", err)
	}
	if err := st.Pool.QueryRow(ctx,
		`SELECT state FROM insideout.ai_circuit_breaker WHERE id = $1`, own).Scan(&state); err != nil {
		t.Fatalf("read own row: %v", err)
	}
	if state != "open" {
		t.Fatalf("own row state = %q after a failed probe, want open", state)
	}

	var prodState string
	if err := st.Pool.QueryRow(ctx,
		`SELECT state FROM insideout.ai_circuit_breaker WHERE id = 'anthropic'`).Scan(&prodState); err != nil {
		t.Fatalf("read production row: %v", err)
	}
	if prodState == "open" {
		t.Fatal("the test row's failure must not have opened the production breaker")
	}
}
