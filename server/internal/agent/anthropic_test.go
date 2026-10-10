package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestToAnthropicMessages_ToolRoundTrip(t *testing.T) {
	msgs := []Message{
		{Role: RoleUser, Content: "hi"},
		{Role: RoleAssistant, ToolCall: &ToolCallRequest{ID: "call_1", Name: "get_prd", Arguments: `{"a":1}`}},
		{Role: RoleTool, Content: "the prd", ToolCallID: "call_1"},
		{Role: RoleAssistant, Content: "done"},
	}
	out := toAnthropicMessages(msgs)
	if len(out) != 4 {
		t.Fatalf("len(out) = %d, want 4", len(out))
	}

	if out[0].Role != "user" || out[0].Content[0].Type != "text" || out[0].Content[0].Text != "hi" {
		t.Fatalf("user message: %+v", out[0])
	}

	assistantToolUse := out[1]
	if assistantToolUse.Role != "assistant" || assistantToolUse.Content[0].Type != "tool_use" {
		t.Fatalf("assistant tool_use message: %+v", assistantToolUse)
	}
	if assistantToolUse.Content[0].ID != "call_1" || assistantToolUse.Content[0].Name != "get_prd" {
		t.Fatalf("assistant tool_use fields: %+v", assistantToolUse.Content[0])
	}
	if string(assistantToolUse.Content[0].Input) != `{"a":1}` {
		t.Fatalf("assistant tool_use input = %q, want %q", assistantToolUse.Content[0].Input, `{"a":1}`)
	}

	// Anthropic has no "tool" role — a tool result must be a user message
	// with a tool_result block.
	toolResult := out[2]
	if toolResult.Role != "user" {
		t.Fatalf("tool result role = %q, want %q (Anthropic has no tool role)", toolResult.Role, "user")
	}
	if toolResult.Content[0].Type != "tool_result" || toolResult.Content[0].ToolUseID != "call_1" || toolResult.Content[0].Content != "the prd" {
		t.Fatalf("tool result content: %+v", toolResult.Content[0])
	}

	if out[3].Role != "assistant" || out[3].Content[0].Text != "done" {
		t.Fatalf("final assistant message: %+v", out[3])
	}
}

func TestToAnthropicTools(t *testing.T) {
	tools := []Tool{
		{Name: "get_prd", Description: "reads the prd", Parameters: map[string]any{"type": "object"}},
	}
	out := toAnthropicTools(tools)
	if len(out) != 1 || out[0].Name != "get_prd" || out[0].Description != "reads the prd" {
		t.Fatalf("toAnthropicTools = %+v", out)
	}
}

// newStallingServer sends SSE headers plus one ping, then goes silent until
// the client gives up — a real hung-upstream over real HTTP (no mocks). The
// idle watchdog must trip while the request is still alive.
func newStallingServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "event: ping\ndata: {\"type\":\"ping\"}\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done() // go silent; wait for the client to tear down
	}))
}

func newTestStreamer(url string, idle time.Duration) *AnthropicStreamer {
	return &AnthropicStreamer{
		baseURL:     url,
		authToken:   "test-token",
		model:       "claude-test",
		maxTokens:   16,
		client:      &http.Client{},
		idleTimeout: idle,
	}
}

// TestStreamChat_UpstreamStall is the F7 regression: a provider that goes
// silent on an otherwise-healthy, still-connected request must surface
// ErrUpstreamStall — NOT context.Canceled, which coach.go would silently
// treat as a routine client disconnect (no SSE error, no circuit failure).
func TestStreamChat_UpstreamStall(t *testing.T) {
	srv := newStallingServer(t)
	defer srv.Close()

	a := newTestStreamer(srv.URL, 50*time.Millisecond) // short leash so the watchdog trips fast
	_, err := a.StreamChat(context.Background(), "sys", []Message{{Role: RoleUser, Content: "hi"}}, nil, nil)
	if !errors.Is(err, ErrUpstreamStall) {
		t.Fatalf("err = %v, want ErrUpstreamStall", err)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v wraps context.Canceled — a stall must not be mislabeled as a disconnect", err)
	}
}

// TestStreamChat_ClientDisconnect is the other half: when the client really
// does hang up, the failure stays a routine context.Canceled, not a stall.
func TestStreamChat_ClientDisconnect(t *testing.T) {
	srv := newStallingServer(t)
	defer srv.Close()

	a := newTestStreamer(srv.URL, 5*time.Second) // long leash; the client cancels first
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel() // the "client" disconnects while the stream is silent
	}()
	_, err := a.StreamChat(ctx, "sys", []Message{{Role: RoleUser, Content: "hi"}}, nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if errors.Is(err, ErrUpstreamStall) {
		t.Fatalf("err = %v, a genuine disconnect must not be reported as an upstream stall", err)
	}
}

// AIDEV-09 wire guard: an empty messages payload must fail locally with a
// named cause — it would otherwise marshal to "messages": [] and cost a
// provider round trip that always 400s.
func TestAnthropic_ForcingToolRejectsEmptyMessages(t *testing.T) {
	a := newTestStreamer("http://unused.test", time.Second)
	_, err := a.StreamChatForcingTool(context.Background(), "sys", nil, Tool{Name: "report_findings"}, nil)
	if err == nil || !strings.Contains(err.Error(), "at least one message") {
		t.Fatalf("want named empty-messages error, got %v", err)
	}
}

// The classifier is shared with the Responses streamer, so the provider it
// names is a parameter; the Anthropic streamer must keep naming itself.
func TestAnthropicStreamer_HTTPErrorNamesAnthropic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`))
	}))
	defer srv.Close()

	a := newTestStreamer(srv.URL, time.Second)
	_, err := a.StreamChat(context.Background(), "sys", []Message{{Role: RoleUser, Content: "hi"}}, nil, nil)
	if !errors.Is(err, ErrProviderConfig) {
		t.Fatalf("err = %v, want wrapping ErrProviderConfig", err)
	}
	if want := "agent: anthropic returned 401 Unauthorized: authentication_error: invalid x-api-key"; !strings.Contains(err.Error(), want) {
		t.Fatalf("err = %q, want it to contain %q", err, want)
	}
}
