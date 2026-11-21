package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreateResetWatch records a password-reset watch. An empty tokenHash is a
// placeholder for an unknown address: it expires and never completes. The returned id is what the client polls.
func (s *Store) CreateResetWatch(ctx context.Context, tokenHash string, expiresAt time.Time) (uuid.UUID, error) {
	id := uuid.New()
	var hash any
	if tokenHash != "" {
		hash = tokenHash
	}
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO insideout.reset_watches (id, token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		id, hash, expiresAt,
	)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// CompleteResetWatch marks the watch that belongs to a consumed reset
// token. A reset that has no watch (a link issued before this table) is
// not an error.
func (s *Store) CompleteResetWatch(ctx context.Context, tokenHash string) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE insideout.reset_watches
		SET completed_at = now()
		WHERE token_hash = $1 AND completed_at IS NULL`,
		tokenHash,
	)
	return err
}

// ResetWatchStatus reports pending, completed, or expired. An unknown id
// is expired, the same answer as a watch that has run out.
func (s *Store) ResetWatchStatus(ctx context.Context, id uuid.UUID) (string, error) {
	var status string
	err := s.Pool.QueryRow(ctx, `
		SELECT CASE
			WHEN completed_at IS NOT NULL THEN 'completed'
			WHEN expires_at > now() THEN 'pending'
			ELSE 'expired'
		END
		FROM insideout.reset_watches
		WHERE id = $1`,
		id,
	).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "expired", nil
	}
	if err != nil {
		return "", err
	}
	return status, nil
}
