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

// ReplaceConflictError reports a tree-replace precondition failure: the live
// roadmap holds LiveCount nodes the caller did not confirm overwriting. It
// matches ErrConflict so handlers map it to 409 and can surface the count.
type ReplaceConflictError struct{ LiveCount int }

func (e *ReplaceConflictError) Error() string { return "roadmap replace not confirmed" }
func (e *ReplaceConflictError) Is(target error) bool { return target == ErrConflict }

// ReplaceRoadmapTree atomically swaps a project's whole roadmap for the given
// tree — used by "build the MVP" generation. Returns the number of nodes
// written. Any workspace member may regenerate (roadmap is collaborative).
//
// Guard: a non-empty roadmap is only replaced when expectedCount confirms the
// exact live count, else ReplaceConflictError. The count check runs behind a
// per-project advisory lock so two concurrent builds can't both pass the check
// then both delete (check-then-act TOCTOU). An empty roadmap needs no confirm.
func (s *Store) ReplaceRoadmapTree(ctx context.Context, actorID, projectID uuid.UUID, expectedCount *int, root RoadmapPlanNode) (int, error) {
	count := 0
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		wsID, err := projectWorkspace(ctx, tx, projectID)
		if err != nil {
			return err
		}
		if _, err := requireMember(ctx, tx, wsID, actorID); err != nil {
			return err
		}

		// Serialize concurrent replacers of this project's tree for the rest of
		// the tx, so the count check + delete + insert below is atomic.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, projectID.String()); err != nil {
			return err
		}

		var liveCount int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM insideout.roadmap_nodes WHERE project_id = $1`, projectID).Scan(&liveCount); err != nil {
			return err
		}
		if liveCount > 0 && (expectedCount == nil || *expectedCount != liveCount) {
			return &ReplaceConflictError{LiveCount: liveCount}
		}

		if _, err := tx.Exec(ctx, `DELETE FROM insideout.roadmap_nodes WHERE project_id = $1`, projectID); err != nil {
			return err
		}

		var insert func(parentID *uuid.UUID, n RoadmapPlanNode) error
		insert = func(parentID *uuid.UUID, n RoadmapPlanNode) error {
			var id uuid.UUID
			err := tx.QueryRow(ctx, `
				INSERT INTO insideout.roadmap_nodes (project_id, parent_id, title, description, position, created_by, updated_by)
				SELECT $1, $2, $3, $4, COALESCE(MAX(position) + 1, 0), $5, $5
				FROM insideout.roadmap_nodes
				WHERE project_id = $1 AND parent_id IS NOT DISTINCT FROM $2
				RETURNING id`,
				projectID, parentID, n.Title, n.Description, actorID,
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

// expandFailAt, when > 0, makes ExpandRoadmapNode fail the Nth child insert
// (1-based). Test-only seam to exercise single-transaction rollback — the
// table has no title-length CHECK to trip, so this is the no-mocks failure
// route. Never set in prod.
var expandFailAt int

// ExpandRoadmapNode inserts all proposed children under parentID in one
// transaction — either the whole AI expansion lands or none of it does.
// Children append after any existing siblings. Any workspace member.
func (s *Store) ExpandRoadmapNode(ctx context.Context, actorID, parentID uuid.UUID, children []RoadmapPlanNode) ([]RoadmapNode, error) {
	var out []RoadmapNode
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		wsID, err := nodeWorkspace(ctx, tx, parentID)
		if err != nil {
			return err
		}
		if _, err := requireMember(ctx, tx, wsID, actorID); err != nil {
			return err
		}
		var projectID uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT project_id FROM insideout.roadmap_nodes WHERE id = $1`, parentID).Scan(&projectID); err != nil {
			return err
		}

		for i, c := range children {
			if expandFailAt > 0 && i+1 == expandFailAt {
				return errors.New("expand: injected failure")
			}
			row := tx.QueryRow(ctx, `
				INSERT INTO insideout.roadmap_nodes (project_id, parent_id, title, description, position, created_by, updated_by)
				SELECT $1, $2, $3, $4, COALESCE(MAX(position) + 1, 0), $5, $5
				FROM insideout.roadmap_nodes
				WHERE project_id = $1 AND parent_id IS NOT DISTINCT FROM $2
				RETURNING `+roadmapNodeColumns,
				projectID, parentID, c.Title, c.Description, actorID,
			)
			n, err := scanRoadmapNode(row)
			if err != nil {
				return err
			}
			out = append(out, *n)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
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
