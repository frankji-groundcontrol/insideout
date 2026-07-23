package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
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

// AI_MAX_CONCURRENT / AI_QUEUE_WAIT defaults — see plan §9 open question 2.
const (
	defaultMaxConcurrent = 4
	defaultQueueWait     = 15 * time.Second
)

type Coach struct {
	store    *store.Store
	streamer ChatStreamer
	log      *slog.Logger
	dispatch *dispatcher
}

func New(st *store.Store, streamer ChatStreamer, log *slog.Logger) *Coach {
	return &Coach{store: st, streamer: streamer, log: log, dispatch: newDispatcher(defaultMaxConcurrent, defaultQueueWait)}
}

// HandleMessage implements the api.Coach interface. Rate-limit and
// circuit-breaker checks happen before the SSE stream opens, so throttling
// still returns plain JSON with the preserved 429/503 contract (see
// docs/plans/2026-07-20-go-rewrite/03-agents.md §4). Dispatch (conversation
// lock + concurrency permit) is acquired before any durable write, so a
// rejected request never leaves a stranded run/message row behind.
func (c *Coach) HandleMessage(w http.ResponseWriter, r *http.Request, conversationIDStr, userIDStr, workspaceIDStr, content string) {
	ctx := r.Context()

	conversationID, err1 := uuid.Parse(conversationIDStr)
	userID, err2 := uuid.Parse(userIDStr)
	workspaceID, err3 := uuid.Parse(workspaceIDStr)
	if err1 != nil || err2 != nil || err3 != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	rl, err := c.store.CheckRateLimit(ctx, userID)
	if err != nil {
		c.log.Error("check rate limit", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	if !rl.Allowed {
		httpx.WriteError(w, http.StatusTooManyRequests, "Rate limit exceeded", "APP_THROTTLE", map[string]interface{}{
			"retry_after_seconds": rl.RetryAfterSeconds, "current_count": rl.CurrentCount, "max_requests": rl.MaxRequests,
		})
		return
	}

	circuit, err := c.store.CheckCircuit(ctx)
	if err != nil {
		c.log.Error("check circuit", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	if !circuit.Allowed {
		httpx.WriteError(w, http.StatusServiceUnavailable, "AI service temporarily unavailable", "CIRCUIT_OPEN", map[string]interface{}{
			"retry_after_seconds": circuit.RetryAfterSeconds, "circuit_state": circuit.State,
		})
		return
	}

	unlock, ok := c.dispatch.tryLockConversation(conversationID)
	if !ok {
		httpx.WriteError(w, http.StatusConflict, "a message is already being processed for this conversation", "CONVERSATION_BUSY", nil)
		return
	}
	defer unlock()

	release, waited, ok := c.dispatch.acquirePermit()
	if !ok {
		httpx.WriteError(w, http.StatusServiceUnavailable, "AI service is busy, try again shortly", "CIRCUIT_OPEN", map[string]interface{}{
			"retry_after_seconds": int(defaultQueueWait.Seconds()),
		})
		return
	}
	defer release()

	conv, err := c.store.GetConversationForOwner(ctx, conversationID, userID)
	if err != nil {
		c.log.Error("get conversation", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	prd, err := c.store.GetPrdForMember(ctx, conv.PrdID, userID)
	if err != nil {
		c.log.Error("get prd", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	run, err := c.store.CreateAIRun(ctx, workspaceID, &conversationID, userID, content)
	if err != nil {
		c.log.Error("create ai run", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	if _, err := c.store.InsertAgentMessage(ctx, userID, conversationID, string(RoleUser), content, nil, nil, nil); err != nil {
		c.log.Error("persist user message", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	assistantMsg, err := c.store.InsertAgentMessage(ctx, userID, conversationID, string(RoleAssistant), "", nil, nil, nil)
	if err != nil {
		c.log.Error("create placeholder assistant message", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	sse, ok := newSSEWriter(w)
	if !ok {
		c.log.Error("sse: response writer does not support flushing")
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	sse.messageStart(assistantMsg.ID.String())

	// Terminal-state guarantee: unless the turn explicitly succeeds below,
	// this defer marks it failed (panic included) and persists whatever
	// streamed. Cleanup runs on a detached context — a client disconnect
	// cancels ctx, and those writes must survive that (they're recording
	// the very fact that it happened).
	var streamed strings.Builder
	succeeded := false
	defer func() {
		if succeeded {
			return
		}
		cleanupCtx := context.WithoutCancel(ctx)
		partial := streamed.String()
		if partial != "" {
			partial += "\n\n[interrupted]"
			_ = c.store.UpdateAgentMessageContent(cleanupCtx, userID, assistantMsg.ID, partial, nil)
		}
		if r := recover(); r != nil {
			_ = c.store.MarkAIRunFailed(cleanupCtx, userID, run.ID, fmt.Sprintf("panic: %v", r))
			_ = c.store.RecordCircuitResult(cleanupCtx, false)
			c.log.Error("agent loop panic", "panic", r)
			panic(r) // repanic so the recover middleware still logs/500s the request
		}
	}()

	history, err := c.loadHistory(ctx, userID, conversationID, historyLimit)
	if err != nil {
		c.log.Error("load history", "error", err)
		c.failTurn(ctx, userID, run.ID, err, sse)
		return
	}
	history = append(history, Message{Role: RoleUser, Content: content})

	sectionUpdated := false
	executor := &toolExecutor{
		store: c.store, actorID: userID, prdID: conv.PrdID, conversationID: conversationID, currentStage: conv.Stage,
		userMessages: userTexts(history),
		onSectionWrite: func(section string) {
			sectionUpdated = true
			sse.prdUpdated(section)
		},
		onStageChange:  sse.stageChanged,
		onFactRecorded: sse.factRecorded,
	}

	onDelta := func(s string) {
		streamed.WriteString(s)
		sse.delta(s)
	}

	finalText, loopErr := c.runLoop(ctx, userID, run.ID, conv.Stage, prd.Title, prd.Sections, history, executor, onDelta)
	if errors.Is(loopErr, ErrContextLength) {
		// One auto-tighten attempt: rebuild with a much smaller window,
		// same stage/prd (the ledger, once H2 lands, rides along inside
		// systemPrompt). Reset what streamed so far — it belongs to the
		// failed attempt, not this one.
		streamed.Reset()
		tightHistory, herr := c.loadHistory(ctx, userID, conversationID, tightenedHistoryLimit)
		if herr == nil {
			tightHistory = append(tightHistory, Message{Role: RoleUser, Content: content})
			finalText, loopErr = c.runLoop(ctx, userID, run.ID, conv.Stage, prd.Title, prd.Sections, tightHistory, executor, onDelta)
		}
	}
	if loopErr != nil {
		c.failTurn(ctx, userID, run.ID, loopErr, sse)
		return
	}

	if err := c.store.MarkAIRunSucceeded(ctx, userID, run.ID, finalText); err != nil {
		c.log.Error("mark ai run succeeded", "error", err)
	}
	_ = c.store.RecordCircuitResult(ctx, true)
	_ = c.store.UpdateAgentMessageContent(ctx, userID, assistantMsg.ID, finalText, nil)
	succeeded = true

	c.maybeRunCritic(ctx, userID, conv.PrdID, conversationID, run.ID, executor.currentStage, sectionUpdated, waited, sse)

	if executor.currentStage == StageFinalize && strings.Contains(finalText, "NO OPEN QUESTIONS") {
		if err := c.store.CompleteConversation(ctx, userID, conversationID); err != nil {
			c.log.Error("complete conversation", "error", err)
		}
	}

	sse.messageEnd(assistantMsg.ID.String(), len([]rune(finalText)))
}

// failTurn classifies loopErr per the taxonomy (§5.2) and reports it —
// SSE error code, whether it counts against the circuit breaker — then
// persists the terminal ai_runs state. Cleanup runs on a context
// detached from the (possibly already-canceled) request context, same
// reasoning as the HandleMessage defer.
func (c *Coach) failTurn(ctx context.Context, userID uuid.UUID, runID uuid.UUID, loopErr error, sse *sseWriter) {
	cleanupCtx := context.WithoutCancel(ctx)
	_ = c.store.MarkAIRunFailed(cleanupCtx, userID, runID, loopErr.Error())

	switch {
	case errors.Is(loopErr, ErrProviderRateLimited):
		_ = c.store.RecordCircuitResult(cleanupCtx, false)
		sse.errorEvent("AI provider is rate limited", "ANTHROPIC_RATE_LIMIT")
	case errors.Is(loopErr, ErrProviderTransient):
		_ = c.store.RecordCircuitResult(cleanupCtx, false)
		c.log.Error("agent loop: provider transient error", "error", loopErr)
		sse.errorEvent("internal error", "")
	case errors.Is(loopErr, ErrContextLength):
		sse.errorEvent("conversation too long — snapshot and start a fresh session", "CONTEXT_LENGTH")
	case errors.Is(loopErr, ErrProviderConfig):
		c.log.Error("agent loop: provider configuration error", "error", loopErr)
		sse.errorEvent(loopErr.Error(), "PROVIDER_CONFIG")
	case errors.Is(loopErr, ErrContentRefusal):
		sse.errorEvent("the model declined to respond", "CONTENT_REFUSAL")
	case errors.Is(loopErr, context.Canceled):
		// Client disconnected. No circuit result — routine, not provider health.
	default:
		_ = c.store.RecordCircuitResult(cleanupCtx, false)
		c.log.Error("agent loop", "error", loopErr)
		sse.errorEvent("internal error", "")
	}
}

// runLoop drives the model until it produces final text (as opposed to a
// tool call), bounded by maxToolIterations. It heartbeats the ai_run
// before each provider call so the stale-run reaper can distinguish a
// long-but-healthy turn from a crash-abandoned one.
func (c *Coach) runLoop(ctx context.Context, actorID, runID uuid.UUID, stage, prdTitle string, sections map[string]string, history []Message, executor *toolExecutor, onDelta func(string)) (string, error) {
	msgs := history
	for i := 0; i < maxToolIterations; i++ {
		_ = c.store.TouchAIRun(ctx, actorID, runID)

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
