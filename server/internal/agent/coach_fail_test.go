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
	"github.com/google/uuid"
)

// Pure, so the global ai_circuit_breaker row is never touched: only failures
// that say something about the provider's health may feed the breaker.
func TestCountsAgainstCircuit(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"rate limited", fmt.Errorf("%w: anthropic returned 429", ErrProviderRateLimited), true},
		{"transient", fmt.Errorf("%w: anthropic returned 503", ErrProviderTransient), true},
		{"upstream stall", ErrUpstreamStall, true},
		{"unclassified provider error", errors.New("agent: anthropic returned 422 Unprocessable Entity: bad field"), true},
		{"context length", fmt.Errorf("%w: prompt is too long", ErrContextLength), false},
		{"provider config", fmt.Errorf("%w: anthropic returned 401", ErrProviderConfig), false},
		{"content refusal", fmt.Errorf("%w: model declined", ErrContentRefusal), false},
		{"client disconnect", fmt.Errorf("client write failed: %w", context.Canceled), false},
		{"iteration cap", fmt.Errorf("%w: exceeded max tool iterations (12)", ErrLoopInternal), false},
		{"store failure mid-loop", fmt.Errorf("%w: %w", ErrLoopInternal, errors.New("conversation lookup failed")), false},
		{"disconnect during a store read", fmt.Errorf("%w: %w", ErrLoopInternal, context.Canceled), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := countsAgainstCircuit(tc.err); got != tc.want {
				t.Fatalf("countsAgainstCircuit(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// A model stuck calling tools hits the iteration cap, and a store read can fail
// mid-turn. Both are our loop's failures, not the provider's: they must carry
// ErrLoopInternal so failTurn keeps them off the one global breaker (five of
// them would otherwise refuse every user for 30 s).
func TestRunLoop_OwnFailuresAreLoopInternal(t *testing.T) {
	st := newTestStore(t)
	_, ex := seedConversation(t, st)
	ex.currentStage = StageDraft
	run := startRun(t, st, ex)
	prd, err := st.GetPrdForMember(context.Background(), ex.prdID, ex.actorID)
	if err != nil {
		t.Fatalf("get prd: %v", err)
	}

	t.Run("iteration cap", func(t *testing.T) {
		streamer := fakeStreamer{chat: func(string, func(string)) (Turn, error) {
			return Turn{ToolCall: &ToolCallRequest{ID: "call", Name: "get_prd", Arguments: "{}"}}, nil
		}}
		c := New(st, streamer, discardLog())
		_, err := c.runLoop(context.Background(), ex.actorID, run.ID, StageDraft, prd.Title, prd.Sections, nil, ex, nil)
		if !errors.Is(err, ErrLoopInternal) {
			t.Fatalf("runLoop after %d tool calls = %v, want an ErrLoopInternal failure", maxToolIterations, err)
		}
		if countsAgainstCircuit(err) {
			t.Fatalf("the iteration-cap error %v would count against the circuit breaker", err)
		}
	})

	t.Run("conversation read fails", func(t *testing.T) {
		gone := *ex
		gone.conversationID = uuid.New() // no such conversation: the loop's own read fails
		streamer := fakeStreamer{chat: func(string, func(string)) (Turn, error) {
			t.Fatal("the provider must not be called when the loop cannot read its conversation")
			return Turn{}, nil
		}}
		c := New(st, streamer, discardLog())
		_, err := c.runLoop(context.Background(), ex.actorID, run.ID, StageDraft, prd.Title, prd.Sections, nil, &gone, nil)
		if !errors.Is(err, ErrLoopInternal) {
			t.Fatalf("runLoop with an unreadable conversation = %v, want an ErrLoopInternal failure", err)
		}
		if !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("runLoop error %v dropped its cause: want store.ErrNotFound still in the chain", err)
		}
	})
}

// loadHistory failing before the provider is ever called is the same class: a
// store error, not provider health. LIMIT -1 makes the read fail for real.
func TestLoadHistory_StoreFailureIsLoopInternal(t *testing.T) {
	st := newTestStore(t)
	_, ex := seedConversation(t, st)
	c := New(st, fakeStreamer{}, discardLog())

	if _, err := c.loadHistory(context.Background(), ex.actorID, ex.conversationID, -1); !errors.Is(err, ErrLoopInternal) {
		t.Fatalf("loadHistory with a failing read = %v, want an ErrLoopInternal failure", err)
	}

	// A client that disconnects during the read must still read as a disconnect.
	gone, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.loadHistory(gone, ex.actorID, ex.conversationID, historyLimit)
	if !errors.Is(err, ErrLoopInternal) || !errors.Is(err, context.Canceled) {
		t.Fatalf("loadHistory on a canceled request = %v, want both ErrLoopInternal and context.Canceled in the chain", err)
	}
}

// The client of a turn that hit the cap gets the same generic error as any
// internal failure, never the loop's own wording; the operator gets the full
// text in the log and in ai_runs.error_message.
func TestHandleMessage_IterationCapEndsInAGenericInternalError(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	userID, ex := seedConversation(t, st)
	conv, err := st.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}
	streamer := fakeStreamer{chat: func(string, func(string)) (Turn, error) {
		return Turn{ToolCall: &ToolCallRequest{ID: "call", Name: "get_prd", Arguments: "{}"}}, nil
	}}
	var logs bytes.Buffer
	c := New(st, streamer, slog.New(slog.NewTextHandler(&logs, nil)))

	rec := httptest.NewRecorder()
	c.HandleMessage(rec, httptest.NewRequest(http.MethodPost, "/", nil),
		ex.conversationID.String(), userID, conv.WorkspaceID.String(), "hi")

	body := rec.Body.String()
	if !strings.Contains(body, `"error":"internal error"`) || strings.Contains(body, "tool iterations") {
		t.Fatalf("want the generic internal error on the wire, without the loop's wording:\n%s", body)
	}
	if !strings.Contains(logs.String(), "level=ERROR") || !strings.Contains(logs.String(), "exceeded max tool iterations") {
		t.Fatalf("the cap failure was not logged at Error level; log:\n%s", logs.String())
	}
	var status, runError string
	if err := st.Pool.QueryRow(ctx, `SELECT status, error_message FROM insideout.ai_runs WHERE conversation_id = $1`, ex.conversationID).Scan(&status, &runError); err != nil {
		t.Fatalf("read the run: %v", err)
	}
	if status != "failed" || !strings.Contains(runError, "exceeded max tool iterations") {
		t.Fatalf("run = %s / %q, want failed with the cap message", status, runError)
	}
}
