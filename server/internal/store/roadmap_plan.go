package store

import (
	"context"
	"errors"
	"fmt"

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
// roadmap holds LiveCount nodes the caller did not confirm overwriting, or,
// with History set, nodes with evidence or done or validated that the caller
// did not force away (rv5-FEAT-03). It matches ErrConflict so handlers map it
// to 409 and can surface the counts.
type ReplaceConflictError struct {
	LiveCount, EvidenceCount, DoneCount, ValidatedCount int
	History                                             bool
}

func (e *ReplaceConflictError) Error() string        { return "roadmap replace not confirmed" }
func (e *ReplaceConflictError) Is(target error) bool { return target == ErrConflict }

// ReplaceOptions are what a replace's caller confirmed.
type ReplaceOptions struct {
	// ExpectedCount is the live node count the caller saw; a non-empty
	// roadmap is only replaced when it matches.
	ExpectedCount *int
	// Force discards nodes that have evidence or are done or validated.
	Force bool
	// EmptyOnly refuses any non-empty roadmap with ErrRoadmapNotEmpty: an
	// agent token builds only onto an empty roadmap (rv5-FEAT-02).
	EmptyOnly bool
}

// ErrRoadmapNotEmpty is an EmptyOnly replace that met a live roadmap.
var ErrRoadmapNotEmpty = errors.New("roadmap is not empty")

// ReplaceRoadmapTree atomically swaps a project's whole roadmap for the given
// tree — used by "build the MVP" generation. Returns the number of nodes
// written. It deletes every node, so it takes the rule every plan write takes:
// the project owner or a workspace admin (BUG-020). The expected count below is
// a staleness guard, not an authorisation.
//
// Guard: a non-empty roadmap is only replaced when ExpectedCount confirms the
// exact live count, and, while any node has evidence or is done or validated
// (the delete cascades through all of it), only with Force; else
// ReplaceConflictError. The checks run behind a per-project advisory lock so
// two concurrent builds can't both pass them then both delete (check-then-act
// TOCTOU). An empty roadmap needs no confirm.
func (s *Store) ReplaceRoadmapTree(ctx context.Context, actorID, projectID uuid.UUID, opts ReplaceOptions, root RoadmapPlanNode) (int, error) {
	count := 0
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if err := requirePlanWriter(ctx, tx, projectID, actorID); err != nil {
			return err
		}

		// Serialize concurrent replacers of this project's tree for the rest of
		// the tx, so the count check + delete + insert below is atomic.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, projectID.String()); err != nil {
			return err
		}

		var c ReplaceConflictError
		if err := tx.QueryRow(ctx, `
			SELECT count(*), count(*) FILTER (WHERE status = 'done'), count(*) FILTER (WHERE status = 'validated'),
				(SELECT count(*) FROM insideout.roadmap_evidence e
				 JOIN insideout.roadmap_nodes en ON en.id = e.node_id WHERE en.project_id = $1)
			FROM insideout.roadmap_nodes WHERE project_id = $1`, projectID,
		).Scan(&c.LiveCount, &c.DoneCount, &c.ValidatedCount, &c.EvidenceCount); err != nil {
			return err
		}
		if opts.EmptyOnly && c.LiveCount > 0 {
			return ErrRoadmapNotEmpty
		}
		if c.LiveCount > 0 && (opts.ExpectedCount == nil || *opts.ExpectedCount != c.LiveCount) {
			return &c
		}
		if !opts.Force && c.EvidenceCount+c.DoneCount+c.ValidatedCount > 0 {
			c.History = true
			return &c
		}

		if _, err := tx.Exec(ctx, `DELETE FROM insideout.roadmap_nodes WHERE project_id = $1`, projectID); err != nil {
			return err
		}
		// A forced rebuild discards history: the feed keeps a note of it.
		if c.EvidenceCount+c.DoneCount+c.ValidatedCount > 0 {
			if _, err := addProjectUpdateTx(ctx, tx, actorID, projectID, "note",
				fmt.Sprintf("Rebuilt the roadmap (%d nodes), discarding %s", c.LiveCount, discardedHistory(&c))); err != nil {
				return err
			}
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
		if err := insert(nil, root); err != nil {
			return err
		}
		// Seed the first leaf in_progress with a deadline, so a freshly
		// built tree shows on Progress instead of filtering out entirely
		// (every new node would otherwise be pending + undeadlined). The
		// builder owns it: work in progress has one human owner (rv5-FEAT-05),
		// on the record as a delegate row by the builder (no feed row), so
		// hand-back and history agree.
		if _, err := tx.Exec(ctx, `
			WITH leaf AS (
				UPDATE insideout.roadmap_nodes leaf
				SET status = 'in_progress', deadline = now() + interval '7 days', owner_user_id = $2
				WHERE leaf.id = (
					SELECT n.id FROM insideout.roadmap_nodes n
					WHERE n.project_id = $1 AND n.parent_id IS NOT NULL
					ORDER BY n.created_at, n.position
					LIMIT 1
				)
				RETURNING leaf.id
			)
			INSERT INTO insideout.roadmap_node_assignments (node_id, to_user_id, assigned_by, kind)
			SELECT id, $2, $2, 'delegate' FROM leaf`, projectID, actorID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	s.noteRoadmapChange(projectID)
	return count, nil
}

// expandFailAt, when > 0, makes ExpandRoadmapNode fail the Nth child insert
// (1-based). Test-only seam to exercise single-transaction rollback — the
// table has no title-length CHECK to trip, so this is the no-mocks failure
// route. Never set in prod.
var expandFailAt int

// ExpandRoadmapNode inserts all proposed children under parentID in one
// transaction — either the whole AI expansion lands or none of it does.
// Children append after any existing siblings. It takes a writer of the parent
// (requireNodeWriter: a delegate breaks down inside their subtree), and each
// child goes in through createRoadmapNodeTx, behind the same per-project lock
// as a manual add.
func (s *Store) ExpandRoadmapNode(ctx context.Context, actorID, parentID uuid.UUID, children []RoadmapPlanNode) ([]RoadmapNode, error) {
	var out []RoadmapNode
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var projectID uuid.UUID
		err := tx.QueryRow(ctx, `SELECT project_id FROM insideout.roadmap_nodes WHERE id = $1`, parentID).Scan(&projectID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		// Up front, so an expansion with nothing to add is refused too.
		if err := requireNodeWriter(ctx, tx, parentID, actorID); err != nil {
			return err
		}
		var ids []uuid.UUID

		for i, c := range children {
			if expandFailAt > 0 && i+1 == expandFailAt {
				return errors.New("expand: injected failure")
			}
			n, err := createRoadmapNodeTx(ctx, tx, actorID, projectID, &parentID, c.Title, c.Description)
			if err != nil {
				return err
			}
			ids = append(ids, n.ID)
		}
		out, err = readNodeViewsTx(ctx, tx, actorID, ids)
		return err
	})
	if err != nil {
		return nil, err
	}
	if len(out) > 0 {
		s.noteRoadmapChange(out[0].ProjectID)
	}
	return out, nil
}

// PrdIDForProject returns the PRD a project was built from (prds.project_id),
// or ErrNotFound for a project with no PRD — the expand path's bridge back
// to the coaching conversation whose locale governs the planner's language.
func (s *Store) PrdIDForProject(ctx context.Context, actorID, projectID uuid.UUID) (uuid.UUID, error) {
	var prdID uuid.UUID
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT id FROM insideout.prds WHERE project_id = $1 ORDER BY created_at DESC LIMIT 1`, projectID).Scan(&prdID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
	if err != nil {
		return uuid.Nil, err
	}
	return prdID, nil
}

// EnsureProjectForPrd returns the project a PRD builds into, creating and
// linking one (titled from the PRD) on first use. This is the PRD → reality
// bridge: an approved PRD becomes a project with a buildable roadmap. Only the
// build confirm calls it. Creating the project takes the CreatePrdCommit rule,
// the PRD's author or a workspace admin (rv5-FEAT-01); an existing link is
// returned to any member, and the roadmap write that follows checks its own rule.
func (s *Store) EnsureProjectForPrd(ctx context.Context, actorID, prdID uuid.UUID) (*Project, error) {
	var p *Project
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		// Serialize concurrent first-builds of this PRD for the rest of the tx
		// (mirror ReplaceRoadmapTree's advisory lock). Without it two
		// overlapping builds both read project_id NULL below and both insert a
		// project, leaving one orphaned under a divergent roadmap. The loser
		// blocks until the winner commits, then reads the committed link and
		// returns the existing project.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, prdID.String()); err != nil {
			return err
		}

		var workspaceID uuid.UUID
		var linkedProject *uuid.UUID
		var title string
		var authorID uuid.UUID
		err := tx.QueryRow(ctx, `SELECT workspace_id, project_id, title, author_id FROM insideout.prds WHERE id = $1`, prdID).
			Scan(&workspaceID, &linkedProject, &title, &authorID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		role, err := requireMember(ctx, tx, workspaceID, actorID)
		if err != nil {
			return err
		}

		if linkedProject != nil {
			row := tx.QueryRow(ctx, `SELECT `+projectColumns+` FROM insideout.projects WHERE id = $1`, *linkedProject)
			var scanErr error
			p, scanErr = scanProject(row)
			return scanErr
		}
		if authorID != actorID && role != "admin" {
			return ErrForbidden
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
		tag, err := tx.Exec(ctx, `UPDATE insideout.prds SET project_id = $2 WHERE id = $1`, prdID, proj.ID)
		if err != nil {
			return err
		}
		// Row-level security answers a refused UPDATE with zero rows: roll the
		// project back rather than leave it unlinked.
		if tag.RowsAffected() != 1 {
			return ErrForbidden
		}
		p = proj
		return nil
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}
