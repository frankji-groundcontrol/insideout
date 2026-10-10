package agent

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// trickleStreamer is the slow-trickle failure mode: a provider that never
// goes silent long enough for the 90s idle watchdog and never finishes on its
// own — it holds the stream open until the caller's context gives up.
// safety bounds the wait so an unbounded turn fails the test instead of
// hanging it.
type trickleStreamer struct {
	safety time.Duration
}

func (s trickleStreamer) StreamChat(ctx context.Context, _ string, _ []Message, _ []Tool, _ func(string)) (Turn, error) {
	select {
	case <-ctx.Done():
		return Turn{}, ctx.Err()
	case <-time.After(s.safety):
		return Turn{}, errors.New("the coach turn outlived the test's safety bound: no total-turn deadline cut it")
	}
}

func (trickleStreamer) StreamChatForcingTool(context.Context, string, []Message, Tool, func(string)) (Turn, error) {
	return Turn{}, nil
}
func (trickleStreamer) Model() string { return "trickle" }

// A trickling provider defeats both existing bounds — maxToolIterations never
// runs out (no tool calls) and the idle watchdog never fires (no silent gap) —
// so a total-turn deadline must cut the turn and end it as a classified
// non-circuit failure, handing back its dispatch permit, instead of pinning a
// permit and the conversation lock indefinitely.
func TestHandleMessage_TotalTurnDeadlineCutsATricklingTurn(t *testing.T) {
	t.Setenv("INSIDEOUT_AI_MAX_CONCURRENT", "1") // the trickled permit is the only one
	t.Setenv("INSIDEOUT_AI_TURN_DEADLINE", "150ms")
	st := newTestStore(t)
	_, ex := seedConversation(t, st)
	conv, err := st.GetConversationForOwner(context.Background(), ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}

	c := New(st, trickleStreamer{safety: 5 * time.Second}, discardLog())
	userID, workspaceID := ex.actorID, conv.WorkspaceID
	resetBreaker(t, c)
	failures0, last0 := breakerState(t, c)

	rec := httptest.NewRecorder()
	c.HandleMessage(rec, httptest.NewRequest(http.MethodPost, "/", nil),
		ex.conversationID.String(), userID.String(), workspaceID.String(), "hello coach")

	var status, errMsg string
	if err := st.Pool.QueryRow(context.Background(),
		`SELECT status, coalesce(error_message, '') FROM insideout.ai_runs WHERE conversation_id = $1`,
		ex.conversationID).Scan(&status, &errMsg); err != nil {
		t.Fatalf("read the run: %v", err)
	}
	if status != "failed" {
		t.Fatalf("ai_run status = %q, want failed", status)
	}
	if !strings.Contains(errMsg, "total turn deadline") {
		t.Fatalf("ai_run.error_message = %q, want the turn cut and surfaced as the total-turn-deadline failure", errMsg)
	}
	if !strings.Contains(rec.Body.String(), "event: error") {
		t.Fatalf("the still-connected client got no error event for the cut turn:\n%s", rec.Body.String())
	}
	failures1, last1 := breakerState(t, c)
	if failures1 != failures0 || last1 != last0 {
		t.Fatalf("a deadline-cut turn fed the circuit breaker: failures %d->%d, last_failure epoch %v->%v", failures0, failures1, last0, last1)
	}
	// The permit came back with the turn: with maxConcurrent=1 the next
	// acquire must succeed immediately, not queue or refuse.
	if _, done, ok := admit(c, userID, workspaceID); !ok {
		t.Fatal("the permit was still held after the deadline-cut turn")
	} else {
		done(nil)
	}
}
