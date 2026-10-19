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
	PasswordHash    string // only GetUserByEmail fills it: login verifies against it
	Username        string
	AvatarURL       *string
	Bio             string
	Keywords        []string
	Role            string
	SystemRole      string
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// userColumns is every users column the app selects, password_hash left out:
// insideout_app has no SELECT on it (20261018200000_users_read_grants.sql), so
// no SELECT or RETURNING built from this list can leak it or fail on it. A
// column added here needs its own GRANT SELECT in a migration.
const userColumns = `id, email, username, avatar_url, bio, keywords, role, system_role, email_verified_at, created_at, updated_at`

// scanUser scans a userColumns row; extra are further destinations for columns
// selected after them.
func scanUser(row pgx.Row, extra ...any) (*User, error) {
	var u User
	err := row.Scan(append([]any{&u.ID, &u.Email, &u.Username, &u.AvatarURL, &u.Bio, &u.Keywords, &u.Role, &u.SystemRole, &u.EmailVerifiedAt, &u.CreatedAt, &u.UpdatedAt}, extra...)...)
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

// GetUserByEmail is login's reader and the only one that carries the stored
// credential (PasswordHash). The app role cannot SELECT password_hash from the
// table, so it comes from insideout._credential_for_email, which answers only
// in the pre-auth context this runs in. An unknown email is ErrNotFound, as
// before, after the same single indexed lookup.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var hash string
	u, err := scanUser(s.Pool.QueryRow(ctx, `
		SELECT `+qualifyColumns("u", userColumns)+`, c.password_hash
		FROM insideout._credential_for_email($1) c
		JOIN insideout.users u ON u.id = c.id`, email), &hash)
	if err != nil {
		return nil, err
	}
	u.PasswordHash = hash
	return u, nil
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
