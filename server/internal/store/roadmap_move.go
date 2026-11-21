package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MoveRoadmapNode reparents a node (newParentID nil = make it a root) and sets
// its sibling position. It refuses to move a node under itself or one of its
// own descendants, which would create a cycle. It takes a manager of the node
// (requireNodeManager) and a writer of the new parent, or a plan writer for a
// new root: a delegate moves what lies below their node, within what they own.
func (s *Store) MoveRoadmapNode(ctx context.Context, actorID, nodeID uuid.UUID, newParentID *uuid.UUID, position int) (*RoadmapNode, error) {
	var n *RoadmapNode
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		projectID, err := authorizeMove(ctx, tx, actorID, nodeID, newParentID)
		if err != nil {
			return err
		}

		// Serialize all structural writes within the project so two opposing
		// moves can't both pass the cycle guard and forge a cycle under
		// READ COMMITTED (same advisory-lock idiom as ReplaceRoadmapTree).
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, projectID.String()); err != nil {
			return err
		}

		if newParentID != nil {
			if err := checkMoveTarget(ctx, tx, nodeID, *newParentID); err != nil {
				return err
			}
		}
		if err := shiftSiblingsForMove(ctx, tx, projectID, nodeID, newParentID, position); err != nil {
			return err
		}
		row := tx.QueryRow(ctx, `
				UPDATE insideout.roadmap_nodes
				SET parent_id = $2, position = $3, updated_by = $4
				WHERE id = $1
				RETURNING `+roadmapNodeColumns,
			nodeID, newParentID, position, actorID,
		)
		written, err := scanRoadmapNode(row)
		if err != nil {
			return err
		}
		n, err = readNodeViewTx(ctx, tx, actorID, written.ID)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.noteRoadmapChange(n.ProjectID)
	return n, nil
}

// authorizeMove finds the node's project and checks the actor manages the node
// and may write under its new parent (a plan writer, for a new root).
func authorizeMove(ctx context.Context, tx pgx.Tx, actorID, nodeID uuid.UUID, newParentID *uuid.UUID) (uuid.UUID, error) {
	var projectID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT project_id FROM insideout.roadmap_nodes WHERE id = $1`, nodeID).Scan(&projectID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, err
	}
	if err := requireNodeManager(ctx, tx, nodeID, actorID); err != nil {
		return uuid.Nil, err
	}
	if newParentID == nil {
		if err := requirePlanWriter(ctx, tx, projectID, actorID); err != nil {
			return uuid.Nil, err
		}
	} else if err := requireNodeWriter(ctx, tx, *newParentID, actorID); err != nil {
		return uuid.Nil, err
	}
	return projectID, nil
}

// checkMoveTarget is the cycle guard and same-project check in one recursive
// walk: the new parent must exist in this project and must NOT be the node
// itself or one of its descendants.
func checkMoveTarget(ctx context.Context, tx pgx.Tx, nodeID, newParentID uuid.UUID) error {
	var bad bool
	err := tx.QueryRow(ctx, `
			WITH RECURSIVE `+descendantsOf("$1")+`
			SELECT NOT EXISTS (
				SELECT 1 FROM insideout.roadmap_nodes target
				WHERE target.id = $2 AND target.project_id = (SELECT project_id FROM descendants LIMIT 1)
			) OR EXISTS (SELECT 1 FROM descendants WHERE id = $2)`,
		nodeID, newParentID,
	).Scan(&bad)
	if err != nil {
		return err
	}
	if bad {
		return ErrConflict
	}
	return nil
}

// shiftSiblingsForMove closes the gap the node leaves, then opens the slot it
// takes, so two siblings never share a position (BUG-017). The node is
// excluded from both so a same-parent move does not count itself, and the
// caller's per-project advisory lock keeps the two updates atomic with the
// rest of the move.
func shiftSiblingsForMove(ctx context.Context, tx pgx.Tx, projectID, nodeID uuid.UUID, newParentID *uuid.UUID, position int) error {
	if _, err := tx.Exec(ctx, `
			UPDATE insideout.roadmap_nodes
			SET position = position - 1
			WHERE project_id = $1
			  AND parent_id IS NOT DISTINCT FROM (SELECT parent_id FROM insideout.roadmap_nodes WHERE id = $2)
			  AND id <> $2
			  AND position > (SELECT position FROM insideout.roadmap_nodes WHERE id = $2)`,
		projectID, nodeID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
			UPDATE insideout.roadmap_nodes
			SET position = position + 1
			WHERE project_id = $1
			  AND parent_id IS NOT DISTINCT FROM $2
			  AND id <> $3
			  AND position >= $4`,
		projectID, newParentID, nodeID, position)
	return err
}
