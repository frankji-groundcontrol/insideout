package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type User struct {
	ID              uuid.UUID
	Email           string
	PasswordHash    string
	Username        string
	AvatarURL       *string
	Bio             string
	Keywords        []string
	Role            string
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

const userColumns = `id, email, password_hash, username, avatar_url, bio, keywords, role, email_verified_at, created_at, updated_at`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Username, &u.AvatarURL, &u.Bio, &u.Keywords, &u.Role, &u.EmailVerifiedAt, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateUser and GetUserByEmail deliberately run without withUserContext
// — registration and login both precede having an authenticated identity
// to set app.user_id to. The users RLS policy (db/migrations/
// 20260720150000_row_level_security.sql) treats app.user_id being unset
// as this trusted pre-auth context, since insideout_app is only ever
// called by our own backend.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash, username string) (*User, error) {
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO insideout.users (email, password_hash, username)
		VALUES ($1, $2, $3)
		RETURNING `+userColumns,
		email, passwordHash, username,
	)
	u, err := scanUser(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrConflict
		}
		return nil, err
	}
	return u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	row := s.Pool.QueryRow(ctx, `SELECT `+userColumns+` FROM insideout.users WHERE email = $1`, email)
	return scanUser(row)
}

// GetUserByID is only ever called for a caller loading their own profile
// (see internal/api/me.go), so id doubles as the RLS actor.
func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u *User
	err := s.withUserContext(ctx, id, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `SELECT `+userColumns+` FROM insideout.users WHERE id = $1`, id)
		var scanErr error
		u, scanErr = scanUser(row)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}

// UpdatePasswordHash overwrites the caller's own password hash — used by
// the login handler to transparently upgrade a migrated bcrypt hash
// (from the old juanleme/auth.users table) to argon2id on first
// successful login. See auth.IsBcryptHash.
func (s *Store) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	return s.withUserContext(ctx, id, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE insideout.users SET password_hash = $2 WHERE id = $1`, id, hash)
		return err
	})
}

// UpdateProfile updates self-editable fields only. Callers must never
// expose a path to change Role through this function — role changes are
// admin-only and have no user-facing endpoint in v1.
func (s *Store) UpdateProfile(ctx context.Context, id uuid.UUID, username, bio string, keywords []string, avatarURL *string) (*User, error) {
	var u *User
	err := s.withUserContext(ctx, id, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			UPDATE insideout.users
			SET username = $2, bio = $3, keywords = $4, avatar_url = $5
			WHERE id = $1
			RETURNING `+userColumns,
			id, username, bio, keywords, avatarURL,
		)
		var scanErr error
		u, scanErr = scanUser(row)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}

// SharesWorkspaceWith reports whether the two users are both members of at
// least one common workspace — used by the users-read authorization rule.
func (s *Store) SharesWorkspaceWith(ctx context.Context, viewerID, targetID uuid.UUID) (bool, error) {
	var exists bool
	err := s.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM insideout.workspace_memberships m1
			JOIN insideout.workspace_memberships m2 ON m1.workspace_id = m2.workspace_id
			WHERE m1.user_id = $1 AND m2.user_id = $2
		)`, viewerID, targetID,
	).Scan(&exists)
	return exists, err
}
