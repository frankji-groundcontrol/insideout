package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AgentMessage struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	Role           string
	Content        string
	ToolCalls      json.RawMessage
	ToolCallID     *string
	Tokens         *int
	CreatedAt      time.Time
}

// jsonParam prepares a nullable json.RawMessage for binding to a jsonb
// column under the simple query protocol: nil stays nil (SQL NULL,
// ::jsonb casts that to NULL::jsonb fine), non-nil becomes a plain Go
// string so it's embedded as unescaped text rather than pgx's default
// bytea hex-encoding for []byte — see the comment on insertPrd in
// prds.go for the full reasoning.
func jsonParam(b json.RawMessage) any {
	if b == nil {
		return nil
	}
	return string(b)
}

func (s *Store) InsertAgentMessage(ctx context.Context, actorID, conversationID uuid.UUID, role, content string, toolCalls json.RawMessage, toolCallID *string, tokens *int) (*AgentMessage, error) {
	var m AgentMessage
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			INSERT INTO insideout.agent_messages (conversation_id, role, content, tool_calls, tool_call_id, tokens)
			VALUES ($1, $2, $3, $4::jsonb, $5, $6)
			RETURNING id, conversation_id, role, content, tool_calls, tool_call_id, tokens, created_at`,
			conversationID, role, content, jsonParam(toolCalls), toolCallID, tokens,
		).Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.ToolCalls, &m.ToolCallID, &m.Tokens, &m.CreatedAt)
	})
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// UpdateAgentMessageContent finalizes a placeholder assistant message
// (inserted empty at message_start time so its id can be sent to the
// client immediately) with the streamed reply's full text.
func (s *Store) UpdateAgentMessageContent(ctx context.Context, actorID, id uuid.UUID, content string, tokens *int) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE insideout.agent_messages SET content = $2, tokens = $3 WHERE id = $1`, id, content, tokens)
		return err
	})
}

func (s *Store) ListAgentMessages(ctx context.Context, actorID, conversationID uuid.UUID) ([]AgentMessage, error) {
	var out []AgentMessage
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, conversation_id, role, content, tool_calls, tool_call_id, tokens, created_at
			FROM insideout.agent_messages
			WHERE conversation_id = $1
			ORDER BY created_at ASC`, conversationID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var m AgentMessage
			if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.ToolCalls, &m.ToolCallID, &m.Tokens, &m.CreatedAt); err != nil {
				return err
			}
			out = append(out, m)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// --- ai_runs: one row per user message sent to the agent, and the
// counting source for the rate limiter. ---

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
// heartbeat for staleAfter as failed. Runs with no authenticated actor
// (a background ticker, not a request) — the ai_runs RLS policy treats
// an unset app.user_id as this trusted system context, same pattern as
// the users table (see 20260721160000_ai_runs_reaper_system_context.sql).
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

// --- Rate limiter + circuit breaker: thin wrappers over the SQL
// functions in db/migrations/..._ai_ops.sql. ---

type RateLimitResult struct {
	Allowed           bool
	CurrentCount      int
	MaxRequests       int
	RetryAfterSeconds int
}

func (s *Store) CheckRateLimit(ctx context.Context, userID uuid.UUID) (*RateLimitResult, error) {
	var res RateLimitResult
	err := s.withUserContext(ctx, userID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT allowed, current_count, max_requests, retry_after_seconds FROM insideout.ai_check_rate_limit($1)`, userID).
			Scan(&res.Allowed, &res.CurrentCount, &res.MaxRequests, &res.RetryAfterSeconds)
	})
	if err != nil {
		return nil, err
	}
	return &res, nil
}

type CircuitResult struct {
	Allowed           bool
	State             string
	RetryAfterSeconds int
}

func (s *Store) CheckCircuit(ctx context.Context) (*CircuitResult, error) {
	var res CircuitResult
	err := s.Pool.QueryRow(ctx, `SELECT allowed, state, retry_after_seconds FROM insideout.ai_check_circuit()`).
		Scan(&res.Allowed, &res.State, &res.RetryAfterSeconds)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *Store) RecordCircuitResult(ctx context.Context, success bool) error {
	_, err := s.Pool.Exec(ctx, `SELECT insideout.ai_record_circuit_result($1)`, success)
	return err
}
