package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MoveRoadmapNode reparents a node (newParentID nil = make it a root) and sets
// its sibling position. It refuses to move a node under itself or one of its
// own descendants, which would create a cycle. Only the project owner or a
// workspace admin.
func (s *Store) MoveRoadmapNode(ctx context.Context, actorID, nodeID uuid.UUID, newParentID *uuid.UUID, position int) (*RoadmapNode, error) {
	var n *RoadmapNode
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var projectID uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT project_id FROM insideout.roadmap_nodes WHERE id = $1`, nodeID).Scan(&projectID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if err := requirePlanWriter(ctx, tx, projectID, actorID); err != nil {
			return err
		}

		// Serialize all structural writes within the project so two opposing
		// moves can't both pass the cycle guard and forge a cycle under
		// READ COMMITTED (same advisory-lock idiom as ReplaceRoadmapTree).
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, projectID.String()); err != nil {
			return err
		}

		if newParentID != nil {
			// Cycle guard + same-project check in one recursive walk: the new
			// parent must exist in this project and must NOT be the node itself
			// or one of its descendants.
			var bad bool
			err := tx.QueryRow(ctx, `
				WITH RECURSIVE descendants AS (
					SELECT id, project_id FROM insideout.roadmap_nodes WHERE id = $1
					UNION
					SELECT n.id, n.project_id FROM insideout.roadmap_nodes n
					JOIN descendants d ON n.parent_id = d.id
				)
				SELECT NOT EXISTS (
					SELECT 1 FROM insideout.roadmap_nodes target
					WHERE target.id = $2 AND target.project_id = (SELECT project_id FROM descendants LIMIT 1)
				) OR EXISTS (SELECT 1 FROM descendants WHERE id = $2)`,
				nodeID, *newParentID,
			).Scan(&bad)
			if err != nil {
				return err
			}
			if bad {
				return ErrConflict
			}
		}

		// Close the gap the node leaves, then open the slot it takes, so
		// two siblings never share a position (BUG-017). The node is
		// excluded from both so a same-parent move does not count
		// itself, and the per-project advisory lock above keeps the
		// two updates atomic with the rest of the move.
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
		if _, err := tx.Exec(ctx, `
				UPDATE insideout.roadmap_nodes
				SET position = position + 1
				WHERE project_id = $1
				  AND parent_id IS NOT DISTINCT FROM $2
				  AND id <> $3
				  AND position >= $4`,
			projectID, newParentID, nodeID, position); err != nil {
			return err
		}
		row := tx.QueryRow(ctx, `
				UPDATE insideout.roadmap_nodes
				SET parent_id = $2, position = $3, updated_by = $4
				WHERE id = $1
				RETURNING `+roadmapNodeColumns,
			nodeID, newParentID, position, actorID,
		)
		var scanErr error
		n, scanErr = scanRoadmapNode(row)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	s.noteRoadmapChange(n.ProjectID)
	return n, nil
}
