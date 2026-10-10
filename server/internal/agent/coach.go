package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// Defaults for the AI concurrency limits, overridable via
// INSIDEOUT_AI_MAX_CONCURRENT and INSIDEOUT_AI_QUEUE_WAIT (see
// aiConcurrencyLimits) — see plan §9 open question 2.
const (
	defaultMaxConcurrent = 4
	defaultQueueWait     = 15 * time.Second
)

// aiConcurrencyLimits resolves the permit-pool size and queue timeout from
// INSIDEOUT_AI_MAX_CONCURRENT and INSIDEOUT_AI_QUEUE_WAIT, falling back to
// the defaults above when a variable is unset or doesn't parse.
func aiConcurrencyLimits() (int, time.Duration) {
	maxConcurrent := defaultMaxConcurrent
	if v, err := strconv.Atoi(os.Getenv("INSIDEOUT_AI_MAX_CONCURRENT")); err == nil && v > 0 {
		maxConcurrent = v
	}
	queueWait := defaultQueueWait
	if v := os.Getenv("INSIDEOUT_AI_QUEUE_WAIT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			queueWait = d
		}
	}
	return maxConcurrent, queueWait
}

// persistTimeout bounds the detached terminal writes (cleanup/persist) that
// must survive a client disconnect but must not outlive the process. Without a
// bound, a wedged DB write on a WithoutCancel context hangs its goroutine — and
// the dispatch permit it holds — forever. 10s is generous for a few small
// writes; raise it if the store ever does something slow on this path.
const persistTimeout = 10 * time.Second

// detachedContext returns a context detached from the (possibly already
// canceled) request ctx — so terminal persistence survives a client disconnect
// — but bounded by persistTimeout so a wedged write can't hang forever. Use for
// the cleanup/persist writes that must outlive the request, never for the
// request's own work.
func detachedContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), persistTimeout)
}

type Coach struct {
	store     *store.Store
	streamer  ChatStreamer
	log       *slog.Logger
	dispatch  *dispatcher
	queueWait time.Duration // permit queue timeout, surfaced as the 503 retry hint
}

func New(st *store.Store, streamer ChatStreamer, log *slog.Logger) *Coach {
	maxConcurrent, queueWait := aiConcurrencyLimits()
	return &Coach{store: st, streamer: streamer, log: log, dispatch: newDispatcher(maxConcurrent, queueWait), queueWait: queueWait}
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
			"retry_after_seconds": int(c.queueWait.Seconds()),
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
	if err := sse.messageStart(assistantMsg.ID.String()); err != nil {
		// Client already gone before the first byte — same handling as a
		// mid-stream disconnect: fail the turn, no circuit result.
		c.log.Error("sse: message_start write failed", "error", err)
		c.failTurn(ctx, userID, run.ID, fmt.Errorf("client write failed: %w", context.Canceled), sse)
		return
	}

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
		cleanupCtx, cancel := detachedContext(ctx)
		defer cancel()
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
	// streamCtx lets a failed SSE write abort the in-flight provider
	// stream: once the client is gone there is nothing left to stream to.
	streamCtx, cancelStream := context.WithCancel(ctx)
	defer cancelStream()
	var sseErr error // first SSE write failure, surfaced after runLoop

	executor := &toolExecutor{
		store: c.store, actorID: userID, prdID: conv.PrdID, conversationID: conversationID, currentStage: conv.Stage,
		userMessages: userTexts(history),
		onSectionWrite: func(section string) {
			sectionUpdated = true
			if err := sse.prdUpdated(section); err != nil && sseErr == nil {
				sseErr = err
				cancelStream()
			}
		},
		onStageChange: sse.stageChanged,
		onFactRecorded: func(f Fact) {
			if err := sse.factRecorded(f); err != nil && sseErr == nil {
				sseErr = err
				cancelStream()
			}
		},
	}

	onDelta := func(s string) {
		if sseErr != nil {
			return
		}
		streamed.WriteString(s)
		if err := sse.delta(s); err != nil {
			sseErr = err
			cancelStream()
		}
	}

	finalText, loopErr := c.runLoop(streamCtx, userID, run.ID, conv.Stage, prd.Title, prd.Sections, history, executor, onDelta)
	// One auto-tighten attempt on a context-length failure: rebuild with a much
	// smaller window, same stage/prd (the ledger, once H2 lands, rides along
	// inside systemPrompt). Only retry when NOTHING has reached the client yet
	// (streamed.Len()==0 — onDelta writes to `streamed` and flushes the SSE
	// delta in lockstep, so this is exactly "the bubble is still empty").
	// If deltas already flushed, retrying would append a second, different
	// answer onto the live bubble while `streamed` was reset to hold only the
	// retry's text — the displayed answer and the persisted one would diverge.
	// In that case we fall through to failTurn instead, so what the user saw is
	// exactly what gets persisted (F16). A context-length error almost always
	// arrives as a pre-stream 400 (empty bubble), so the retry still fires for
	// the common case.
	if errors.Is(loopErr, ErrContextLength) && streamed.Len() == 0 {
		tightHistory, herr := c.loadHistory(ctx, userID, conversationID, tightenedHistoryLimit)
		if herr == nil {
			tightHistory = append(tightHistory, Message{Role: RoleUser, Content: content})
			finalText, loopErr = c.runLoop(streamCtx, userID, run.ID, conv.Stage, prd.Title, prd.Sections, tightHistory, executor, onDelta)
		}
	}
	if sseErr != nil {
		// An SSE write failed mid-turn — the client is gone. Same handling
		// as a disconnect (failTurn's context.Canceled case): no error
		// event, no circuit result; the defer persists what streamed.
		c.log.Error("sse: client write failed, aborting turn", "error", sseErr)
		c.failTurn(ctx, userID, run.ID, fmt.Errorf("client write failed: %w", context.Canceled), sse)
		return
	}
	if loopErr != nil {
		c.failTurn(ctx, userID, run.ID, loopErr, sse)
		return
	}

	// The stream finished; persist the success on a detached context. If the
	// client disconnects right after the last token, ctx is cancelled and
	// these writes would fail — leaving the run stuck "running" with an empty
	// message while `succeeded` still flipped true so the failure defer skips.
	// Mirror the defer's cleanup_ctx treatment for the success path.
	persistCtx, cancel := detachedContext(ctx)
	defer cancel()
	if err := c.store.MarkAIRunSucceeded(persistCtx, userID, run.ID, finalText); err != nil {
		c.log.Error("mark ai run succeeded", "error", err)
	}
	_ = c.store.RecordCircuitResult(persistCtx, true)
	_ = c.store.UpdateAgentMessageContent(persistCtx, userID, assistantMsg.ID, finalText, nil)
	succeeded = true

	// Critic streams further SSE to the client — drop it if they're gone.
	c.maybeRunCritic(ctx, userID, conv.PrdID, conversationID, run.ID, executor.currentStage, finalText, sectionUpdated, waited, sse)

	if executor.currentStage == StageFinalize && strings.Contains(finalText, "NO OPEN QUESTIONS") {
		if err := c.store.CompleteConversation(persistCtx, userID, conversationID); err != nil {
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
	cleanupCtx, cancel := detachedContext(ctx)
	defer cancel()
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
	case errors.Is(loopErr, ErrUpstreamStall):
		// The provider went silent and the idle watchdog killed the stream
		// while the client was still connected — a hung upstream, not a
		// disconnect. Counts against the circuit breaker, and the still-there
		// user gets an honest error instead of a silently dead bubble.
		_ = c.store.RecordCircuitResult(cleanupCtx, false)
		c.log.Error("agent loop: upstream stream stalled (idle watchdog)", "error", loopErr)
		sse.errorEvent("AI provider stopped responding — try again", "UPSTREAM_STALL")
	case errors.Is(loopErr, context.Canceled):
		// Client disconnected. No circuit result — routine, not provider health.
	default:
		_ = c.store.RecordCircuitResult(cleanupCtx, false)
		c.log.Error("agent loop", "error", loopErr)
		sse.errorEvent("internal error", "")
	}
}
