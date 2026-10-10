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
		var scanErr error
		c, scanErr = latestConversationForPrd(ctx, tx, prdID, userID)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

// latestConversationForPrd must be called with an open transaction.
func latestConversationForPrd(ctx context.Context, tx pgx.Tx, prdID, userID uuid.UUID) (*AgentConversation, error) {
	return scanAgentConversation(tx.QueryRow(ctx, `
		SELECT `+agentConversationColumns+`
		FROM insideout.agent_conversations
		WHERE prd_id = $1 AND user_id = $2
		ORDER BY created_at DESC
		LIMIT 1`,
		prdID, userID,
	))
}

// EnsureConversationForPrd returns actorID's own coach conversation on the PRD,
// starting one if they have none; created reports a new row. Conversations are
// owner-only and ConvertIdea makes just the converter's, so every other author
// or admin needs this to coach the PRD. The gate is the one UpdateSections
// applies to the sections the coach edits. The per-(PRD, user) advisory lock
// closes the check-then-insert window: the table has no unique key on it, so
// concurrent starts would each add a row.
func (s *Store) EnsureConversationForPrd(ctx context.Context, actorID, prdID uuid.UUID) (*AgentConversation, bool, error) {
	var conv *AgentConversation
	created := false
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		prd, err := requirePrdAuthorOrAdmin(ctx, tx, prdID, actorID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "prd_conversation:"+prdID.String()+":"+actorID.String()); err != nil {
			return err
		}
		conv, err = latestConversationForPrd(ctx, tx, prdID, actorID)
		if !errors.Is(err, ErrNotFound) {
			return err
		}
		conv, err = insertAgentConversation(ctx, tx, prd.WorkspaceID, actorID, prdID)
		created = err == nil
		return err
	})
	if err != nil {
		return nil, false, err
	}
	return conv, created, nil
}

func (s *Store) UpdateConversationStage(ctx context.Context, actorID, conversationID uuid.UUID, stage string) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE insideout.agent_conversations SET stage = $2 WHERE id = $1`, conversationID, stage)
		return err
	})
}

// ConfirmConversationStage writes a stage the coach already proposed. The
// proposal lives in meta, not in stage, so declining leaves the row as it
// was. A missing or different proposal is ErrNotFound.
func (s *Store) ConfirmConversationStage(ctx context.Context, actorID, conversationID uuid.UUID, stage string) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE insideout.agent_conversations
			SET stage = $3, meta = meta - 'pendingStage'
			WHERE id = $1 AND user_id = $2 AND meta->>'pendingStage' = $3`,
			conversationID, actorID, stage,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
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
