// Package agent implements the PRD Coach: a four-stage coaching
// conversation that guides a user from a raw idea to a structured PRD,
// writing sections via tool calls and streaming replies over SSE. See
// docs/plans/2026-07-20-go-rewrite/03-agents.md.
package agent

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/llm"
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
	streamer     llm.ChatStreamer
	log          *slog.Logger
	dispatch     *dispatcher
	queueWait    time.Duration // permit queue timeout, surfaced as the 503 retry hint
	turnDeadline time.Duration // total wall-clock bound on one turn (see defaultTurnDeadline)
}

func New(st *store.Store, streamer llm.ChatStreamer, log *slog.Logger) *Coach {
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
		httpx.WriteError(w, http.StatusConflict, "a message is already being processed for this conversation", "conversation_busy", nil)
		return
	}
	defer unlock()

	release, waited, ok := c.acquire(w)
	if !ok {
		return
	}
	defer release()

	t, ok := c.openTurn(ctx, w, conversationID, userID, workspaceID, content, waited)
	if !ok {
		return
	}
	defer c.abandonTurn(t)
	finalText, ok := c.runTurn(t)
	if !ok {
		return
	}
	c.closeTurn(t, finalText)
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
