package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// Review-4 QUAL-06. The writes that end a turn (mark the run failed or
// succeeded, record the breaker result, keep the text that streamed, heartbeat
// the run) are best effort: they cannot change an answer the client already has.
// A lost one must still leave a trace. A run whose terminal write is lost stays
// "running" until the reaper, and a breaker that misses failures never opens, so
// the operator has to be told which write was lost and why.
//
// Each test loses the database for real at the point the bookkeeping runs (the
// store's pool is closed, so every write fails with the pool's own error), then
// reads the coach's log. Nothing is mocked: the store and the database are real.

// bookkeepingOps are the store writes that make up a turn's terminal state; the
// log names a lost one by its store method.
var bookkeepingOps = []string{
	"MarkAIRunFailed", "MarkAIRunSucceeded", "RecordCircuitResult", "TouchAIRun", "UpdateAgentMessageContent",
}

func capturingLog() (*bytes.Buffer, *slog.Logger) {
	var logs bytes.Buffer
	return &logs, slog.New(slog.NewTextHandler(&logs, nil))
}

// lostDBError closes st's pool, so that every later write fails for real, and
// returns the text of the error such a write fails with.
func lostDBError(t *testing.T, st *store.Store) string {
	t.Helper()
	st.Close() // idempotent: the test's cleanup closes it again
	err := st.RecordCircuitResult(context.Background(), true)
	if err == nil {
		t.Fatal("a write on a closed pool succeeded")
	}
	return err.Error()
}

// wantLost fails for every op that has no warning-or-error line in logs naming
// it together with the error it failed with.
func wantLost(t *testing.T, logs *bytes.Buffer, cause string, ops ...string) {
	t.Helper()
	for _, op := range ops {
		found := false
		for _, line := range strings.Split(logs.String(), "\n") {
			leveled := strings.Contains(line, "level=WARN") || strings.Contains(line, "level=ERROR")
			if leveled && strings.Contains(line, op) && strings.Contains(line, cause) {
				found = true
			}
		}
		if !found {
			t.Errorf("the lost %s write left no warning naming it with %q; log:\n%s", op, cause, logs)
		}
	}
}

// drivenTurn is what one coach turn left behind.
type drivenTurn struct {
	logs     *bytes.Buffer
	body     string // the SSE stream the client saw
	panicked any    // what HandleMessage panicked with, if it did
}

// driveTurn runs one coach turn on a real store whose provider is chat. chat is
// handed the store, so that it can lose the database while it "answers".
func driveTurn(t *testing.T, chat func(st *store.Store, onDelta func(string)) (Turn, error)) drivenTurn {
	t.Helper()
	st := newTestStore(t)
	userID, ex := seedConversation(t, st)
	conv, err := st.GetConversationForOwner(context.Background(), ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}
	logs, log := capturingLog()
	c := New(st, fakeStreamer{chat: func(_ string, onDelta func(string)) (Turn, error) { return chat(st, onDelta) }}, log)

	out := drivenTurn{logs: logs}
	rec := httptest.NewRecorder()
	func() {
		defer func() { out.panicked = recover() }()
		c.HandleMessage(rec, httptest.NewRequest(http.MethodPost, "/", nil),
			ex.conversationID.String(), userID, conv.WorkspaceID.String(), "hi")
	}()
	out.body = rec.Body.String()
	return out
}

// plannerCall admits a call that is not a coach turn (the roadmap planner) on a
// real store, and returns the store, the coach's log and the done func the call
// ends with.
func plannerCall(t *testing.T) (*store.Store, *bytes.Buffer, func(error)) {
	t.Helper()
	st := newTestStore(t)
	logs, log := capturingLog()
	c := New(st, nil, log)
	userID, workspaceID := admitFixture(t, c)
	_, done, ok := admit(c, userID, workspaceID)
	if !ok {
		t.Fatal("the planner call was refused")
	}
	return st, logs, done
}

func TestTerminalStateWritesThatFailAreLogged(t *testing.T) {
	t.Run("failTurn", func(t *testing.T) {
		st := newTestStore(t)
		_, ex := seedConversation(t, st)
		run := startRun(t, st, ex)
		logs, log := capturingLog()
		c := New(st, fakeStreamer{}, log)
		cause := lostDBError(t, st)

		sse, _ := newSSEWriter(httptest.NewRecorder())
		c.failTurn(context.Background(), ex.actorID, run.ID, errors.New("agent: anthropic returned 422: bad field"), sse)

		wantLost(t, logs, cause, "MarkAIRunFailed", "RecordCircuitResult")
	})

	t.Run("a planner call that ends well", func(t *testing.T) {
		st, logs, done := plannerCall(t)
		cause := lostDBError(t, st)

		done(nil)

		wantLost(t, logs, cause, "MarkAIRunSucceeded", "RecordCircuitResult")
	})

	t.Run("a planner call that fails on the provider", func(t *testing.T) {
		st, logs, done := plannerCall(t)
		cause := lostDBError(t, st)

		done(fmt.Errorf("%w: anthropic returned 503", ErrProviderTransient))

		wantLost(t, logs, cause, "MarkAIRunFailed", "RecordCircuitResult")
	})

	t.Run("the heartbeat of a run", func(t *testing.T) {
		st := newTestStore(t)
		_, ex := seedConversation(t, st)
		ex.currentStage = StageDraft
		run := startRun(t, st, ex)
		prd, err := st.GetPrdForMember(context.Background(), ex.prdID, ex.actorID)
		if err != nil {
			t.Fatalf("get prd: %v", err)
		}
		logs, log := capturingLog()
		c := New(st, fakeStreamer{chat: func(string, func(string)) (Turn, error) {
			t.Fatal("the provider must not be called when the loop cannot read its conversation")
			return Turn{}, nil
		}}, log)
		cause := lostDBError(t, st)

		// Losing the heartbeat changes nothing about the loop's own result.
		_, err = c.runLoop(context.Background(), ex.actorID, run.ID, StageDraft, prd.Title, prd.Sections, nil, ex, nil)
		if !errors.Is(err, ErrLoopInternal) {
			t.Fatalf("runLoop on a lost database = %v, want an ErrLoopInternal failure", err)
		}

		wantLost(t, logs, cause, "TouchAIRun")
	})

	t.Run("a turn that ends well", func(t *testing.T) {
		var lost *store.Store
		out := driveTurn(t, func(st *store.Store, onDelta func(string)) (Turn, error) {
			lost = st
			onDelta("an answer")
			st.Close() // the database goes away just as the provider finishes
			return Turn{Text: "an answer"}, nil
		})
		if out.panicked != nil {
			t.Fatalf("a lost database panicked the turn: %v", out.panicked)
		}

		// The success write of the breaker, and the streamed text the defer
		// tries to keep once the final message write has failed.
		wantLost(t, out.logs, lostDBError(t, lost), "RecordCircuitResult", "UpdateAgentMessageContent")
	})

	t.Run("a turn that panics", func(t *testing.T) {
		var lost *store.Store
		out := driveTurn(t, func(st *store.Store, onDelta func(string)) (Turn, error) {
			lost = st
			onDelta("part of an answer")
			st.Close()
			panic("the loop blew up")
		})
		if out.panicked != "the loop blew up" {
			t.Fatalf("HandleMessage panicked with %v, want the original panic re-raised for the recover middleware", out.panicked)
		}

		wantLost(t, out.logs, lostDBError(t, lost), "UpdateAgentMessageContent", "MarkAIRunFailed", "RecordCircuitResult")
	})
}

// The converse: a bookkeeping write that works says nothing, so the new lines
// stay a signal.
func TestTerminalStateWritesThatWorkAreNotLogged(t *testing.T) {
	out := driveTurn(t, func(_ *store.Store, onDelta func(string)) (Turn, error) {
		onDelta("an answer")
		return Turn{Text: "an answer"}, nil
	})
	if out.panicked != nil || !strings.Contains(out.body, "message_end") {
		t.Fatalf("the turn did not finish (panic %v); stream:\n%s", out.panicked, out.body)
	}
	for _, op := range bookkeepingOps {
		if strings.Contains(out.logs.String(), op) {
			t.Errorf("a healthy turn logged %s; log:\n%s", op, out.logs)
		}
	}

	_, logs, done := plannerCall(t)
	done(nil)
	for _, op := range bookkeepingOps {
		if strings.Contains(logs.String(), op) {
			t.Errorf("a healthy planner call logged %s; log:\n%s", op, logs)
		}
	}
}
