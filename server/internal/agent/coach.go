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

// defaultTurnDeadline bounds a whole coach turn in wall-clock time. The
// iteration cap and the idle watchdog bound tool loops and silence, but a
// slow-trickle upstream (a delta every <90s) trips neither — and each turn
// holds one of defaultMaxConcurrent permits plus the conversation lock.
// 12 minutes sits far above any healthy turn (the model finishes, or the
// idle watchdog fires, long before it). Overridable via
// INSIDEOUT_AI_TURN_DEADLINE, same pattern as the knobs above.
const defaultTurnDeadline = 12 * time.Minute

// turnDeadlineFromEnv resolves the total-turn bound from
// INSIDEOUT_AI_TURN_DEADLINE, falling back to defaultTurnDeadline when the
// variable is unset, unparsable, or non-positive.
func turnDeadlineFromEnv() time.Duration {
	if v := os.Getenv("INSIDEOUT_AI_TURN_DEADLINE"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return defaultTurnDeadline
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
	store        *store.Store
	streamer     ChatStreamer
	log          *slog.Logger
	dispatch     *dispatcher
	queueWait    time.Duration // permit queue timeout, surfaced as the 503 retry hint
	turnDeadline time.Duration // total wall-clock bound on one turn (see defaultTurnDeadline)
}

func New(st *store.Store, streamer ChatStreamer, log *slog.Logger) *Coach {
	maxConcurrent, queueWait := aiConcurrencyLimits()
	return &Coach{store: st, streamer: streamer, log: log, dispatch: newDispatcher(maxConcurrent, queueWait), queueWait: queueWait, turnDeadline: turnDeadlineFromEnv()}
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

	if !c.preflight(w, r, userID) {
		return
	}

	unlock, ok := c.dispatch.tryLockConversation(conversationID)
	if !ok {
		httpx.WriteError(w, http.StatusConflict, "a message is already being processed for this conversation", "CONVERSATION_BUSY", nil)
		return
	}
	defer unlock()

	release, waited, ok := c.acquire(w)
	if !ok {
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
	if err := sse.messageStart(assistantMsg.ID.String(), c.streamer.Model()); err != nil {
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
	// No append of `content`: it was stored above, so the history already ends
	// with this turn (the empty assistant placeholder is not replayed).

	sectionUpdated := false
	// streamCtx lets a failed SSE write abort the in-flight provider
	// stream: once the client is gone there is nothing left to stream to.
	streamCtx, cancelStream := context.WithCancel(ctx)
	defer cancelStream()
	var sseErr error // first SSE write failure, surfaced after runLoop

	executor := &toolExecutor{
		store: c.store, actorID: userID, prdID: conv.PrdID, conversationID: conversationID, currentStage: conv.Stage,
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

	// The total-turn deadline spans both runLoop attempts (the auto-tighten
	// retry below included): a trickling provider defeats the idle watchdog
	// and never runs out of iterations, but it cannot outlive this bound —
	// the permit and the conversation lock come back no matter what.
	turnCtx, cancelTurn := context.WithTimeout(streamCtx, c.turnDeadline)
	defer cancelTurn()

	finalText, loopErr := c.runLoop(turnCtx, userID, run.ID, conv.Stage, prd.Title, prd.Sections, history, executor, onDelta)
	// One auto-tighten attempt on a context-length failure: rebuild with a much
	// smaller window, same stage/prd (the ledger rides along inside
	// systemPrompt, rebuilt from the conversation on each provider call). Only
	// retry when NOTHING has reached the client yet
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
			finalText, loopErr = c.runLoop(turnCtx, userID, run.ID, conv.Stage, prd.Title, prd.Sections, tightHistory, executor, onDelta)
		}
	}
	if loopErr != nil && errors.Is(turnCtx.Err(), context.DeadlineExceeded) {
		// The turn outlived its total bound. Classified as our own loop
		// failing (like the iteration cap), not provider health: a turn that
		// merely ran long says nothing the breaker should act on. DeadlineExceeded
		// specifically — a client disconnect cancels turnCtx with Canceled and
		// keeps its routine no-fault handling below.
		loopErr = fmt.Errorf("%w: exceeded the total turn deadline (%s)", ErrLoopInternal, c.turnDeadline)
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

	// Persist the success on a detached context so a client disconnect after the
	// last token can't strand the run "running". `succeeded` flips only if the
	// message write lands; otherwise the defer persists what streamed.
	persistCtx, cancel := detachedContext(ctx)
	defer cancel()
	if err := c.store.MarkAIRunSucceeded(persistCtx, userID, run.ID, finalText); err != nil {
		c.log.Error("mark ai run succeeded", "error", err)
	}
	_ = c.store.RecordCircuitResult(persistCtx, true)
	if err := c.store.UpdateAgentMessageContent(persistCtx, userID, assistantMsg.ID, finalText, nil); err != nil {
		c.log.Error("persist final assistant message", "error", err)
	} else {
		succeeded = true
	}

	// Critic streams further SSE to the client — drop it if they're gone.
	c.maybeRunCritic(ctx, userID, conv.PrdID, conversationID, run.ID, executor.currentStage, finalText, sectionUpdated, waited, sse)

	if executor.currentStage == StageFinalize && endsWithMarkerLine(finalText) {
		if err := c.store.CompleteConversation(persistCtx, userID, conversationID); err != nil {
			c.log.Error("complete conversation", "error", err)
		}
	}

	sse.messageEnd(assistantMsg.ID.String(), len([]rune(finalText)))
}

// finalizeDoneMarker is the exact line a finalize report must END with for
// the conversation to complete (prompts.go: the model writes it only when no
// open questions remain, and lists the questions otherwise).
const finalizeDoneMarker = "NO OPEN QUESTIONS"

// endsWithMarkerLine reports whether text's last non-empty line — the report's
// actual final line, whitespace-trimmed — equals the finalize marker exactly.
// The marker QUOTED anywhere inside the report (e.g. an explanation of what a
// clean report ends with) must not complete a conversation that still lists
// open questions.
func endsWithMarkerLine(text string) bool {
	lines := strings.Split(text, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line == finalizeDoneMarker
		}
	}
	return false
}
