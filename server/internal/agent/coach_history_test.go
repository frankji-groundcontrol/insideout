package agent

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// historyRecorder is a coach-turn stub that keeps the messages every provider
// call received. failFirst makes call 1 fail before any text is streamed.
type historyRecorder struct {
	calls     [][]Message
	failFirst error
}

func (h *historyRecorder) StreamChat(_ context.Context, _ string, msgs []Message, _ []Tool, _ func(string)) (Turn, error) {
	h.calls = append(h.calls, slices.Clone(msgs))
	if len(h.calls) == 1 && h.failFirst != nil {
		return Turn{}, h.failFirst
	}
	return Turn{Text: "ok"}, nil
}

func (*historyRecorder) StreamChatForcingTool(context.Context, string, []Message, Tool, func(string)) (Turn, error) {
	return Turn{}, nil
}

func (*historyRecorder) Model() string { return "recorder" }

func roleText(msgs []Message) []string {
	out := make([]string, len(msgs))
	for i, m := range msgs {
		out[i] = string(m.Role) + ":" + m.Content
	}
	return out
}

// turnAfterOneExchange runs one coach turn on a conversation that already holds
// one stored exchange and returns what the provider received, plus the
// transcript as stored afterwards.
func turnAfterOneExchange(t *testing.T, stub *historyRecorder) (calls [][]Message, stored []string) {
	t.Helper()
	st := newTestStore(t)
	ctx := context.Background()
	userID, ex := seedConversation(t, st)
	conv, err := st.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}
	for _, m := range []struct{ role, content string }{{"user", "earlier question"}, {"assistant", "earlier answer"}} {
		if _, err := st.InsertAgentMessage(ctx, ex.actorID, ex.conversationID, m.role, m.content, nil, nil, nil); err != nil {
			t.Fatalf("seed %s message: %v", m.role, err)
		}
	}

	c := New(st, stub, discardLog())
	c.HandleMessage(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil),
		ex.conversationID.String(), userID, conv.WorkspaceID.String(), "latest question")

	msgs, err := st.ListAgentMessages(ctx, ex.actorID, ex.conversationID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	for _, m := range msgs {
		stored = append(stored, m.Role+":"+m.Content)
	}
	return stub.calls, stored
}

// The member's message is stored before the history is read, so the history
// already ends with it. Appending it again showed the model the latest user
// turn twice on every turn. What is stored must stay the same either way.
func TestHandleMessage_ModelSeesTheLatestUserTurnOnce(t *testing.T) {
	calls, stored := turnAfterOneExchange(t, &historyRecorder{})

	wantStored := []string{"user:earlier question", "assistant:earlier answer", "user:latest question", "assistant:ok"}
	if !slices.Equal(stored, wantStored) {
		t.Fatalf("stored transcript = %v, want %v", stored, wantStored)
	}
	if len(calls) != 1 {
		t.Fatalf("%d provider calls, want 1", len(calls))
	}
	want := []string{"user:earlier question", "assistant:earlier answer", "user:latest question"}
	if got := roleText(calls[0]); !slices.Equal(got, want) {
		t.Fatalf("model input = %v, want %v: the latest user turn must appear exactly once, last", got, want)
	}
}

// The context-length retry rebuilds the history with a smaller window; it must
// not repeat the latest user turn either.
func TestHandleMessage_TightenedRetryAlsoSeesTheLatestUserTurnOnce(t *testing.T) {
	stub := &historyRecorder{failFirst: fmt.Errorf("%w: prompt is too long", ErrContextLength)}
	calls, _ := turnAfterOneExchange(t, stub)

	if len(calls) != 2 {
		t.Fatalf("%d provider calls, want 2 (the failed one and the tightened retry)", len(calls))
	}
	want := []string{"user:earlier question", "assistant:earlier answer", "user:latest question"}
	if got := roleText(calls[1]); !slices.Equal(got, want) {
		t.Fatalf("tightened retry input = %v, want %v: the latest user turn must appear exactly once, last", got, want)
	}
}
