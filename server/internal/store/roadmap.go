package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RoadmapNode is one node in a project's branched roadmap tree. A NULL
// ParentID marks a root; siblings share a ParentID and are developed in
// parallel, ordered by Position. See docs/plans/2026-07-22-idea-to-reality.md.
type RoadmapNode struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	ParentID    *uuid.UUID
	Title       string
	Description string
	Status      string
	Position    int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// RoadmapNodeFields are the mutable fields of a node (status transitions and
// content edits). Structure changes go through MoveRoadmapNode.
type RoadmapNodeFields struct {
	Title       string
	Description string
	Status      string
}

const roadmapNodeColumns = `id, project_id, parent_id, title, description, status, position, created_at, updated_at`

func scanRoadmapNode(row pgx.Row) (*RoadmapNode, error) {
	var n RoadmapNode
	err := row.Scan(&n.ID, &n.ProjectID, &n.ParentID, &n.Title, &n.Description, &n.Status, &n.Position, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &n, err
}

// projectWorkspace resolves the workspace a project belongs to — the scope of
// every roadmap authorization check.
func projectWorkspace(ctx context.Context, tx pgx.Tx, projectID uuid.UUID) (uuid.UUID, error) {
	var wsID uuid.UUID
	err := tx.QueryRow(ctx, `SELECT workspace_id FROM insideout.projects WHERE id = $1`, projectID).Scan(&wsID)
	if errors.Is(err, pgx.ErrNoRows) {
		return wsID, ErrNotFound
	}
	return wsID, err
}

// nodeWorkspace resolves the workspace a node belongs to via its project.
func nodeWorkspace(ctx context.Context, tx pgx.Tx, nodeID uuid.UUID) (uuid.UUID, error) {
	var wsID uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT p.workspace_id FROM insideout.roadmap_nodes n
		JOIN insideout.projects p ON p.id = n.project_id
		WHERE n.id = $1`, nodeID).Scan(&wsID)
	if errors.Is(err, pgx.ErrNoRows) {
		return wsID, ErrNotFound
	}
	return wsID, err
}

// CreateRoadmapNode appends a node under parentID (nil = root) at the next
// sibling position. The roadmap is collaborative: any workspace member may add.
func (s *Store) CreateRoadmapNode(ctx context.Context, actorID, projectID uuid.UUID, parentID *uuid.UUID, title, description string) (*RoadmapNode, error) {
	var n *RoadmapNode
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		wsID, err := projectWorkspace(ctx, tx, projectID)
		if err != nil {
			return err
		}
		if _, err := requireMember(ctx, tx, wsID, actorID); err != nil {
			return err
		}

		if parentID != nil {
			// The parent must live in the same project, else the tree could
			// silently span projects.
			var parentProject uuid.UUID
			err := tx.QueryRow(ctx, `SELECT project_id FROM insideout.roadmap_nodes WHERE id = $1`, *parentID).Scan(&parentProject)
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			if err != nil {
				return err
			}
			if parentProject != projectID {
				return ErrConflict
			}
		}

		// IS NOT DISTINCT FROM matches parent_id = $2 including NULL = NULL.
		row := tx.QueryRow(ctx, `
			INSERT INTO insideout.roadmap_nodes (project_id, parent_id, title, description, position)
			SELECT $1, $2, $3, $4,
			       COALESCE(MAX(position) + 1, 0)
			FROM insideout.roadmap_nodes
			WHERE project_id = $1 AND parent_id IS NOT DISTINCT FROM $2
			RETURNING `+roadmapNodeColumns,
			projectID, parentID, title, description,
		)
		var scanErr error
		n, scanErr = scanRoadmapNode(row)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return n, nil
}

// ListRoadmap returns every node of a project, roots first then grouped by
// parent, ordered by position — the frontend assembles the tree from this
// flat slice. Any workspace member may read.
func (s *Store) ListRoadmap(ctx context.Context, actorID, projectID uuid.UUID) ([]RoadmapNode, error) {
	var out []RoadmapNode
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		wsID, err := projectWorkspace(ctx, tx, projectID)
		if err != nil {
			return err
		}
		if _, err := requireMember(ctx, tx, wsID, actorID); err != nil {
			return err
		}

		rows, err := tx.Query(ctx, `
			SELECT `+roadmapNodeColumns+`
			FROM insideout.roadmap_nodes
			WHERE project_id = $1
			ORDER BY parent_id NULLS FIRST, position, created_at`, projectID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var n RoadmapNode
			if err := rows.Scan(&n.ID, &n.ProjectID, &n.ParentID, &n.Title, &n.Description, &n.Status, &n.Position, &n.CreatedAt, &n.UpdatedAt); err != nil {
				return err
			}
			out = append(out, n)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateRoadmapNode edits a node's content/status. Any workspace member.
func (s *Store) UpdateRoadmapNode(ctx context.Context, actorID, nodeID uuid.UUID, f RoadmapNodeFields) (*RoadmapNode, error) {
	var n *RoadmapNode
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		wsID, err := nodeWorkspace(ctx, tx, nodeID)
		if err != nil {
			return err
		}
		if _, err := requireMember(ctx, tx, wsID, actorID); err != nil {
			return err
		}

		row := tx.QueryRow(ctx, `
			UPDATE insideout.roadmap_nodes
			SET title = $2, description = $3, status = $4
			WHERE id = $1
			RETURNING `+roadmapNodeColumns,
			nodeID, f.Title, f.Description, f.Status,
		)
		var scanErr error
		n, scanErr = scanRoadmapNode(row)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return n, nil
}

// MoveRoadmapNode reparents a node (newParentID nil = make it a root) and sets
// its sibling position. It refuses to move a node under itself or one of its
// own descendants, which would create a cycle. Any workspace member.
func (s *Store) MoveRoadmapNode(ctx context.Context, actorID, nodeID uuid.UUID, newParentID *uuid.UUID, position int) (*RoadmapNode, error) {
	var n *RoadmapNode
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		wsID, err := nodeWorkspace(ctx, tx, nodeID)
		if err != nil {
			return err
		}
		if _, err := requireMember(ctx, tx, wsID, actorID); err != nil {
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
					UNION ALL
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

		row := tx.QueryRow(ctx, `
			UPDATE insideout.roadmap_nodes
			SET parent_id = $2, position = $3
			WHERE id = $1
			RETURNING `+roadmapNodeColumns,
			nodeID, newParentID, position,
		)
		var scanErr error
		n, scanErr = scanRoadmapNode(row)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return n, nil
}

// DeleteRoadmapNode removes a node; its subtree cascades via the parent_id
// FK's ON DELETE CASCADE. Any workspace member.
func (s *Store) DeleteRoadmapNode(ctx context.Context, actorID, nodeID uuid.UUID) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		wsID, err := nodeWorkspace(ctx, tx, nodeID)
		if err != nil {
			return err
		}
		if _, err := requireMember(ctx, tx, wsID, actorID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `DELETE FROM insideout.roadmap_nodes WHERE id = $1`, nodeID)
		return err
	})
}
