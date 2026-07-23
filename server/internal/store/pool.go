// Package store holds the PostgreSQL connection pool and all pgx-based
// domain queries. No ORM, no code generator — plain SQL, hand-scanned.
package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// qualifyColumns prefixes every comma-separated column in cols with
// alias. Several *Columns constants (projectColumns, ideaColumns,
// prdColumns) are safe to SELECT bare almost everywhere, but a few
// queries JOIN another table and need every column qualified to avoid
// "column reference is ambiguous" — `"SELECT p." + cols` only prefixes
// the first column, silently leaving the rest bare (found via a real
// integration-test failure once workspace_memberships was joined
// alongside a table sharing a column name).
func qualifyColumns(alias, cols string) string {
	parts := strings.Split(cols, ", ")
	for i, p := range parts {
		parts[i] = alias + "." + p
	}
	return strings.Join(parts, ", ")
}

type Store struct {
	Pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("store: parse database url: %w", err)
	}
	// PgBouncer/Supavisor in transaction-pooling mode (the ?pgbouncer=true
	// connection string Supabase issues on port 6543) multiplexes many
	// clients over few backend connections, so server-side prepared
	// statements from pgx's default extended-protocol caching don't
	// survive across statements. Fall back to the simple protocol
	// whenever that mode is in play; a dedicated/session-mode Postgres
	// (bundled docker-compose postgres, or Supabase's session pooler on
	// port 5432) is unaffected either way.
	if strings.Contains(databaseURL, "pgbouncer=true") {
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("store: open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	return &Store{Pool: pool}, nil
}

func (s *Store) Close() {
	s.Pool.Close()
}

// withUserContext runs fn inside a transaction with app.user_id set to
// actorID for its duration, so the RLS policies in
// db/migrations/20260720150000_row_level_security.sql (readable via
// insideout.current_user_id()) can enforce the same rule the caller has
// already checked in Go — a DB-level backstop against app-layer bugs.
// Every store function touching an RLS-protected table must go through
// this, using the authenticated caller's id as actorID — except the two
// pre-auth users.go paths (GetUserByEmail, CreateUser), which run before
// any identity exists and are documented there.
func (s *Store) withUserContext(ctx context.Context, actorID uuid.UUID, fn func(tx pgx.Tx) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT set_config('app.user_id', $1, true)`, actorID.String()); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
