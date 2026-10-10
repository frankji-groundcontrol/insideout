package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// USER-03: the offline template coach must disclose itself in-band. Its reply
// is a bare, locale-neutral echo of the member's message — no operator
// env-var instructions, no fixed language — and the turn's message_start
// frame carries offline:true so the client can badge the panel instead of
// passing canned text off as real coaching.
func TestTemplate_ReplyEchoesWithoutOperatorInstructions(t *testing.T) {
	var reply string
	_, err := NewTemplateStreamer().StreamChat(
		context.Background(), "",
		[]Message{{Role: RoleUser, Content: "who is it for?"}},
		nil, func(s string) { reply = s },
	)
	if err != nil {
		t.Fatalf("stream chat: %v", err)
	}
	if reply != "who is it for?" {
		t.Fatalf("template reply = %q, want the user's message echoed with nothing added", reply)
	}
}

func TestHandleMessage_TemplateStreamDisclosesOffline(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	userID, ex := seedConversation(t, st)
	conv, err := st.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}

	rec := httptest.NewRecorder()
	c := New(st, NewTemplateStreamer(), discardLog())
	c.HandleMessage(rec, httptest.NewRequest(http.MethodPost, "/", nil),
		ex.conversationID.String(), userID, conv.WorkspaceID.String(), "hello")

	payload := messageStartData(t, rec.Body.String())
	if payload["offline"] != true {
		t.Fatalf("message_start offline = %v (%T), want explicit true so the client can badge the canned reply; frame payload: %v", payload["offline"], payload["offline"], payload)
	}
	if payload["model"] != "template" {
		t.Fatalf("message_start model = %v, want the streamer identity \"template\"", payload["model"])
	}
}

// messageStartData returns the JSON payload of the message_start frame in an
// SSE response body.
func messageStartData(t *testing.T, body string) map[string]any {
	t.Helper()
	for _, frame := range strings.Split(body, "\n\n") {
		if !strings.Contains(frame, "event: message_start") {
			continue
		}
		for _, line := range strings.Split(frame, "\n") {
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var payload map[string]any
			if err := json.Unmarshal([]byte(line[len("data: "):]), &payload); err != nil {
				t.Fatalf("message_start data is not JSON: %v", err)
			}
			return payload
		}
	}
	t.Fatalf("no message_start frame in body:\n%s", body)
	return nil
}
