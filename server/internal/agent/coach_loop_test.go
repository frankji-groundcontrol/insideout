package agent

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// fakeStreamer answers StreamChat from a script the test supplies; the
// critic's forced-tool path is never reached from the coach loop.
type fakeStreamer struct {
	chat func(system string, onDelta func(string)) (Turn, error)
}

func (f fakeStreamer) StreamChat(_ context.Context, system string, _ []Message, _ []Tool, onDelta func(string)) (Turn, error) {
	return f.chat(system, onDelta)
}

func (fakeStreamer) StreamChatForcingTool(context.Context, string, []Message, Tool, func(string)) (Turn, error) {
	return Turn{}, nil
}

func (fakeStreamer) Model() string { return "fake" }

func discardLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// startRun opens the ai_run a turn heartbeats and writes telemetry against.
func startRun(t *testing.T, st *store.Store, ex *toolExecutor) *store.AIRun {
	t.Helper()
	ctx := context.Background()
	conv, err := st.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}
	run, err := st.CreateAIRun(ctx, conv.WorkspaceID, &ex.conversationID, ex.actorID, "test turn")
	if err != nil {
		t.Fatalf("create ai run: %v", err)
	}
	return run
}

// The model rewrites a section on provider call 1; call 2 must be prompted
// with the PRD as it is now, not the snapshot taken before the turn began.
func TestRunLoop_PromptShowsSectionWrittenEarlierInTheTurn(t *testing.T) {
	st := newTestStore(t)
	_, ex := seedConversation(t, st)
	ex.currentStage = StageDraft
	run := startRun(t, st, ex)
	prd, err := st.GetPrdForMember(context.Background(), ex.prdID, ex.actorID)
	if err != nil {
		t.Fatalf("get prd: %v", err)
	}

	written := "BACKGROUND-" + uuid.NewString()
	args, _ := json.Marshal(map[string]string{"section": "background", "markdown": written})
	var systems []string
	streamer := fakeStreamer{chat: func(system string, _ func(string)) (Turn, error) {
		systems = append(systems, system)
		if len(systems) == 1 {
			return Turn{ToolCall: &ToolCallRequest{ID: "call-1", Name: "update_prd_section", Arguments: string(args)}}, nil
		}
		return Turn{Text: "done"}, nil
	}}

	c := New(st, streamer, discardLog())
	got, err := c.runLoop(context.Background(), ex.actorID, run.ID, StageDraft, prd.Title, prd.Sections, nil, ex, nil)
	if err != nil {
		t.Fatalf("runLoop: %v", err)
	}
	if got != "done" || len(systems) != 2 {
		t.Fatalf("runLoop = %q after %d provider calls, want %q after 2", got, len(systems), "done")
	}
	if strings.Contains(systems[0], written) {
		t.Fatalf("the first prompt already contains text that was only written afterwards")
	}
	if !strings.Contains(systems[1], written) {
		t.Fatalf("the second prompt lacks the section the model wrote on call 1: it was built from the stale pre-turn snapshot")
	}
}

// If re-reading the PRD fails mid-turn the loop keeps prompting from the last
// snapshot it has: a stale prompt, never a failed turn.
func TestRunLoop_KeepsSnapshotWhenPrdRefreshFails(t *testing.T) {
	st := newTestStore(t)
	_, ex := seedConversation(t, st)
	ex.currentStage = StageDraft
	run := startRun(t, st, ex)
	ex.prdID = uuid.New() // no such PRD: every GetPrdForMember now fails, the refresh included

	snapshot := map[string]string{"background": "SNAPSHOT-" + uuid.NewString()}
	var systems []string
	streamer := fakeStreamer{chat: func(system string, _ func(string)) (Turn, error) {
		systems = append(systems, system)
		if len(systems) == 1 {
			return Turn{ToolCall: &ToolCallRequest{ID: "call-1", Name: "get_prd", Arguments: "{}"}}, nil
		}
		return Turn{Text: "done"}, nil
	}}

	c := New(st, streamer, discardLog())
	got, err := c.runLoop(context.Background(), ex.actorID, run.ID, StageDraft, "Title", snapshot, nil, ex, nil)
	if err != nil || got != "done" {
		t.Fatalf("runLoop = %q, %v; a failed PRD read must not fail the turn", got, err)
	}
	for i, s := range systems {
		if !strings.Contains(s, snapshot["background"]) {
			t.Fatalf("prompt %d dropped the pre-turn snapshot after the refresh failed", i+1)
		}
	}
}
