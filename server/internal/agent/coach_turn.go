package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/llm"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// turnState is one coach turn, shared by its open, run and close steps
// (HandleMessage drives them in that order).
type turnState struct {
	ctx            context.Context // the request context
	userID         uuid.UUID
	conversationID uuid.UUID
	waited         bool
	conv           *store.AgentConversation
	prd            *store.Prd
	run            *store.AIRun
	assistantMsg   *store.AgentMessage
	sse            *sseWriter
	executor       *toolExecutor

	streamed       strings.Builder // every delta that reached the client
	succeeded      bool
	sectionUpdated bool
	sseErr         error              // first SSE write failure, surfaced after runLoop
	cancelStream   context.CancelFunc // aborts the in-flight provider stream
}

// openTurn loads the conversation and PRD, creates the run and both message
// rows, and opens the SSE stream. On false it has already answered: a plain
// JSON error before the stream opened, or a failed turn after.
func (c *Coach) openTurn(ctx context.Context, w http.ResponseWriter, conversationID, userID, workspaceID uuid.UUID, content string, waited bool) (*turnState, bool) {
	t := &turnState{ctx: ctx, userID: userID, conversationID: conversationID, waited: waited}
	var err error
	t.conv, err = c.store.GetConversationForOwner(ctx, conversationID, userID)
	if err != nil {
		c.log.Error("get conversation", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return nil, false
	}
	t.prd, err = c.store.GetPrdForMember(ctx, t.conv.PrdID, userID)
	if err != nil {
		c.log.Error("get prd", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return nil, false
	}

	t.run, err = c.store.CreateAIRun(ctx, workspaceID, &conversationID, userID, content)
	if err != nil {
		c.log.Error("create ai run", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return nil, false
	}

	if _, err := c.store.InsertAgentMessage(ctx, userID, conversationID, string(llm.RoleUser), content, nil, nil, nil); err != nil {
		c.log.Error("persist user message", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return nil, false
	}
	t.assistantMsg, err = c.store.InsertAgentMessage(ctx, userID, conversationID, string(llm.RoleAssistant), "", nil, nil, nil)
	if err != nil {
		c.log.Error("create placeholder assistant message", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return nil, false
	}

	var ok bool
	t.sse, ok = newSSEWriter(w)
	if !ok {
		c.log.Error("sse: response writer does not support flushing")
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return nil, false
	}
	if err := t.sse.messageStart(t.assistantMsg.ID.String(), c.streamer.Model()); err != nil {
		// Client already gone before the first byte — same handling as a
		// mid-stream disconnect: fail the turn, no circuit result.
		c.log.Error("sse: message_start write failed", "error", err)
		c.failTurn(ctx, userID, t.run.ID, fmt.Errorf("client write failed: %w", context.Canceled), t.sse)
		return nil, false
	}
	return t, true
}

// abandonTurn is the terminal-state guarantee, deferred once the stream is
// open: unless the turn explicitly succeeded, it persists whatever streamed
// and, on a panic, marks the run failed. Cleanup runs on a detached context —
// a client disconnect cancels ctx, and those writes must survive that
// (they're recording the very fact that it happened).
func (c *Coach) abandonTurn(t *turnState) {
	if t.succeeded {
		return
	}
	cleanupCtx, cancel := detachedContext(t.ctx)
	defer cancel()
	partial := t.streamed.String()
	if partial != "" {
		partial += "\n\n[interrupted]"
		logErr(c.log, "UpdateAgentMessageContent", c.store.UpdateAgentMessageContent(cleanupCtx, t.userID, t.assistantMsg.ID, partial, nil), "run_id", t.run.ID)
	}
	if r := recover(); r != nil {
		logErr(c.log, "MarkAIRunFailed", c.store.MarkAIRunFailed(cleanupCtx, t.userID, t.run.ID, fmt.Sprintf("panic: %v", r)), "run_id", t.run.ID)
		logErr(c.log, "RecordCircuitResult", c.store.RecordCircuitResult(cleanupCtx, false), "run_id", t.run.ID)
		c.log.Error("agent loop panic", "panic", r)
		panic(r) // repanic so the recover middleware still logs/500s the request
	}
}

// noteSSEErr keeps the first SSE write failure and aborts the in-flight
// provider stream: once the client is gone there is nothing left to stream to.
func (t *turnState) noteSSEErr(err error) {
	if err != nil && t.sseErr == nil {
		t.sseErr = err
		t.cancelStream()
	}
}

// onDelta streams one text delta to the client and records it as streamed.
func (t *turnState) onDelta(s string) {
	if t.sseErr != nil {
		return
	}
	t.streamed.WriteString(s)
	t.noteSSEErr(t.sse.delta(s))
}

// newExecutor builds the turn's tool executor, wired to the turn's SSE frames.
func (c *Coach) newExecutor(t *turnState) *toolExecutor {
	return &toolExecutor{
		store: c.store, actorID: t.userID, prdID: t.conv.PrdID, conversationID: t.conversationID, currentStage: t.conv.Stage,
		onSectionWrite: func(section string) {
			t.sectionUpdated = true
			t.noteSSEErr(t.sse.prdUpdated(section))
		},
		onStageChange:  t.sse.stageChanged,
		onFactRecorded: func(f Fact) { t.noteSSEErr(t.sse.factRecorded(f)) },
	}
}

// runTurn runs the model loop — with one auto-tighten retry on a
// context-length failure — under the total-turn deadline. On false the turn
// has already been failed.
func (c *Coach) runTurn(t *turnState) (string, bool) {
	ctx := t.ctx
	history, err := c.loadHistory(ctx, t.userID, t.conversationID, historyLimit)
	if err != nil {
		c.log.Error("load history", "error", err)
		c.failTurn(ctx, t.userID, t.run.ID, err, t.sse)
		return "", false
	}
	// No append of `content`: it was stored above, so the history already ends
	// with this turn (the empty assistant placeholder is not replayed).

	// streamCtx lets a failed SSE write abort the in-flight provider
	// stream: once the client is gone there is nothing left to stream to.
	streamCtx, cancelStream := context.WithCancel(ctx)
	defer cancelStream()
	t.cancelStream = cancelStream
	t.executor = c.newExecutor(t)

	// The total-turn deadline spans both runLoop attempts (the auto-tighten
	// retry below included): a trickling provider defeats the idle watchdog
	// and never runs out of iterations, but it cannot outlive this bound —
	// the permit and the conversation lock come back no matter what.
	turnCtx, cancelTurn := context.WithTimeout(streamCtx, c.turnDeadline)
	defer cancelTurn()

	conv, prd := t.conv, t.prd
	finalText, loopErr := c.runLoop(turnCtx, t.userID, t.run.ID, conv.Stage, prd.Title, prd.Sections, history, t.executor, t.onDelta)
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
	if errors.Is(loopErr, llm.ErrContextLength) && t.streamed.Len() == 0 {
		tightHistory, herr := c.loadHistory(ctx, t.userID, t.conversationID, tightenedHistoryLimit)
		if herr == nil {
			finalText, loopErr = c.runLoop(turnCtx, t.userID, t.run.ID, conv.Stage, prd.Title, prd.Sections, tightHistory, t.executor, t.onDelta)
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
	if t.sseErr != nil {
		// An SSE write failed mid-turn — the client is gone. Same handling
		// as a disconnect (failTurn's context.Canceled case): no error
		// event, no circuit result; the defer persists what streamed.
		c.log.Error("sse: client write failed, aborting turn", "error", t.sseErr)
		c.failTurn(ctx, t.userID, t.run.ID, fmt.Errorf("client write failed: %w", context.Canceled), t.sse)
		return "", false
	}
	if loopErr != nil {
		c.failTurn(ctx, t.userID, t.run.ID, loopErr, t.sse)
		return "", false
	}
	return finalText, true
}

// closeTurn persists the success, runs the critic, completes a finished
// conversation and ends the stream.
func (c *Coach) closeTurn(t *turnState, finalText string) {
	ctx, userID, conversationID := t.ctx, t.userID, t.conversationID
	// Persist the success on a detached context so a client disconnect after the
	// last token can't strand the run "running". `succeeded` flips only if the
	// message write lands; otherwise abandonTurn persists what streamed.
	persistCtx, cancel := detachedContext(ctx)
	defer cancel()
	if err := c.store.MarkAIRunSucceeded(persistCtx, userID, t.run.ID, finalText); err != nil {
		c.log.Error("mark ai run succeeded", "error", err)
	}
	logErr(c.log, "RecordCircuitResult", c.store.RecordCircuitResult(persistCtx, true), "run_id", t.run.ID)
	if err := c.store.UpdateAgentMessageContent(persistCtx, userID, t.assistantMsg.ID, finalText, nil); err != nil {
		c.log.Error("persist final assistant message", "error", err)
	} else {
		t.succeeded = true
	}

	// Critic streams further SSE to the client — drop it if they're gone.
	c.maybeRunCritic(ctx, userID, t.conv.PrdID, conversationID, t.run.ID, t.executor.currentStage, finalText, t.sectionUpdated, t.waited, t.sse)

	if t.executor.currentStage == StageFinalize && endsWithMarkerLine(finalText) {
		if err := c.store.CompleteConversation(persistCtx, userID, conversationID); err != nil {
			c.log.Error("complete conversation", "error", err)
		}
	}

	t.sse.messageEnd(t.assistantMsg.ID.String(), len([]rune(finalText)))
}
