package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

func (s *Store) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*Session, error) {
	var sess Session
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO insideout.sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at`,
		userID, tokenHash, expiresAt,
	).Scan(&sess.ID, &sess.UserID, &sess.TokenHash, &sess.ExpiresAt, &sess.RevokedAt, &sess.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

// GetActiveSessionByHash returns the session only if it is unexpired and
// unrevoked — the row disappearing from this query's result is exactly
// what makes refresh-token reuse (after rotation) fail closed.
func (s *Store) GetActiveSessionByHash(ctx context.Context, tokenHash string) (*Session, error) {
	var sess Session
	err := s.Pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM insideout.sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()`,
		tokenHash,
	).Scan(&sess.ID, &sess.UserID, &sess.TokenHash, &sess.ExpiresAt, &sess.RevokedAt, &sess.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) RevokeSession(ctx context.Context, id uuid.UUID) error {
	_, err := s.Pool.Exec(ctx, `UPDATE insideout.sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, id)
	return err
}

// RotateSession revokes the old session and creates a new one atomically,
// implementing the rotate-on-every-refresh policy from
// docs/plans/2026-07-20-go-rewrite/02-backend-go.md §2.
func (s *Store) RotateSession(ctx context.Context, oldID, userID uuid.UUID, newTokenHash string, expiresAt time.Time) (*Session, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE insideout.sessions SET revoked_at = now() WHERE id = $1`, oldID); err != nil {
		return nil, err
	}

	var sess Session
	err = tx.QueryRow(ctx, `
		INSERT INTO insideout.sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at`,
		userID, newTokenHash, expiresAt,
	).Scan(&sess.ID, &sess.UserID, &sess.TokenHash, &sess.ExpiresAt, &sess.RevokedAt, &sess.CreatedAt)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &sess, nil
}
