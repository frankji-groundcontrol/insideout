package agent

import (
	"bytes"
	"context"
	"fmt"
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

// The provider's own error text (up to 4 KiB of its response, which can carry
// gateway or account details) is for the operator: it belongs in the log and in
// ai_runs.error_message. The wire to the end user carries only the error code
// and a fixed message, as it does for every other failure class.
func TestHandleMessage_ProviderConfigErrorDoesNotLeakTheProviderBody(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	userID, ex := seedConversation(t, st)
	conv, err := st.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}

	const secret = "SECRET-BODY-123"
	var logs bytes.Buffer
	streamer := fakeStreamer{chat: func(string, func(string)) (Turn, error) {
		return Turn{}, fmt.Errorf("%w: agent: anthropic returned 401 Unauthorized: authentication_error: %s", ErrProviderConfig, secret)
	}}
	c := New(st, streamer, slog.New(slog.NewTextHandler(&logs, nil)))

	rec := httptest.NewRecorder()
	c.HandleMessage(rec, httptest.NewRequest(http.MethodPost, "/", nil),
		ex.conversationID.String(), userID, conv.WorkspaceID.String(), "hi")

	body := rec.Body.String()
	if !strings.Contains(body, `"code":"PROVIDER_CONFIG"`) {
		t.Fatalf("the stream has no PROVIDER_CONFIG error event:\n%s", body)
	}
	if strings.Contains(body, secret) {
		t.Fatalf("the SSE stream carried the provider's response body to the end user:\n%s", body)
	}
	if !strings.Contains(logs.String(), secret) {
		t.Fatalf("the operator's log lost the provider's own message; log:\n%s", logs.String())
	}
	var runError string
	if err := st.Pool.QueryRow(ctx, `SELECT error_message FROM insideout.ai_runs WHERE conversation_id = $1`, ex.conversationID).Scan(&runError); err != nil {
		t.Fatalf("read the run's error_message: %v", err)
	}
	if !strings.Contains(runError, secret) {
		t.Fatalf("ai_runs.error_message = %q, want the provider's actual message kept for the operator", runError)
	}
}
