package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ProposalItem is one structured action carried by a proposal; today
// only add_node, with an optional parent hint (node title).
type ProposalItem struct {
	ID         uuid.UUID
	UpdateID   uuid.UUID
	Action     string
	Title      string
	ParentHint string
}

// AddProposalItems records a proposal's structured items (propose time).
// Caller has already written the project_updates row in the same
// conceptual transaction; items are capped by the API layer.
func (s *Store) AddProposalItems(ctx context.Context, actorID, updateID uuid.UUID, items []ProposalItem) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		for _, it := range items {
			if _, err := tx.Exec(ctx, `
				INSERT INTO insideout.proposal_items (update_id, action, title, parent_hint)
				VALUES ($1, $2, $3, $4)`, updateID, it.Action, it.Title, it.ParentHint); err != nil {
				return err
			}
		}
		return nil
	})
}

// ListProposalItems returns a proposal's items.
func (s *Store) ListProposalItems(ctx context.Context, actorID, updateID uuid.UUID) ([]ProposalItem, error) {
	var out []ProposalItem
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, update_id, action, title, parent_hint
			FROM insideout.proposal_items WHERE update_id = $1
			ORDER BY created_at, id`, updateID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var it ProposalItem
			if err := rows.Scan(&it.ID, &it.UpdateID, &it.Action, &it.Title, &it.ParentHint); err != nil {
				return err
			}
			out = append(out, it)
		}
		return rows.Err()
	})
	return out, err
}

// ApplyProposalItemsForUpdate resolves the proposal's project (under
// the decider's RLS context) and applies its items.
func (s *Store) ApplyProposalItemsForUpdate(ctx context.Context, actorID, updateID uuid.UUID) ([]uuid.UUID, error) {
	var projectID uuid.UUID
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT project_id FROM insideout.project_updates WHERE id = $1`, updateID).Scan(&projectID)
	})
	if err != nil {
		return nil, err
	}
	return s.ApplyProposalItems(ctx, actorID, projectID, updateID)
}

// ApplyProposalItems creates the proposal's add_node items as real
// roadmap nodes, executed as the deciding human (actorID). parentHint
// matches a node title in the project (first match); empty → root.
// Returns the created node ids.
func (s *Store) ApplyProposalItems(ctx context.Context, actorID, projectID, updateID uuid.UUID) ([]uuid.UUID, error) {
	items, err := s.ListProposalItems(ctx, actorID, updateID)
	if err != nil {
		return nil, err
	}
	created := make([]uuid.UUID, 0, len(items))
	for _, it := range items {
		if it.Action != "add_node" {
			continue
		}
		node, err := s.CreateRoadmapNode(ctx, actorID, projectID, resolveParent(ctx, s, actorID, projectID, it.ParentHint), it.Title, "proposed: "+it.Title)
		if err != nil {
			return created, err
		}
		created = append(created, node.ID)
	}
	return created, nil
}

func resolveParent(ctx context.Context, s *Store, actorID, projectID uuid.UUID, hint string) *uuid.UUID {
	if hint == "" {
		return nil
	}
	nodes, err := s.ListRoadmap(ctx, actorID, projectID)
	if err != nil {
		return nil
	}
	for _, n := range nodes {
		if n.Title == hint {
			id := n.ID
			return &id
		}
	}
	return nil
}
