package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AIRun is an ai_runs row: one per user message sent to the agent, and the
// counting source for the rate limiter.
type AIRun struct {
	ID             uuid.UUID
	WorkspaceID    uuid.UUID
	ConversationID *uuid.UUID
	UserID         uuid.UUID
	Prompt         string
	Status         string
}

func (s *Store) CreateAIRun(ctx context.Context, workspaceID uuid.UUID, conversationID *uuid.UUID, userID uuid.UUID, prompt string) (*AIRun, error) {
	var run AIRun
	err := s.withUserContext(ctx, userID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			INSERT INTO insideout.ai_runs (workspace_id, conversation_id, user_id, prompt, status)
			VALUES ($1, $2, $3, $4, 'running')
			RETURNING id, workspace_id, conversation_id, user_id, prompt, status`,
			workspaceID, conversationID, userID, prompt,
		).Scan(&run.ID, &run.WorkspaceID, &run.ConversationID, &run.UserID, &run.Prompt, &run.Status)
	})
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (s *Store) MarkAIRunSucceeded(ctx context.Context, actorID, runID uuid.UUID, response string) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE insideout.ai_runs SET status = 'succeeded', response = $2 WHERE id = $1`, runID, response)
		return err
	})
}

// MarkAIRunFailed always transitions to 'failed' — this is the fix for
// the old edge function's bug where a provider 429 left the run stranded
// in 'pending' forever (see docs/plans/2026-07-20-go-rewrite/01-database.md §6).
func (s *Store) MarkAIRunFailed(ctx context.Context, actorID, runID uuid.UUID, errMsg string) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE insideout.ai_runs SET status = 'failed', error_message = $2 WHERE id = $1`, runID, errMsg)
		return err
	})
}

// TouchAIRun updates ai_runs.updated_at to now — a per-provider-call
// heartbeat so the stale-run reaper can tell a long-but-healthy turn
// (repeated heartbeats) from a genuinely crash-abandoned one (none).
func (s *Store) TouchAIRun(ctx context.Context, actorID, runID uuid.UUID) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE insideout.ai_runs SET updated_at = now() WHERE id = $1`, runID)
		return err
	})
}

// ReapStaleAIRuns marks any run stuck in pending/running with no
// heartbeat for staleAfter as failed, then purges rows past
// retentionWindow — without the purge the append-mostly tables grow
// forever (REL-03). Deleting runs older than the window cannot touch
// rate limiting, which counts created_at over the last hour at most.
// Runs with no authenticated actor (a background ticker, not a request)
// — the ai_runs RLS policy treats an unset app.user_id as this trusted
// system context, same pattern as the users table (see
// 20260721160000_ai_runs_reaper_system_context.sql). Conversations use the
// same unset-actor branch (20261020140000). Sessions,
// auth_tokens, invitations and reset_watches sit outside RLS entirely.
// Returns the number of runs reaped.
func (s *Store) ReapStaleAIRuns(ctx context.Context, staleAfter time.Duration) (int, error) {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE insideout.ai_runs
		SET status = 'failed', error_message = 'reaped: stale, no heartbeat'
		WHERE status IN ('pending', 'running') AND updated_at < now() - make_interval(secs => $1)`,
		staleAfter.Seconds(),
	)
	if err != nil {
		return 0, err
	}
	if err := s.purgePastRetention(ctx); err != nil {
		return 0, err
	}
	if err := s.purgeIdleConversations(ctx); err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// InsertAIRunEvent writes to ai_run_events, pure system telemetry with no
// per-end-user access story — deliberately left outside RLS scope (see
// db/migrations/20260720150000_row_level_security.sql), so no actor/
// withUserContext is needed here.
func (s *Store) InsertAIRunEvent(ctx context.Context, runID uuid.UUID, eventType, model string, inputTokens, outputTokens, latencyMs *int, requestMessages, responsePayload json.RawMessage) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO insideout.ai_run_events (run_id, event_type, model, input_tokens, output_tokens, latency_ms, request_messages, response_payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb)`,
		runID, eventType, model, inputTokens, outputTokens, latencyMs, jsonParam(requestMessages), jsonParam(responsePayload),
	)
	return err
}
