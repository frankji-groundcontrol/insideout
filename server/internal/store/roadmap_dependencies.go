package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Node dependencies (2026-11-22 plan, "Dependency contract"): a node waits on
// other nodes of its project, across branches. The writer of the waiting node
// adds and removes links; any node of the project may be waited on.

var (
	// ErrDependencySelf: a node cannot wait on itself.
	ErrDependencySelf = fmt.Errorf("%w: a node cannot depend on itself", ErrValidation)
	// ErrDependencyInTree: a link between a node and its own ancestor or
	// descendant, which the tree already says.
	ErrDependencyInTree = fmt.Errorf("%w: a node cannot depend on its own ancestor or descendant", ErrConflict)
	// ErrDependencyCycle: the link would close a cycle of waiting.
	ErrDependencyCycle = fmt.Errorf("%w: this dependency would close a cycle", ErrConflict)
)

// dependsOnColumn is the node view's waited-on ids, oldest link first.
const dependsOnColumn = `ARRAY(SELECT d.depends_on_id FROM insideout.roadmap_dependencies d
		       WHERE d.node_id = n.id ORDER BY d.created_at, d.depends_on_id)`

// AddRoadmapDependency makes nodeID wait on dependsOnID and returns the view
// of nodeID. An existing link is no change. Both nodes must be visible and in
// one project (else ErrNotFound); the actor must write nodeID.
func (s *Store) AddRoadmapDependency(ctx context.Context, actorID, nodeID, dependsOnID uuid.UUID) (*RoadmapNode, error) {
	if nodeID == dependsOnID {
		return nil, ErrDependencySelf
	}
	var n *RoadmapNode
	changed := false
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if _, err := lockDependencyTx(ctx, tx, actorID, nodeID, &dependsOnID); err != nil {
			return err
		}
		if err := checkDependencyTx(ctx, tx, nodeID, dependsOnID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO insideout.roadmap_dependencies (node_id, depends_on_id, created_by)
			VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, nodeID, dependsOnID, actorID)
		if err != nil {
			return err
		}
		changed = tag.RowsAffected() > 0
		n, err = readNodeViewTx(ctx, tx, actorID, nodeID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if changed {
		s.noteRoadmapChange(n.ProjectID)
	}
	return n, nil
}

// RemoveRoadmapDependency drops the link nodeID → dependsOnID; ErrNotFound
// when there is none.
func (s *Store) RemoveRoadmapDependency(ctx context.Context, actorID, nodeID, dependsOnID uuid.UUID) error {
	var projectID uuid.UUID
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var err error
		if projectID, err = lockDependencyTx(ctx, tx, actorID, nodeID, nil); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `DELETE FROM insideout.roadmap_dependencies WHERE node_id = $1 AND depends_on_id = $2`,
			nodeID, dependsOnID)
		if err == nil && tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return err
	})
	if err != nil {
		return err
	}
	s.noteRoadmapChange(projectID)
	return nil
}

// lockDependencyTx checks the actor writes nodeID and, when given, that
// dependsOnID is a visible node of the same project, then takes the
// per-project advisory lock every tree write takes. It returns the project.
func lockDependencyTx(ctx context.Context, tx pgx.Tx, actorID, nodeID uuid.UUID, dependsOnID *uuid.UUID) (uuid.UUID, error) {
	var projectID uuid.UUID
	err := tx.QueryRow(ctx, `SELECT project_id FROM insideout.roadmap_nodes WHERE id = $1`, nodeID).Scan(&projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return projectID, ErrNotFound
	}
	if err != nil {
		return projectID, err
	}
	if dependsOnID != nil {
		var same bool
		err := tx.QueryRow(ctx, `SELECT project_id = $2 FROM insideout.roadmap_nodes WHERE id = $1`, *dependsOnID, projectID).Scan(&same)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && !same) {
			return projectID, ErrNotFound
		}
		if err != nil {
			return projectID, err
		}
	}
	if err := requireNodeWriter(ctx, tx, nodeID, actorID); err != nil {
		return projectID, err
	}
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, projectID.String())
	return projectID, err
}

// checkDependencyTx refuses a link between a node and its own ancestor or
// descendant, and one that closes a cycle: dependsOnID already waits on
// nodeID through other links. An existing link passes, so a re-add is no
// change.
func checkDependencyTx(ctx context.Context, tx pgx.Tx, nodeID, dependsOnID uuid.UUID) error {
	var exists, inTree, cycle bool
	err := tx.QueryRow(ctx, `
		WITH RECURSIVE up1 AS (
			SELECT id, parent_id FROM insideout.roadmap_nodes WHERE id = $1
			UNION
			SELECT n.id, n.parent_id FROM insideout.roadmap_nodes n JOIN up1 ON n.id = up1.parent_id
		), up2 AS (
			SELECT id, parent_id FROM insideout.roadmap_nodes WHERE id = $2
			UNION
			SELECT n.id, n.parent_id FROM insideout.roadmap_nodes n JOIN up2 ON n.id = up2.parent_id
		), reach AS (
			SELECT depends_on_id AS id FROM insideout.roadmap_dependencies WHERE node_id = $2
			UNION
			SELECT d.depends_on_id FROM insideout.roadmap_dependencies d JOIN reach r ON d.node_id = r.id
		)
		SELECT EXISTS (SELECT 1 FROM insideout.roadmap_dependencies WHERE node_id = $1 AND depends_on_id = $2),
		       EXISTS (SELECT 1 FROM up1 WHERE id = $2) OR EXISTS (SELECT 1 FROM up2 WHERE id = $1),
		       EXISTS (SELECT 1 FROM reach WHERE id = $1)`, nodeID, dependsOnID).Scan(&exists, &inTree, &cycle)
	switch {
	case err != nil:
		return err
	case exists:
		return nil
	case inTree:
		return ErrDependencyInTree
	case cycle:
		return ErrDependencyCycle
	}
	return nil
}
