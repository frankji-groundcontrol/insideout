package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AgentConversation struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	PrdID       uuid.UUID
	Stage       string
	Status      string
	Meta        json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const agentConversationColumns = `id, workspace_id, user_id, prd_id, stage, status, meta, created_at, updated_at`

func scanAgentConversation(row pgx.Row) (*AgentConversation, error) {
	var c AgentConversation
	err := row.Scan(&c.ID, &c.WorkspaceID, &c.UserID, &c.PrdID, &c.Stage, &c.Status, &c.Meta, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &c, err
}

// insertAgentConversation must be called with an open transaction — used
// by ConvertIdea (ideas.go) when an idea becomes a PRD.
func insertAgentConversation(ctx context.Context, tx pgx.Tx, workspaceID, userID, prdID uuid.UUID) (*AgentConversation, error) {
	row := tx.QueryRow(ctx, `
		INSERT INTO insideout.agent_conversations (workspace_id, user_id, prd_id)
		VALUES ($1, $2, $3)
		RETURNING `+agentConversationColumns,
		workspaceID, userID, prdID,
	)
	return scanAgentConversation(row)
}

// GetConversationForOwner returns the conversation only if it belongs to
// userID — conversations are strictly owner-only (read and write), per
// docs/plans/2026-07-20-go-rewrite/01-database.md §5. userID doubles as
// the RLS actor (self-referential, same pattern as GetMembership).
func (s *Store) GetConversationForOwner(ctx context.Context, conversationID, userID uuid.UUID) (*AgentConversation, error) {
	var c *AgentConversation
	err := s.withUserContext(ctx, userID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT `+agentConversationColumns+`
			FROM insideout.agent_conversations
			WHERE id = $1 AND user_id = $2`,
			conversationID, userID,
		)
		var scanErr error
		c, scanErr = scanAgentConversation(row)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

// GetLatestConversationForPrd finds the most recent conversation for a
// PRD, owned by userID — lets the PRD workspace page resume coaching on
// revisit instead of only right after idea conversion.
func (s *Store) GetLatestConversationForPrd(ctx context.Context, prdID, userID uuid.UUID) (*AgentConversation, error) {
	var c *AgentConversation
	err := s.withUserContext(ctx, userID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT `+agentConversationColumns+`
			FROM insideout.agent_conversations
			WHERE prd_id = $1 AND user_id = $2
			ORDER BY created_at DESC
			LIMIT 1`,
			prdID, userID,
		)
		var scanErr error
		c, scanErr = scanAgentConversation(row)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Store) UpdateConversationStage(ctx context.Context, actorID, conversationID uuid.UUID, stage string) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE insideout.agent_conversations SET stage = $2 WHERE id = $1`, conversationID, stage)
		return err
	})
}

func (s *Store) UpdateConversationMeta(ctx context.Context, actorID, conversationID uuid.UUID, meta json.RawMessage) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE insideout.agent_conversations SET meta = $2::jsonb WHERE id = $1`, conversationID, jsonParam(meta))
		return err
	})
}

func (s *Store) CompleteConversation(ctx context.Context, actorID, conversationID uuid.UUID) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE insideout.agent_conversations SET status = 'completed' WHERE id = $1`, conversationID)
		return err
	})
}
