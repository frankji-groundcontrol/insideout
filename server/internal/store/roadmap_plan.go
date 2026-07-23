package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RoadmapPlanNode is a generated roadmap tree (from the AI planner or the
// template fallback) before it's persisted. Children branch in parallel.
type RoadmapPlanNode struct {
	Title       string
	Description string
	Children    []RoadmapPlanNode
}

// ReplaceRoadmapTree atomically swaps a project's whole roadmap for the given
// tree — used by "build the MVP" generation. Returns the number of nodes
// written. Any workspace member may regenerate (roadmap is collaborative).
func (s *Store) ReplaceRoadmapTree(ctx context.Context, actorID, projectID uuid.UUID, root RoadmapPlanNode) (int, error) {
	count := 0
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		wsID, err := projectWorkspace(ctx, tx, projectID)
		if err != nil {
			return err
		}
		if _, err := requireMember(ctx, tx, wsID, actorID); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `DELETE FROM insideout.roadmap_nodes WHERE project_id = $1`, projectID); err != nil {
			return err
		}

		var insert func(parentID *uuid.UUID, n RoadmapPlanNode) error
		insert = func(parentID *uuid.UUID, n RoadmapPlanNode) error {
			var id uuid.UUID
			err := tx.QueryRow(ctx, `
				INSERT INTO insideout.roadmap_nodes (project_id, parent_id, title, description, position)
				SELECT $1, $2, $3, $4, COALESCE(MAX(position) + 1, 0)
				FROM insideout.roadmap_nodes
				WHERE project_id = $1 AND parent_id IS NOT DISTINCT FROM $2
				RETURNING id`,
				projectID, parentID, n.Title, n.Description,
			).Scan(&id)
			if err != nil {
				return err
			}
			count++
			for _, c := range n.Children {
				if err := insert(&id, c); err != nil {
					return err
				}
			}
			return nil
		}
		return insert(nil, root)
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetRoadmapNode returns one node for a workspace member (used by AI expand).
func (s *Store) GetRoadmapNode(ctx context.Context, actorID, nodeID uuid.UUID) (*RoadmapNode, error) {
	var n *RoadmapNode
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if _, err := nodeWorkspace(ctx, tx, nodeID); err != nil {
			return err
		}
		row := tx.QueryRow(ctx, `SELECT `+roadmapNodeColumns+` FROM insideout.roadmap_nodes WHERE id = $1`, nodeID)
		var scanErr error
		n, scanErr = scanRoadmapNode(row)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return n, nil
}

// EnsureProjectForPrd returns the project a PRD builds into, creating and
// linking one (titled from the PRD) on first use. This is the PRD → reality
// bridge: an approved/refined PRD becomes a project with a buildable roadmap.
func (s *Store) EnsureProjectForPrd(ctx context.Context, actorID, prdID uuid.UUID) (*Project, error) {
	var p *Project
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var workspaceID uuid.UUID
		var linkedProject *uuid.UUID
		var title string
		err := tx.QueryRow(ctx, `SELECT workspace_id, project_id, title FROM insideout.prds WHERE id = $1`, prdID).
			Scan(&workspaceID, &linkedProject, &title)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if _, err := requireMember(ctx, tx, workspaceID, actorID); err != nil {
			return err
		}

		if linkedProject != nil {
			row := tx.QueryRow(ctx, `SELECT `+projectColumns+` FROM insideout.projects WHERE id = $1`, *linkedProject)
			var scanErr error
			p, scanErr = scanProject(row)
			return scanErr
		}

		row := tx.QueryRow(ctx, `
			INSERT INTO insideout.projects (workspace_id, title, description, owner_id, created_by)
			VALUES ($1, $2, '', $3, $3)
			RETURNING `+projectColumns,
			workspaceID, title, actorID,
		)
		proj, err := scanProject(row)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE insideout.prds SET project_id = $2 WHERE id = $1`, prdID, proj.ID); err != nil {
			return err
		}
		p = proj
		return nil
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}
