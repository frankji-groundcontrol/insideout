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
	// FamilyID names the lineage the session belongs to: the id of the session
	// a login created, unchanged through every rotation (rv4-SEC-04). A session
	// written before the column existed is its own family.
	FamilyID uuid.UUID
}

// sessionColumns is the one column list every session query returns, in the
// order scanSession reads it.
const sessionColumns = `id, user_id, token_hash, expires_at, revoked_at, created_at, COALESCE(family_id, id)`

func scanSession(row pgx.Row) (*Session, error) {
	var sess Session
	if err := row.Scan(&sess.ID, &sess.UserID, &sess.TokenHash, &sess.ExpiresAt, &sess.RevokedAt, &sess.CreatedAt, &sess.FamilyID); err != nil {
		return nil, err
	}
	return &sess, nil
}

// CreateSession starts a new lineage: a login's session is its own family root.
func (s *Store) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*Session, error) {
	return scanSession(s.Pool.QueryRow(ctx, `
		INSERT INTO insideout.sessions (id, user_id, token_hash, expires_at, family_id)
		VALUES ($1, $2, $3, $4, $1)
		RETURNING `+sessionColumns,
		uuid.New(), userID, tokenHash, expiresAt,
	))
}

// GetActiveSessionByHash returns the session only if it is unexpired and
// unrevoked — the row disappearing from this query's result is exactly
// what makes refresh-token reuse (after rotation) fail closed.
func (s *Store) GetActiveSessionByHash(ctx context.Context, tokenHash string) (*Session, error) {
	sess, err := scanSession(s.Pool.QueryRow(ctx, `
		SELECT `+sessionColumns+`
		FROM insideout.sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()`,
		tokenHash,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return sess, err
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
// The successor joins the old session's family, so the lineage stays one unit
// however many times it rotates. Telling a harmless lost race from the reuse of
// a long-spent token is RevokeFamilyOnReuse's job, not this method's: here the
// loser of a race is simply refused.
func (s *Store) RotateSession(ctx context.Context, oldID, userID uuid.UUID, newTokenHash string, expiresAt time.Time) (*Session, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var family uuid.UUID
	err = tx.QueryRow(ctx,
		`UPDATE insideout.sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL RETURNING COALESCE(family_id, id)`,
		oldID,
	).Scan(&family)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrConflict // already rotated/revoked — refuse to mint a second session
	}
	if err != nil {
		return nil, err
	}

	sess, err := scanSession(tx.QueryRow(ctx, `
		INSERT INTO insideout.sessions (user_id, token_hash, expires_at, family_id)
		VALUES ($1, $2, $3, $4)
		RETURNING `+sessionColumns,
		userID, newTokenHash, expiresAt, family,
	))
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return sess, nil
}

// RevokeFamilyOnReuse answers a refresh token that is no longer live (rv4-SEC-04).
// If the token's session was revoked more than grace ago, the token is spent
// and being offered again: somebody holds a copy besides the client that
// rotated it, and no one can tell from here which of them is the owner. So it
// ends every live session of that family, newest included, and the owner signs
// in again. It returns the user and how many live sessions it ended; (Nil, 0)
// means nothing happened: the token is unknown, still live, expired unrevoked,
// or was spent inside the grace.
//
// The grace is what keeps legitimate concurrency alive. Two CLI or MCP
// processes sharing one login, two tabs, or a retried request present the same
// one-time token moments apart; the loser is refused by RotateSession, and its
// replay must not take the winner's fresh pair down with it. The comparison
// runs on the database clock so every server replica agrees.
//
// Sessions revoked another way (logout, password reset) match too, and ending
// their family is harmless: those paths already ended every live session in it.
func (s *Store) RevokeFamilyOnReuse(ctx context.Context, tokenHash string, grace time.Duration) (uuid.UUID, int, error) {
	rows, err := s.Pool.Query(ctx, `
		WITH spent AS (
			SELECT COALESCE(family_id, id) AS family
			FROM insideout.sessions
			WHERE token_hash = $1 AND revoked_at < now() - make_interval(secs => $2)
		)
		UPDATE insideout.sessions SET revoked_at = now()
		WHERE revoked_at IS NULL AND family_id IN (SELECT family FROM spent)
		RETURNING user_id`,
		tokenHash, grace.Seconds(),
	)
	if err != nil {
		return uuid.Nil, 0, err
	}
	users, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil || len(users) == 0 {
		return uuid.Nil, 0, err
	}
	return users[0], len(users), nil
}
