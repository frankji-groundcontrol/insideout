package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// holdingStreamer streams one delta and then holds the turn open until its
// context ends, like a provider that is mid-answer when the server shuts down.
type holdingStreamer struct{ streaming chan struct{} }

func (h holdingStreamer) StreamChat(ctx context.Context, _ string, _ []Message, _ []Tool, onDelta func(string)) (Turn, error) {
	onDelta("partial answer")
	select {
	case h.streaming <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return Turn{}, ctx.Err()
}

func (holdingStreamer) StreamChatForcingTool(context.Context, string, []Message, Tool, func(string)) (Turn, error) {
	return Turn{}, nil
}

func (holdingStreamer) Model() string { return "holding" }

// REL-16: runServe cuts in-flight requests shortly before the shutdown
// window ends (it cancels the server's base context), so a coach turn has to
// reach its terminal state on that cancellation: the run failed, the streamed
// text kept with the interrupted marker, all of it persisted before
// HandleMessage returns. This already holds (failTurn's context.Canceled
// case plus the detached cleanup), so no coach change is needed; the test
// pins it, because the shutdown cut depends on it.
func TestHandleMessage_CutMidTurnEndsFailedWithTheStreamedText(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	userID, ex := seedConversation(t, st)
	conv, err := st.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}

	streaming := make(chan struct{}, 1)
	c := New(st, holdingStreamer{streaming: streaming}, discardLog())
	reqCtx, cut := context.WithCancel(ctx)
	defer cut()
	req := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(reqCtx)
	returned := make(chan struct{})
	go func() {
		defer close(returned)
		c.HandleMessage(httptest.NewRecorder(), req, ex.conversationID.String(), userID, conv.WorkspaceID.String(), "hi")
	}()

	select {
	case <-streaming:
	case <-time.After(10 * time.Second):
		t.Fatal("the turn never started streaming")
	}
	cut() // what cancelling the server's base context does to every in-flight request
	select {
	case <-returned:
	case <-time.After(10 * time.Second):
		t.Fatal("HandleMessage did not return after its request context was cut")
	}

	var status string
	if err := st.Pool.QueryRow(ctx, `SELECT status FROM insideout.ai_runs WHERE conversation_id = $1`, ex.conversationID).Scan(&status); err != nil {
		t.Fatalf("read ai_run: %v", err)
	}
	if status != "failed" {
		t.Fatalf("ai_run status = %q once HandleMessage returned, want %q (a cut turn must not be left running for the reaper)", status, "failed")
	}
	msgs, err := st.ListAgentMessages(ctx, ex.actorID, ex.conversationID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	var assistant *store.AgentMessage
	for i := range msgs {
		if msgs[i].Role == string(RoleAssistant) {
			assistant = &msgs[i]
		}
	}
	if assistant == nil {
		t.Fatal("no assistant message was created for the turn")
	}
	if want := "partial answer\n\n[interrupted]"; assistant.Content != want {
		t.Fatalf("assistant message = %q once HandleMessage returned, want the streamed text kept as %q", assistant.Content, want)
	}
}
