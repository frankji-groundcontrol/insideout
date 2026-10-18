package store

import (
	"context"
	"errors"

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

// AddProposalItems records a proposal's structured items (propose time),
// in the order given. Position is that order: created_at is one
// statement timestamp, so it cannot order a batch (BUG-019). Caller
// has already written the project_updates row; items are capped by
// the API layer.
func (s *Store) AddProposalItems(ctx context.Context, actorID, updateID uuid.UUID, items []ProposalItem) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		return addProposalItemsTx(ctx, tx, updateID, items)
	})
}

func addProposalItemsTx(ctx context.Context, tx pgx.Tx, updateID uuid.UUID, items []ProposalItem) error {
	for i, it := range items {
		if _, err := tx.Exec(ctx, `
				INSERT INTO insideout.proposal_items (update_id, action, title, parent_hint, position)
				VALUES ($1, $2, $3, $4, $5)`, updateID, it.Action, it.Title, it.ParentHint, i); err != nil {
			return err
		}
	}
	return nil
}

// AddProposal files an agent proposal (the project_updates row, plan writers
// only) together with its structured items in one transaction, so a failure
// part-way cannot leave a proposal without its items for a human to accept.
// The items' UpdateID is ignored: they belong to the update this call creates.
func (s *Store) AddProposal(ctx context.Context, actorID, projectID uuid.UUID, content string, items []ProposalItem) (*ProjectUpdate, error) {
	var u *ProjectUpdate
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var err error
		if u, err = addProjectUpdateTx(ctx, tx, actorID, projectID, "agent_proposal", content); err != nil {
			return err
		}
		return addProposalItemsTx(ctx, tx, u.ID, items)
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}

// ListProposalItems returns a proposal's items.
func (s *Store) ListProposalItems(ctx context.Context, actorID, updateID uuid.UUID) ([]ProposalItem, error) {
	var out []ProposalItem
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
				SELECT id, update_id, action, title, parent_hint
				FROM insideout.proposal_items WHERE update_id = $1
				ORDER BY position, created_at, id`, updateID)
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
// roadmap nodes, executed as the deciding human (actorID), all in one
// transaction (see applyProposalItemsTx). parentHint matches a node
// title in the project (first match); empty → root. Returns the created
// node ids: only the items not applied before, none on a repeat.
func (s *Store) ApplyProposalItems(ctx context.Context, actorID, projectID, updateID uuid.UUID) ([]uuid.UUID, error) {
	var created []uuid.UUID
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var err error
		created, err = applyProposalItemsTx(ctx, tx, actorID, projectID, updateID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if len(created) > 0 {
		s.noteRoadmapChange(projectID)
	}
	return created, nil
}

// applyProposalItemsTx applies the proposal's not-yet-applied add_node items
// inside the caller's transaction. Each item is claimed before its node is
// created (applied_at stamped WHERE applied_at IS NULL), and the claim and the
// node commit or roll back together, so an item is applied once however many
// accepts repeat or race (a racer blocks on the claimed row, then finds it
// stamped and skips it), and a failure part-way leaves nothing applied. The
// caller announces the roadmap change once its transaction has committed.
func applyProposalItemsTx(ctx context.Context, tx pgx.Tx, actorID, projectID, updateID uuid.UUID) ([]uuid.UUID, error) {
	// Authority before the claim: the claim is an RLS-filtered UPDATE, which for
	// someone who may not apply would match no row and read as "already applied".
	if err := requirePlanWriter(ctx, tx, projectID, actorID); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `
		SELECT id, title, parent_hint FROM insideout.proposal_items
		WHERE update_id = $1 AND action = 'add_node' AND applied_at IS NULL
		ORDER BY position, created_at, id`, updateID)
	if err != nil {
		return nil, err
	}
	var todo []ProposalItem
	for rows.Next() {
		var it ProposalItem
		if err := rows.Scan(&it.ID, &it.Title, &it.ParentHint); err != nil {
			rows.Close()
			return nil, err
		}
		todo = append(todo, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	created := make([]uuid.UUID, 0, len(todo))
	for _, it := range todo {
		claim, err := tx.Exec(ctx,
			`UPDATE insideout.proposal_items SET applied_at = now() WHERE id = $1 AND applied_at IS NULL`, it.ID)
		if err != nil {
			return nil, err
		}
		if claim.RowsAffected() == 0 {
			continue // a racing accept stamped it between our read and our claim
		}
		parent, err := resolveParentTx(ctx, tx, projectID, it.ParentHint)
		if err != nil {
			return nil, err
		}
		node, err := createRoadmapNodeTx(ctx, tx, actorID, projectID, parent, it.Title, "proposed: "+it.Title)
		if err != nil {
			return nil, err
		}
		created = append(created, node.ID)
	}
	return created, nil
}

// resolveParentTx finds the node a parentHint names: the first node of the
// project with that title, in the order ListRoadmap lists them. It reads inside
// the apply's transaction so a node created by an earlier item of the same
// apply can be a later item's parent. No match (or no hint) → root.
func resolveParentTx(ctx context.Context, tx pgx.Tx, projectID uuid.UUID, hint string) (*uuid.UUID, error) {
	if hint == "" {
		return nil, nil
	}
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT id FROM insideout.roadmap_nodes
		WHERE project_id = $1 AND title = $2
		ORDER BY parent_id NULLS FIRST, position, created_at, id
		LIMIT 1`, projectID, hint).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}
