package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// The store's authorization predicates (rv5-STRS-05). Each runs inside the
// caller's transaction, so the check and the write it guards see one state.

// requireAdminMember allows a current admin of the workspace.
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
// change the plan, and the project row itself (repo link, sync cursor):
// the level the projects_update RLS policy requires. A plain member can read it and leave notes, but
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
