package store

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/db"
)

// Migrate bootstraps the insideout schema's migration tracking table (if
// missing) and applies any embedded .sql file not yet recorded, in
// filename order, each inside its own transaction.
func (s *Store) Migrate(ctx context.Context) ([]string, error) {
	// CREATE SCHEMA requires CREATE privilege on the *database*, not just
	// on the schema itself — even a redundant IF NOT EXISTS attempt fails
	// with "permission denied for database" for a role that's merely the
	// schema's owner but not the database's (the shared-instance target,
	// where an admin pre-provisions the schema; see
	// docs/plans/2026-07-20-go-rewrite/01-database.md §2). Only attempt
	// creation when the schema is actually missing — the dedicated-
	// instance target (insideout_app owns the whole database) hits that
	// path on its first-ever run.
	var schemaExists bool
	if err := s.Pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'insideout')`,
	).Scan(&schemaExists); err != nil {
		return nil, fmt.Errorf("migrate: check schema: %w", err)
	}
	if !schemaExists {
		if _, err := s.Pool.Exec(ctx, `CREATE SCHEMA insideout`); err != nil {
			return nil, fmt.Errorf("migrate: create schema: %w", err)
		}
	}
	if _, err := s.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS insideout.schema_migrations (
			filename text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)
	`); err != nil {
		return nil, fmt.Errorf("migrate: bootstrap: %w", err)
	}

	entries, err := db.Migrations.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("migrate: read embedded migrations: %w", err)
	}

	var filenames []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		filenames = append(filenames, e.Name())
	}
	sort.Strings(filenames)

	applied := map[string]bool{}
	rows, err := s.Pool.Query(ctx, "SELECT filename FROM insideout.schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("migrate: list applied: %w", err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, fmt.Errorf("migrate: scan applied: %w", err)
		}
		applied[name] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrate: rows: %w", err)
	}

	var justApplied []string
	for _, name := range filenames {
		if applied[name] {
			continue
		}
		contents, err := db.Migrations.ReadFile("migrations/" + name)
		if err != nil {
			return nil, fmt.Errorf("migrate: read %s: %w", name, err)
		}

		tx, err := s.Pool.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("migrate: begin %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, string(contents)); err != nil {
			tx.Rollback(ctx)
			return nil, fmt.Errorf("migrate: apply %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO insideout.schema_migrations (filename) VALUES ($1)", name); err != nil {
			tx.Rollback(ctx)
			return nil, fmt.Errorf("migrate: record %s: %w", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("migrate: commit %s: %w", name, err)
		}
		justApplied = append(justApplied, name)
	}

	return justApplied, nil
}
