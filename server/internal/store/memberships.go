package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Membership struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	Role        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type MemberWithUser struct {
	Membership
	Username string
	Email    string
}

// GetMembership is always called with userID as both the RLS actor and
// the row being looked up (checking "is userID a member") — self-
// referential against workspace_memberships_select's own EXISTS check,
// which is fine since a row is always visible to itself.
func (s *Store) GetMembership(ctx context.Context, workspaceID, userID uuid.UUID) (*Membership, error) {
	var m Membership
	err := s.withUserContext(ctx, userID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT id, workspace_id, user_id, role, created_at, updated_at
			FROM insideout.workspace_memberships WHERE workspace_id = $1 AND user_id = $2`,
			workspaceID, userID,
		).Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.Role, &m.CreatedAt, &m.UpdatedAt)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// ListMembers requires actorID to already be a member (enforced by RLS
// returning zero rows for non-members, not by a separate check — callers
// already verify membership first via GetMembership for the 404 case).
func (s *Store) ListMembers(ctx context.Context, actorID, workspaceID uuid.UUID) ([]MemberWithUser, error) {
	var out []MemberWithUser
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT m.id, m.workspace_id, m.user_id, m.role, m.created_at, m.updated_at, u.username, u.email
			FROM insideout.workspace_memberships m
			JOIN insideout.users u ON u.id = m.user_id
			WHERE m.workspace_id = $1
			ORDER BY m.created_at ASC`, workspaceID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var m MemberWithUser
			if err := rows.Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.Role, &m.CreatedAt, &m.UpdatedAt, &m.Username, &m.Email); err != nil {
				return err
			}
			out = append(out, m)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateMemberRole requires actorID to be a workspace admin, re-checked in
// the same transaction as the write (TOCTOU-safe).
func (s *Store) UpdateMemberRole(ctx context.Context, actorID, workspaceID, targetUserID uuid.UUID, newRole string) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if err := requireAdminMember(ctx, tx, workspaceID, actorID); err != nil {
			return err
		}

		tag, err := tx.Exec(ctx, `
			UPDATE insideout.workspace_memberships SET role = $3
			WHERE workspace_id = $1 AND user_id = $2`,
			workspaceID, targetUserID, newRole,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// RemoveMember allows a workspace admin to remove anyone, or a member to
// remove themselves (self-leave).
func (s *Store) RemoveMember(ctx context.Context, actorID, workspaceID, targetUserID uuid.UUID) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if actorID != targetUserID {
			if err := requireAdminMember(ctx, tx, workspaceID, actorID); err != nil {
				return err
			}
		}

		tag, err := tx.Exec(ctx, `
			DELETE FROM insideout.workspace_memberships WHERE workspace_id = $1 AND user_id = $2`,
			workspaceID, targetUserID,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	})
}

func requireAdminMember(ctx context.Context, tx pgx.Tx, workspaceID, actorID uuid.UUID) error {
	var role string
	err := tx.QueryRow(ctx, `
		SELECT role FROM insideout.workspace_memberships
		WHERE workspace_id = $1 AND user_id = $2`,
		workspaceID, actorID,
	).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	if role != "admin" {
		return ErrForbidden
	}
	return nil
}

// requireMember is used by domain packages (projects, ideas, prds) that
// need "must currently be a member" re-checked inside their own write
// transactions.
func requireMember(ctx context.Context, tx pgx.Tx, workspaceID, userID uuid.UUID) (role string, err error) {
	// No FOR KEY SHARE: with FORCE RLS on a table whose policy calls
	// _is_member, row locks silently return zero rows (BUG-007). The
	// check and the write still share this transaction.
	err = tx.QueryRow(ctx, `
		SELECT role FROM insideout.workspace_memberships
		WHERE workspace_id = $1 AND user_id = $2`,
		workspaceID, userID,
	).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrForbidden
	}
	return role, err
}
