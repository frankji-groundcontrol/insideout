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
	AuthTokenMagicLink     = "magic_link"
	AuthTokenPasswordReset = "password_reset"
	AuthTokenLoginCode     = "login_code"
	LoginCodeMaxAttempts   = 5
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

// VerifyLoginCode checks a 6-digit sign-in code for a user. Every guess is
// charged and compared in ONE statement: it takes an attempt on the user's
// still-live code and reports whether the hash matched, so the attempt slot
// is reserved before the answer exists. The exact hash consumes the code
// (single-use + expiry, like ConsumeAuthToken); the LoginCodeMaxAttempts-th
// guess, right or wrong, retires it, so a low-entropy code cannot be
// brute-forced within its TTL. Two statements (compare, then charge) would let
// a burst of concurrent guesses all be compared before the first charge
// landed; here Postgres re-checks the WHERE clause on the latest row version
// once a concurrent guess releases the row lock, so racing guesses serialise
// and the (cap+1)-th finds no live code. Miss, exhausted, expired, and
// never-requested are all ErrNotFound — the caller cannot tell them apart,
// and neither can a probing client.
func (s *Store) VerifyLoginCode(ctx context.Context, userID uuid.UUID, codeHash string) error {
	var matched bool
	err := s.Pool.QueryRow(ctx, `
		UPDATE insideout.auth_tokens
		SET attempts = attempts + 1,
		    used_at = CASE WHEN token_hash = $3 OR attempts + 1 >= $4 THEN now() ELSE used_at END
		WHERE user_id = $1 AND kind = $2 AND used_at IS NULL AND expires_at > now() AND attempts < $4
		RETURNING token_hash = $3`,
		userID, AuthTokenLoginCode, codeHash, LoginCodeMaxAttempts,
	).Scan(&matched)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !matched {
		return ErrNotFound
	}
	return nil
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
