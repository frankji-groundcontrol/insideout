package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Invitation-only signup (qwenimage pattern). invitations and settings
// are trusted backend-only tables like sessions/auth_tokens — no RLS, no
// app-facing paths except the gated register handler and admin routes.

// Signup modes for the settings key "signup_mode".
const (
	SignupInvitationOnly = "invitation_only"
	SignupOpen           = "open"
	SignupClosed         = "closed"
)

// GetSetting returns a settings value as its raw JSON string (e.g.
// `"invitation_only"` — quotes included, since values are jsonb text).
func (s *Store) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := s.Pool.QueryRow(ctx, `
		SELECT value::text FROM insideout.settings WHERE key = $1`, key,
	).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// SignupMode returns the effective signup mode, failing closed to
// invitation_only when the setting is missing or malformed — an
// unconfigured deployment must not drift open.
func (s *Store) SignupMode(ctx context.Context) string {
	v, err := s.GetSetting(ctx, "signup_mode")
	if err != nil || v != `"`+SignupOpen+`"` && v != `"`+SignupClosed+`"` && v != `"`+SignupInvitationOnly+`"` {
		return SignupInvitationOnly
	}
	return v[1 : len(v)-1]
}

// SetSetting upserts a settings value (value must be valid JSON text).
func (s *Store) SetSetting(ctx context.Context, key, jsonValue string) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO insideout.settings (key, value) VALUES ($1, $2::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = now()`,
		key, jsonValue)
	return err
}

type Invitation struct {
	ID        uuid.UUID
	Email     string
	ExpiresAt time.Time
	UsedAt    *time.Time
	UsedBy    *uuid.UUID
	RevokedAt *time.Time
	CreatedBy *uuid.UUID
	CreatedAt time.Time
}

const invitationColumns = `id, email, expires_at, used_at, used_by, revoked_at, created_by, created_at`

func scanInvitation(row pgx.Row) (*Invitation, error) {
	var i Invitation
	err := row.Scan(&i.ID, &i.Email, &i.ExpiresAt, &i.UsedAt, &i.UsedBy, &i.RevokedAt, &i.CreatedBy, &i.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &i, nil
}

// CreateInvitation mints an invite code for one address and supersedes
// the address's prior unused invitation — one live invite per email, the
// same one-live-per-kind discipline as auth tokens.
func (s *Store) CreateInvitation(ctx context.Context, createdBy uuid.UUID, email, codeHash string, expiresAt time.Time) (*Invitation, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		DELETE FROM insideout.invitations
		WHERE email = $1 AND used_at IS NULL AND revoked_at IS NULL`,
		email,
	); err != nil {
		return nil, err
	}
	row := tx.QueryRow(ctx, `
		INSERT INTO insideout.invitations (email, code_hash, expires_at, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING `+invitationColumns,
		email, codeHash, expiresAt, createdBy,
	)
	inv, err := scanInvitation(row)
	if err != nil {
		return nil, err
	}
	return inv, tx.Commit(ctx)
}

// ListInvitations returns every invitation newest-first (single-tenant
// admin surface; the code hash is deliberately not selected).
func (s *Store) ListInvitations(ctx context.Context) ([]*Invitation, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT `+invitationColumns+` FROM insideout.invitations
		ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Invitation
	for rows.Next() {
		var i Invitation
		if err := rows.Scan(&i.ID, &i.Email, &i.ExpiresAt, &i.UsedAt, &i.UsedBy, &i.RevokedAt, &i.CreatedBy, &i.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &i)
	}
	return out, rows.Err()
}

// RevokeInvitation kills a still-usable invitation.
func (s *Store) RevokeInvitation(ctx context.Context, id uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE insideout.invitations SET revoked_at = now()
		WHERE id = $1 AND used_at IS NULL AND revoked_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ConsumeInvitation claims the address's invitation for this exact code
// atomically — the UPDATE is the claim, so two racing registrations
// cannot both spend it. Email-bound: a leaked code is useless for any
// other address. usedBy may be uuid.Nil at claim time (pre-registration);
// it lands as SQL NULL and is attributed after the user exists.
func (s *Store) ConsumeInvitation(ctx context.Context, email, codeHash string, usedBy uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE insideout.invitations
		SET used_at = now(), used_by = NULLIF($3::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
		WHERE email = $1 AND code_hash = $2
		  AND used_at IS NULL AND revoked_at IS NULL AND expires_at > now()`,
		email, codeHash, usedBy)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AttributeInvitation stamps the registered user onto their just-spent
// invitation (the claim happened pre-registration with a nil user).
// Best-effort by design.
func (s *Store) AttributeInvitation(ctx context.Context, email, codeHash string, usedBy uuid.UUID) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE insideout.invitations SET used_by = $3
		WHERE email = $1 AND code_hash = $2 AND used_at IS NOT NULL AND used_by IS NULL`,
		email, codeHash, usedBy)
	return err
}

// SetSystemRole promotes or demotes a user by email. Cross-user UPDATEs
// on users are blocked by RLS for the app role, so this runs on the
// owner pool (BYPASSRLS) — used by the promote-admin operator
// subcommand, never by an HTTP route.
func (s *Store) SetSystemRole(ctx context.Context, email, role string) error {
	tag, err := s.migratePoolOrMain().Exec(ctx, `
		UPDATE insideout.users SET system_role = $2, updated_at = now()
		WHERE email = $1`, email, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
