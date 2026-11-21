package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// DeleteRoadmapNode removes a node; its subtree cascades via the parent_id
// FK's ON DELETE CASCADE, evidence with it. It takes a manager of the node
// (requireNodeManager): a delegate deletes below the node they were handed,
// never that node. A subtree holding evidence or done or validated nodes is
// deleted only with force, else ReplaceConflictError with History and the
// counts; a forced discard of history leaves a note in the project feed
// (rv5-TREEB-10).
func (s *Store) DeleteRoadmapNode(ctx context.Context, actorID, nodeID uuid.UUID, force bool) error {
	var projectID uuid.UUID
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var title string
		if err := tx.QueryRow(ctx, `SELECT project_id, title FROM insideout.roadmap_nodes WHERE id = $1`, nodeID).Scan(&projectID, &title); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if err := requireNodeManager(ctx, tx, nodeID, actorID); err != nil {
			return err
		}
		var c ReplaceConflictError
		if err := tx.QueryRow(ctx, `WITH RECURSIVE `+descendantsOf("$1")+`
			SELECT count(*), count(*) FILTER (WHERE n.status = 'done'), count(*) FILTER (WHERE n.status = 'validated'),
				(SELECT count(*) FROM insideout.roadmap_evidence e WHERE e.node_id IN (SELECT id FROM descendants))
			FROM insideout.roadmap_nodes n WHERE n.id IN (SELECT id FROM descendants)`, nodeID,
		).Scan(&c.LiveCount, &c.DoneCount, &c.ValidatedCount, &c.EvidenceCount); err != nil {
			return err
		}
		history := c.EvidenceCount+c.DoneCount+c.ValidatedCount > 0
		if history && !force {
			c.History = true
			return &c
		}
		if _, err := tx.Exec(ctx, `DELETE FROM insideout.roadmap_nodes WHERE id = $1`, nodeID); err != nil {
			return err
		}
		if !history {
			return nil
		}
		_, err := addProjectUpdateTx(ctx, tx, actorID, projectID, "note",
			fmt.Sprintf("Deleted %q (%d nodes), discarding %s", title, c.LiveCount, discardedHistory(&c)))
		return err
	})
	if err != nil {
		return err
	}
	s.noteRoadmapChange(projectID)
	return nil
}

// discardedHistory names what a forced delete or rebuild threw away.
func discardedHistory(c *ReplaceConflictError) string {
	return fmt.Sprintf("%d evidence, %d done and %d validated", c.EvidenceCount, c.DoneCount, c.ValidatedCount)
}
