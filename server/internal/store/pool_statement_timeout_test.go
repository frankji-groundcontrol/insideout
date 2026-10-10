package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// REL-02: every statement the request pool sends must be bounded server-side
// by a statement timeout, so a hung or lock-waiting query cannot pin its
// handler goroutine and a pool connection past the shutdown drain window.
// pg_sleep deliberately outruns any sane bound — a pool without one just
// lets the sleep finish.
func TestOpen_StatementTimeoutCutsLongQueries(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	start := time.Now()
	_, err := st.Pool.Exec(ctx, `SELECT pg_sleep(20)`)
	if err == nil {
		t.Fatalf("pg_sleep(20) ran to completion in %s: the app pool sends queries with no server-side statement bound", time.Since(start).Round(time.Second))
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "57014" {
		t.Fatalf("pg_sleep error = %v, want query_canceled (SQLSTATE 57014) from a server-side statement timeout", err)
	}
}

// The default is armed on the app pool config (fast, no dial) and an
// operator's own statement_timeout in the DSN still wins — the same
// retune-without-redeploy affordance as pool_max_conns.
func TestBoundStatements_ArmsTheDefault(t *testing.T) {
	cfg, err := poolConfig(poolTestDSN, 10)
	if err != nil {
		t.Fatalf("poolConfig: %v", err)
	}
	boundStatements(cfg)
	if got := cfg.ConnConfig.RuntimeParams["statement_timeout"]; got != "15000" {
		t.Errorf("statement_timeout runtime param = %q, want 15000 (15s)", got)
	}
}

func TestBoundStatements_ADSNTimeoutWins(t *testing.T) {
	cfg, err := poolConfig(poolTestDSN+"?statement_timeout=2s", 10)
	if err != nil {
		t.Fatalf("poolConfig: %v", err)
	}
	boundStatements(cfg)
	if got := cfg.ConnConfig.RuntimeParams["statement_timeout"]; got != "2s" {
		t.Errorf("statement_timeout runtime param = %q, want the DSN's 2s to win over the default", got)
	}
}
