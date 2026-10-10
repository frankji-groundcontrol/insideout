package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// maxToolIterations bounds the in-request tool-call loop so a
// misbehaving model can't hang a request forever. Raised from 6 (H2's
// per-fact record_fact calls plus an 8-section draft burst can exceed 6
// tool calls in one turn).
const maxToolIterations = 12

// historyLimit is how many past user/assistant text turns are replayed
// as context on each new message. Tool-call bookkeeping from past turns
// is deliberately excluded — the model only needs to know what it
// concluded, not replay its own past tool mechanics, which keeps context
// compact. Full detail still lives in agent_messages for audit/debugging.
// `// ponytail: naive last-N; add token-budgeted packing or summarization if sessions blow the context window`
const historyLimit = 20

// tightenedHistoryLimit is the fallback history window on a context-length
// auto-tighten retry (§5.2) — one attempt, then surface honestly.
const tightenedHistoryLimit = 6

// truncatedMarker is appended to a final answer the provider cut off at its
// output-token cap (stop_reason=max_tokens, surfaced via Turn.Truncated). The
// partial answer is still persisted, but the marker keeps it from being read —
// and replayed into future context — as a finished response (F13).
const truncatedMarker = "\n\n[response truncated — hit the token limit; ask me to continue]"

// runLoop drives the model until it produces final text (as opposed to a
// tool call), bounded by maxToolIterations. It heartbeats the ai_run
// before each provider call so the stale-run reaper can distinguish a
// long-but-healthy turn from a crash-abandoned one.
func (c *Coach) runLoop(ctx context.Context, actorID, runID uuid.UUID, stage, prdTitle string, sections map[string]string, history []Message, executor *toolExecutor, onDelta func(string)) (string, error) {
	msgs := history
	for i := 0; i < maxToolIterations; i++ {
		_ = c.store.TouchAIRun(ctx, actorID, runID)

		// The previous iteration may have rewritten sections, so prompt from
		// the PRD as it is now, not the pre-turn snapshot (this also freshens
		// the auto-tighten retry). A failed read keeps the last snapshot: a
		// slightly stale prompt beats failing the turn.
		if prd, err := c.store.GetPrdForMember(ctx, executor.prdID, executor.actorID); err != nil {
			c.log.Error("refresh prd sections for the prompt", "error", err)
		} else {
			sections = prd.Sections
		}

		conv, err := c.store.GetConversationForOwner(ctx, executor.conversationID, actorID)
		if err != nil {
			return "", err
		}
		lm, extra := loadLedger(conv.Meta)
		findingsText := formatFindingsForPrompt(loadCriticFindings(extra))
		started := time.Now()
		turn, err := c.streamer.StreamChat(ctx, systemPrompt(stage, prdTitle, sections, formatLedgerForPrompt(lm), findingsText), msgs, coachTools(), onDelta)
		c.recordTelemetry(ctx, runID, "coach", started, turn.Usage, err)
		if err != nil {
			return "", err
		}
		if turn.ToolCall == nil {
			if turn.Truncated {
				return turn.Text + truncatedMarker, nil
			}
			return turn.Text, nil
		}

		toolCallsJSON, _ := json.Marshal([]map[string]any{{
			"id": turn.ToolCall.ID, "name": turn.ToolCall.Name, "arguments": turn.ToolCall.Arguments,
		}})
		if _, err := c.store.InsertAgentMessage(ctx, executor.actorID, executor.conversationID, string(RoleAssistant), "", toolCallsJSON, nil, nil); err != nil {
			c.log.Error("persist tool-call message", "error", err)
		}

		result, execErr := executor.Execute(ctx, *turn.ToolCall)
		if execErr != nil {
			result = fmt.Sprintf("error: %s", execErr.Error())
		}
		if executor.currentStage != stage {
			// advance_stage just changed the stage mid-turn — rebuild the
			// system prompt before the next provider call so behavior
			// actually switches this turn, not next (§10.9).
			stage = executor.currentStage
		}

		toolCallID := turn.ToolCall.ID
		if _, err := c.store.InsertAgentMessage(ctx, executor.actorID, executor.conversationID, string(RoleTool), result, nil, &toolCallID, nil); err != nil {
			c.log.Error("persist tool-result message", "error", err)
		}

		msgs = append(msgs,
			Message{Role: RoleAssistant, ToolCall: turn.ToolCall},
			Message{Role: RoleTool, Content: result, ToolCallID: turn.ToolCall.ID},
		)
	}
	return "", fmt.Errorf("agent: exceeded max tool iterations (%d)", maxToolIterations)
}

func (c *Coach) loadHistory(ctx context.Context, userID, conversationID uuid.UUID, limit int) ([]Message, error) {
	all, err := c.store.ListAgentMessages(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}

	var out []Message
	for _, m := range all {
		if m.Role != string(RoleUser) && m.Role != string(RoleAssistant) {
			continue
		}
		if m.Role == string(RoleAssistant) && m.Content == "" {
			continue // a tool-call-only turn, not a final text reply / 仅工具调用轮次，非最终文本回复
		}
		out = append(out, Message{Role: Role(m.Role), Content: m.Content})
	}
	if len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

// userTexts extracts just the user's own turns — record_fact's quote
// grounding check only trusts what the user actually typed, not the
// assistant's own words.
func userTexts(history []Message) []string {
	var out []string
	for _, m := range history {
		if m.Role == RoleUser {
			out = append(out, m.Content)
		}
	}
	return out
}
