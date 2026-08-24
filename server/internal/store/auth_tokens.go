package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Auth-token kinds for emailed one-time links and codes.
const (
	AuthTokenMagicLink      = "magic_link"
	AuthTokenPasswordReset  = "password_reset"
	AuthTokenLoginCode      = "login_code"
	LoginCodeMaxAttempts    = 5
)

// CreateAuthToken stores an emailed one-time token's hash and supersedes the
// user's prior unused token of the same kind — one live link per flow, so a
// re-request invalidates the email already sitting in the inbox. Like
// CreateUser/GetUserByEmail this runs without withUserContext: the token is
// minted from a pre-auth request that carries no identity.
func (s *Store) CreateAuthToken(ctx context.Context, userID uuid.UUID, kind, tokenHash string, expiresAt time.Time) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		DELETE FROM insideout.auth_tokens
		WHERE user_id = $1 AND kind = $2 AND used_at IS NULL`,
		userID, kind,
	); err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO insideout.auth_tokens (user_id, kind, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)`,
		userID, kind, tokenHash, expiresAt,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ConsumeAuthToken marks the token used and returns its user, in one
// statement: the WHERE clause enforces single-use and expiry, so two racing
// consumers cannot both win (zero rows → ErrNotFound) — the same
// affected-rows discipline as RotateSession.
func (s *Store) ConsumeAuthToken(ctx context.Context, kind, tokenHash string) (uuid.UUID, error) {
	var userID uuid.UUID
	err := s.Pool.QueryRow(ctx, `
		UPDATE insideout.auth_tokens
		SET used_at = now()
		WHERE kind = $1 AND token_hash = $2 AND used_at IS NULL AND expires_at > now()
		RETURNING user_id`,
		kind, tokenHash,
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

// VerifyLoginCode checks a 6-digit sign-in code for a user. The exact hash
// consumes the code atomically (single-use + expiry, like ConsumeAuthToken).
// A miss charges the user's still-live code: LoginCodeMaxAttempts misses
// retire it, so a low-entropy code cannot be brute-forced within its TTL.
// Miss, exhausted, expired, and never-requested are all ErrNotFound — the
// caller cannot tell them apart, and neither can a probing client.
func (s *Store) VerifyLoginCode(ctx context.Context, userID uuid.UUID, codeHash string) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE insideout.auth_tokens
		SET used_at = now()
		WHERE user_id = $1 AND kind = $2 AND token_hash = $3 AND used_at IS NULL AND expires_at > now()`,
		userID, AuthTokenLoginCode, codeHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 1 {
		return nil
	}

	_, err = s.Pool.Exec(ctx, `
		UPDATE insideout.auth_tokens
		SET attempts = attempts + 1,
		    used_at = CASE WHEN attempts + 1 >= $2 THEN now() ELSE used_at END
		WHERE user_id = $1 AND kind = $3 AND used_at IS NULL AND expires_at > now()`,
		userID, LoginCodeMaxAttempts, AuthTokenLoginCode)
	if err != nil {
		return err
	}
	return ErrNotFound
}

// RevokeUserSessions kills every live refresh session for a user — called
// after a password reset so stolen cookies die with the old password.
func (s *Store) RevokeUserSessions(ctx context.Context, userID uuid.UUID) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE insideout.sessions SET revoked_at = now()
		WHERE user_id = $1 AND revoked_at IS NULL`,
		userID,
	)
	return err
}
