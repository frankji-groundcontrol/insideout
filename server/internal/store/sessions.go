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
//
// Reuse safety: the revoke carries `AND revoked_at IS NULL` and we check the
// affected-row count. If two concurrent refreshes (or a replayed token) race
// on the same old session, exactly one revokes it; the loser touches zero
// rows and bails with ErrConflict WITHOUT minting a second live session.
// Without the guard both would revoke-then-insert, leaving two valid
// sessions from one token. RevokeSession's identical guard is the precedent.
//
// ponytail: on detected reuse we only refuse this rotation. Full reuse
// detection would also revoke the whole session family (force logout
// everywhere); that's a policy call and needs session lineage this schema
// doesn't track. Add it if replay becomes a real concern.
func (s *Store) RotateSession(ctx context.Context, oldID, userID uuid.UUID, newTokenHash string, expiresAt time.Time) (*Session, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `UPDATE insideout.sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, oldID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrConflict // already rotated/revoked — refuse to mint a second session
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
