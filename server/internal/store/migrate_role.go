package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// prepareMigrateConn makes this transaction run as insideout_owner.
// A superuser is allowed only long enough to SET LOCAL ROLE; DEFINER
// functions must not be owned by a superuser.
func prepareMigrateConn(ctx context.Context, tx pgx.Tx) error {
	var user string
	var super bool
	if err := tx.QueryRow(ctx, `SELECT current_user, current_setting('is_superuser') = 'on'`).Scan(&user, &super); err != nil {
		return fmt.Errorf("migrate: current_user: %w", err)
	}
	if super {
		if _, err := tx.Exec(ctx, `SET LOCAL ROLE insideout_owner`); err != nil {
			return fmt.Errorf("migrate: SET LOCAL ROLE insideout_owner (create the NOSUPERUSER role first): %w", err)
		}
		return nil
	}
	if user != "insideout_owner" {
		return fmt.Errorf("migrate: connected as %q; use DATABASE_OWNER_URL (insideout_owner). SECURITY DEFINER objects must not be owned by a superuser or by insideout_app", user)
	}
	var ownerSuper bool
	if err := tx.QueryRow(ctx, `SELECT rolsuper FROM pg_roles WHERE rolname = 'insideout_owner'`).Scan(&ownerSuper); err != nil {
		return fmt.Errorf("migrate: inspect insideout_owner: %w", err)
	}
	if ownerSuper {
		return fmt.Errorf("migrate: insideout_owner is a superuser; ALTER ROLE insideout_owner NOSUPERUSER")
	}
	return nil
}

func (s *Store) beginMigrateTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := s.migrator().Begin(ctx)
	if err != nil {
		return nil, err
	}
	if err := prepareMigrateConn(ctx, tx); err != nil {
		tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}

func (s *Store) migrateExec(ctx context.Context, sql string, args ...any) error {
	tx, err := s.beginMigrateTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) migrateQueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return s.migrator().QueryRow(ctx, sql, args...)
}

func (s *Store) migratePoolOrMain() *pgxpool.Pool { return s.migrator() }
