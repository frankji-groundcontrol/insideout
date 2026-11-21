package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RoadmapEvidence is one matched delivery-evidence row on a leaf node
// (commits, PRs, deployments — PRODUCT.md: evidence, never outcomes) or
// one human-entered validation observation (kind 'validation').
type RoadmapEvidence struct {
	ID        uuid.UUID
	NodeID    uuid.UUID
	Kind      string
	Detail    string
	SourceURL string
	CreatedAt time.Time
	// ObservedAt: the date the human saw a validation observation
	// (nil on the machine-appended kinds, which carry only CreatedAt).
	ObservedAt *time.Time
}

// AddRoadmapEvidence appends one evidence row as actorID (the webhook
// passes the project owner it resolved). Duplicate detail per node is
// idempotent (unique index) so GitHub redeliveries are free.
func (s *Store) AddRoadmapEvidence(ctx context.Context, actorID, nodeID uuid.UUID, kind, detail, sourceURL string) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var projectID uuid.UUID
		var ownerID *uuid.UUID
		if err := tx.QueryRow(ctx,
			`SELECT n.project_id, p.owner_id FROM insideout.roadmap_nodes n
			 JOIN insideout.projects p ON p.id = n.project_id WHERE n.id = $1`, nodeID,
		).Scan(&projectID, &ownerID); err != nil {
			return err
		}
		if ownerID == nil || *ownerID != actorID {
			var admin bool
			var ws uuid.UUID
			if err := tx.QueryRow(ctx,
				`SELECT p.workspace_id, insideout._is_admin(p.workspace_id, $2)
				 FROM insideout.projects p WHERE p.id = $1`, projectID, actorID).Scan(&ws, &admin); err != nil {
				return err
			}
			if !admin {
				return ErrForbidden
			}
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO insideout.roadmap_evidence (node_id, kind, detail, source_url)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT DO NOTHING`, nodeID, kind, detail, sourceURL)
		return err
	})
}

// ListRoadmapEvidence returns a node's evidence, newest first, or ErrNotFound
// for a node that does not exist or that actorID cannot see (rv5-PARC-10).
func (s *Store) ListRoadmapEvidence(ctx context.Context, actorID, nodeID uuid.UUID) ([]RoadmapEvidence, error) {
	// Membership rides the tables' RLS policies; withUserContext supplies
	// the identity.
	var out []RoadmapEvidence
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var visible bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM insideout.roadmap_nodes WHERE id = $1)`, nodeID).Scan(&visible); err != nil {
			return err
		}
		if !visible {
			return ErrNotFound
		}
		rows, err := tx.Query(ctx, `
			SELECT id, node_id, kind, detail, source_url, created_at, observed_at
			FROM insideout.roadmap_evidence WHERE node_id = $1
			ORDER BY created_at DESC, id LIMIT 200`, nodeID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var e RoadmapEvidence
			if err := rows.Scan(&e.ID, &e.NodeID, &e.Kind, &e.Detail, &e.SourceURL, &e.CreatedAt, &e.ObservedAt); err != nil {
				return err
			}
			out = append(out, e)
		}
		return rows.Err()
	})
	return out, err
}

// IsRoadmapLeaf reports whether nodeID belongs to projectID and has no
// children; failures count as non-leaf (evidence attaches to leaves
// only, so the safe default is no write). Runs as actorID for RLS.
func (s *Store) IsRoadmapLeaf(ctx context.Context, actorID, projectID, nodeID uuid.UUID) bool {
	var leaf bool
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM insideout.roadmap_nodes
				WHERE id = $1 AND project_id = $2
			) AND NOT EXISTS (
				SELECT 1 FROM insideout.roadmap_nodes c
				WHERE c.parent_id = $1
			)`, nodeID, projectID).Scan(&leaf)
	})
	return err == nil && leaf
}
