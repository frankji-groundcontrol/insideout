package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrLastAdmin refuses a change that would leave a workspace without an admin.
// Role changes, PRD approval and invites run on admins, JoinWorkspace writes
// members only and the membership INSERT rule allows an admin row only for the
// first member of a workspace, so an admin-less workspace can never be
// administered again. It matches ErrConflict: handlers answer 409.
var ErrLastAdmin = fmt.Errorf("%w: a workspace needs at least one admin", ErrConflict)

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

// ListMembers requires actorID to be a member, re-checked in the same
// transaction as the read: a non-member gets ErrForbidden, not an empty list
// that looks like a workspace with no members. RLS on workspace_memberships
// filters the same rows (20260819190000 restored the FORCE that 20260720153000
// had dropped), but a roster read shouldn't hinge on that table's RLS state —
// every other member-gated store call checks in Go too. The API still answers
// 404 up front via GetMembership.
func (s *Store) ListMembers(ctx context.Context, actorID, workspaceID uuid.UUID) ([]MemberWithUser, error) {
	var out []MemberWithUser
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if _, err := requireMember(ctx, tx, workspaceID, actorID); err != nil {
			return err
		}

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
// the same transaction as the write (TOCTOU-safe), and refuses (ErrLastAdmin)
// to take the role away from the workspace's only admin.
func (s *Store) UpdateMemberRole(ctx context.Context, actorID, workspaceID, targetUserID uuid.UUID, newRole string) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if err := lockWorkspaceAdmins(ctx, tx, workspaceID); err != nil {
			return err
		}
		if err := requireAdminMember(ctx, tx, workspaceID, actorID); err != nil {
			return err
		}
		if newRole != "admin" {
			if err := requireAnotherAdmin(ctx, tx, workspaceID, targetUserID); err != nil {
				return err
			}
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
// remove themselves (self-leave). The workspace's only admin cannot leave or
// be removed (ErrLastAdmin): they promote someone first, or delete the
// workspace.
func (s *Store) RemoveMember(ctx context.Context, actorID, workspaceID, targetUserID uuid.UUID) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if err := lockWorkspaceAdmins(ctx, tx, workspaceID); err != nil {
			return err
		}
		if actorID != targetUserID {
			if err := requireAdminMember(ctx, tx, workspaceID, actorID); err != nil {
				return err
			}
		}
		if err := requireAnotherAdmin(ctx, tx, workspaceID, targetUserID); err != nil {
			return err
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

// lockWorkspaceAdmins serialises the changes that can shrink a workspace's set
// of admins, for the rest of the transaction. Two admins demoting each other
// would otherwise each read "two admins" and both write. It is an advisory
// lock and not FOR UPDATE on the rows: row locks under FORCE RLS silently
// return zero rows when a policy calls another table (BUG-007), and the lock
// must come before the actor check so a loser re-reads who is still an admin.
// The key is per workspace, so unrelated workspaces never wait for each other.
func lockWorkspaceAdmins(ctx context.Context, tx pgx.Tx, workspaceID uuid.UUID) error {
	_, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtext('insideout.workspace_admins'), hashtext($1))`, workspaceID.String())
	return err
}

// requireAnotherAdmin returns ErrLastAdmin when targetUserID is an admin of the
// workspace and nobody else is. Call it with the admin lock held, before the
// write that takes the target's admin role (or membership) away.
func requireAnotherAdmin(ctx context.Context, tx pgx.Tx, workspaceID, targetUserID uuid.UUID) error {
	var targetIsAdmin bool
	var others int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(bool_or(user_id = $2), false), count(*) FILTER (WHERE user_id <> $2)
		FROM insideout.workspace_memberships
		WHERE workspace_id = $1 AND role = 'admin'`,
		workspaceID, targetUserID,
	).Scan(&targetIsAdmin, &others); err != nil {
		return err
	}
	if targetIsAdmin && others == 0 {
		return ErrLastAdmin
	}
	return nil
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

// requirePlanWriter allows the project owner or a workspace admin to
// change the plan. A plain member can read it and leave notes, but
// not add, edit, move, or delete roadmap nodes, and not file a
// proposal (BUG-020).
func requirePlanWriter(ctx context.Context, tx pgx.Tx, projectID, userID uuid.UUID) error {
	var workspaceID uuid.UUID
	var ownerID *uuid.UUID
	err := tx.QueryRow(ctx,
		`SELECT workspace_id, owner_id FROM insideout.projects WHERE id = $1`, projectID,
	).Scan(&workspaceID, &ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if ownerID != nil && *ownerID == userID {
		return nil
	}
	role, err := requireMember(ctx, tx, workspaceID, userID)
	if err != nil {
		return err
	}
	if role != "admin" {
		return ErrForbidden
	}
	return nil
}
