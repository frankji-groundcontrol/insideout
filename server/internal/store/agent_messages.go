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

// ListRecentChatMessages returns the newest limit user turns and assistant
// text replies of a conversation, oldest first: what the coach replays as
// model context. Tool results and assistant rows that only carry a tool call
// (empty content) are left out. The filter and the LIMIT run in SQL so a long
// conversation is never loaded whole just to keep its tail.
func (s *Store) ListRecentChatMessages(ctx context.Context, actorID, conversationID uuid.UUID, limit int) ([]AgentMessage, error) {
	var out []AgentMessage
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, conversation_id, role, content, tool_calls, tool_call_id, tokens, created_at
			FROM (
				SELECT id, conversation_id, role, content, tool_calls, tool_call_id, tokens, created_at
				FROM insideout.agent_messages
				WHERE conversation_id = $1
				  AND (role = 'user' OR (role = 'assistant' AND content <> ''))
				ORDER BY created_at DESC
				LIMIT $2
			) recent
			ORDER BY created_at ASC`, conversationID, limit,
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

// ListUserMessageTexts returns what the user typed in a conversation: every
// turn, oldest first. It is what record_fact grounds a quote against, so it is
// deliberately not the replay window of ListRecentChatMessages, which drops old
// turns to keep the model's context small.
func (s *Store) ListUserMessageTexts(ctx context.Context, actorID, conversationID uuid.UUID) ([]string, error) {
	var out []string
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT content
			FROM insideout.agent_messages
			WHERE conversation_id = $1 AND role = 'user'
			ORDER BY created_at ASC`, conversationID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var text string
			if err := rows.Scan(&text); err != nil {
				return err
			}
			out = append(out, text)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
