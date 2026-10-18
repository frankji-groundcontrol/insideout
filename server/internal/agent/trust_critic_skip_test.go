package agent

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// rv4-AIDEV-02: a critic pass that did not run (queue contention, or two
// unusable outputs) left only a meta marker that opens the finalize gate. The
// person was never told, and the marker outlived a later round that did run, so
// a reviewed PRD could still read as "skipped".

// silentStub never calls the forced tool, which runCritic sees as "the model
// did not call report_findings" on both of its attempts.
type silentStub struct{}

func (silentStub) StreamChat(context.Context, string, []Message, []Tool, func(string)) (Turn, error) {
	return Turn{}, nil
}

func (silentStub) StreamChatForcingTool(context.Context, string, []Message, Tool, func(string)) (Turn, error) {
	return Turn{}, nil
}

func (silentStub) Model() string { return "stub" }

func storedCriticState(t *testing.T, st *store.Store, ex *toolExecutor) criticState {
	t.Helper()
	conv, err := st.GetConversationForOwner(context.Background(), ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("reload conversation: %v", err)
	}
	_, extra := loadLedger(conv.Meta)
	return loadCriticState(extra)
}

func TestMaybeRunCritic_ContentionSkipIsAnnouncedOnTheStream(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)
	run := startRun(t, st, ex)
	rec := httptest.NewRecorder()
	sse, _ := newSSEWriter(rec)

	c := New(st, &forcingStub{}, discardLog())
	c.maybeRunCritic(ctx, ex.actorID, ex.prdID, ex.conversationID, run.ID, StageCritique, "no ranked list", false, true, sse)

	if got := storedCriticState(t, st, ex).Skipped; got != "contention" {
		t.Fatalf("stored skip marker = %q, want contention", got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: critic_skipped") || !strings.Contains(body, `"reason":"contention"`) || !strings.Contains(body, `"rounds":0`) {
		t.Fatalf("a skipped critic must tell the person on the stream; the stream carried %q", body)
	}
}

func TestMaybeRunCritic_OutputFailureSkipIsAnnouncedOnTheStream(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)
	run := startRun(t, st, ex)
	rec := httptest.NewRecorder()
	sse, _ := newSSEWriter(rec)

	c := New(st, silentStub{}, discardLog())
	c.maybeRunCritic(ctx, ex.actorID, ex.prdID, ex.conversationID, run.ID, StageCritique, "no ranked list", false, false, sse)

	if got := storedCriticState(t, st, ex).Skipped; got != "critic-output-failure" {
		t.Fatalf("stored skip marker = %q, want critic-output-failure", got)
	}
	if body := rec.Body.String(); !strings.Contains(body, "event: critic_skipped") || !strings.Contains(body, `"reason":"critic-output-failure"`) {
		t.Fatalf("a critic that produced no usable output must tell the person; the stream carried %q", body)
	}
}

// The critic retries on every critique turn while no round has run, so a skip
// from a busy moment is routinely followed by a round that does run. The
// finalize card reads the marker, so it must not outlive that round.
func TestMaybeRunCritic_ARoundThatRunsClearsTheSkipMarker(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)
	run := startRun(t, st, ex)
	sse, _ := newSSEWriter(httptest.NewRecorder())
	c := New(st, &forcingStub{}, discardLog())

	c.maybeRunCritic(ctx, ex.actorID, ex.prdID, ex.conversationID, run.ID, StageCritique, "no ranked list", false, true, sse)
	if got := storedCriticState(t, st, ex); got.Skipped != "contention" || got.RoundCount != 0 {
		t.Fatalf("precondition: a contended turn should leave {0 contention}, got %+v", got)
	}

	c.maybeRunCritic(ctx, ex.actorID, ex.prdID, ex.conversationID, run.ID, StageCritique, "no ranked list", false, false, sse)
	if got := storedCriticState(t, st, ex); got.RoundCount != 1 || got.Skipped != "" {
		t.Fatalf("after a round that ran, state = %+v, want roundCount 1 and no skip marker (the PRD was reviewed)", got)
	}
}
