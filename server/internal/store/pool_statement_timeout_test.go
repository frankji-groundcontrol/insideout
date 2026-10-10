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
