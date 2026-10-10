package agent

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// A NUL is the one character PostgreSQL's text type refuses, so a final reply
// carrying one makes the final assistant-message write fail for real, with no
// fault injection and no mock. The text the client saw streamed (clean) must
// then still be persisted by the failure defer, and the failed write logged,
// instead of leaving the message empty with nothing in the logs.
func TestHandleMessage_FailedFinalWriteFallsBackToTheStreamedText(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	userID, ex := seedConversation(t, st)
	conv, err := st.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}

	var logs bytes.Buffer
	streamer := fakeStreamer{chat: func(_ string, onDelta func(string)) (Turn, error) {
		onDelta("hello")
		return Turn{Text: "hello\x00"}, nil
	}}
	c := New(st, streamer, slog.New(slog.NewTextHandler(&logs, nil)))

	c.HandleMessage(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil),
		ex.conversationID.String(), userID, conv.WorkspaceID.String(), "hi")

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
	if want := "hello\n\n[interrupted]"; assistant.Content != want {
		t.Fatalf("assistant message = %q after the final write failed, want the streamed text kept as %q", assistant.Content, want)
	}
	if !strings.Contains(logs.String(), "persist final assistant message") {
		t.Fatalf("the failed final write was not logged; log:\n%s", logs.String())
	}
}
