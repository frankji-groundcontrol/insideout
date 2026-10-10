package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// The validation leg of the product loop (FEAT-04 / REL-07, review 3):
// the human half the webhook kinds never cover. An observation is typed
// by a person — no invitations, no metric collection — and 'validated'
// is a milestone a human confirms with a reason, never a state a sync
// may claim.

// AddValidationObservation records one human-entered observation on a
// leaf node as evidence of the validation kind: what was observed, its
// source, and the date it was seen. Only the project owner or a
// workspace admin writes evidence (the table's INSERT policy); leaves
// only, so the observation names one piece of work. (node_id, kind,
// detail) dedupe applies: recording the same text again is a no-op.
func (s *Store) AddValidationObservation(ctx context.Context, actorID, nodeID uuid.UUID, observed, sourceURL string, observedAt time.Time) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var projectID uuid.UUID
		var children int
		if err := tx.QueryRow(ctx, `
			SELECT n.project_id,
			       (SELECT count(*) FROM insideout.roadmap_nodes c WHERE c.parent_id = n.id)
			FROM insideout.roadmap_nodes n WHERE n.id = $1`, nodeID,
		).Scan(&projectID, &children); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if children > 0 {
			// A branch with children would blur which work was observed.
			return ErrConflict
		}
		if err := requirePlanWriter(ctx, tx, projectID, actorID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO insideout.roadmap_evidence (node_id, kind, detail, source_url, observed_at)
			VALUES ($1, 'validation', $2, $3, $4)
			ON CONFLICT DO NOTHING`, nodeID, observed, sourceURL, observedAt)
		return err
	})
}

// ValidateRoadmapNode confirms a done node as validated, recording who,
// when and the human's reason (PRODUCT.md: completion is overridden by a
// human only with a reason). It is the only move into 'validated' —
// UpdateRoadmapNode refuses that transition — and the milestone stays on
// the row when the node reopens for the next version.
func (s *Store) ValidateRoadmapNode(ctx context.Context, actorID, nodeID uuid.UUID, reason string) (*RoadmapNode, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, ErrValidation
	}
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
		var current string
		if err := tx.QueryRow(ctx, `SELECT status FROM insideout.roadmap_nodes WHERE id = $1`, nodeID).Scan(&current); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		// Validation comes after the work is finished (PRODUCT.md's
		// implemented → released → validated); an unfinished node has
		// nothing to validate against yet.
		if current != "done" {
			return fmt.Errorf("%w: status %q cannot move to %q", ErrInvalidTransition, current, "validated")
		}
		row := tx.QueryRow(ctx, `
			UPDATE insideout.roadmap_nodes
			SET status = 'validated', validation_reason = $2, validated_by = $3, validated_at = now(), updated_by = $3
			WHERE id = $1
			RETURNING `+roadmapNodeColumns,
			nodeID, reason, actorID)
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
