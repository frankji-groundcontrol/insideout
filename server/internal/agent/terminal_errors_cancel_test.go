package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

// A cancelled context is not a lost write. Only the run heartbeat runs on the
// request's own context, and a client that left mid-turn is routine (failTurn
// treats it as no fault), so it must not turn every disconnect into an error
// line. A terminal write runs detached and bounded, so what can end it is the
// bound, and that one is a lost write the operator must see.
func TestLogErr_ACancelledContextIsNotALostWrite(t *testing.T) {
	st := newTestStore(t)
	_, ex := seedConversation(t, st)
	run := startRun(t, st, ex)

	t.Run("a client that left", func(t *testing.T) {
		logs, log := capturingLog()
		gone, cancel := context.WithCancel(context.Background())
		cancel()

		logErr(log, "TouchAIRun", st.TouchAIRun(gone, ex.actorID, run.ID), "run_id", run.ID)

		if logs.Len() != 0 {
			t.Fatalf("a heartbeat cancelled by a client that left was logged as a lost write:\n%s", logs)
		}
	})

	t.Run("a write that outlived its bound", func(t *testing.T) {
		logs, log := capturingLog()
		late, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()
		<-late.Done()

		logErr(log, "MarkAIRunFailed", st.MarkAIRunFailed(late, ex.actorID, run.ID, "late"), "run_id", run.ID)

		wantLost(t, logs, "deadline exceeded", "MarkAIRunFailed")
		if !strings.Contains(logs.String(), run.ID.String()) {
			t.Errorf("the lost write does not say which run it was for (%s); log:\n%s", run.ID, logs)
		}
	})
}
