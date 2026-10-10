package agent

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestToResponsesInputRoundTrip(t *testing.T) {
	in := toResponsesInput([]Message{
		{Role: RoleUser, Content: "hi"},
		{Role: RoleAssistant, ToolCall: &ToolCallRequest{ID: "c1", Name: "get_prd", Arguments: `{"id":"1"}`}},
		{Role: RoleTool, ToolCallID: "c1", Content: "prd"},
	})
	if in[0]["role"] != "user" || in[0]["content"] != "hi" {
		t.Fatalf("user: %+v", in[0])
	}
	if in[1]["type"] != "function_call" || in[1]["call_id"] != "c1" {
		t.Fatalf("call: %+v", in[1])
	}
	if in[2]["type"] != "function_call_output" || in[2]["call_id"] != "c1" {
		t.Fatalf("output: %+v", in[2])
	}
}

func TestParseResponsesStreamTextAndUsage(t *testing.T) {
	raw := strings.Join([]string{
		`event: response.output_text.delta`,
		`data: {"type":"response.output_text.delta","delta":"Hel"}`,
		``,
		`event: response.output_text.delta`,
		`data: {"type":"response.output_text.delta","delta":"lo"}`,
		``,
		`event: response.completed`,
		`data: {"type":"response.completed","response":{"usage":{"input_tokens":3,"output_tokens":2},"status":"completed"}}`,
		``,
	}, "\n")
	var got strings.Builder
	turn, err := parseResponsesStream(strings.NewReader(raw), nil, func(s string) { got.WriteString(s) })
	if err != nil {
		t.Fatal(err)
	}
	if turn.Text != "Hello" || got.String() != "Hello" {
		t.Fatalf("text=%q streamed=%q", turn.Text, got.String())
	}
	if turn.Usage.InputTokens != 3 || turn.Usage.OutputTokens != 2 {
		t.Fatalf("usage=%+v", turn.Usage)
	}
}

func TestParseResponsesStreamToolCall(t *testing.T) {
	raw := strings.Join([]string{
		`data: {"type":"response.output_item.added","item":{"type":"function_call","call_id":"c9","name":"get_prd"}}`,
		``,
		`data: {"type":"response.function_call_arguments.delta","delta":"{\"x\":"}`,
		``,
		`data: {"type":"response.function_call_arguments.delta","delta":"1}"}`,
		``,
	}, "\n")
	turn, err := parseResponsesStream(strings.NewReader(raw), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if turn.ToolCall == nil || turn.ToolCall.Name != "get_prd" || turn.ToolCall.ID != "c9" {
		t.Fatalf("tool=%+v", turn.ToolCall)
	}
	if turn.ToolCall.Arguments != `{"x":1}` {
		t.Fatalf("args=%q", turn.ToolCall.Arguments)
	}
}

// AIDEV-05 / AIDEV-18: a Responses-API provider's HTTP failure used to be
// reported as "agent: anthropic returned ..." — a misleading on-call signal
// in the logs and in ai_runs.error_message. The classification (a config
// error, never retried) is right; only the provider named in the text was not.
func TestResponsesStreamer_HTTPErrorNamesTheResponsesProvider(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"type":"invalid_api_key","message":"bad key"}}`))
	}))
	defer srv.Close()

	s, err := NewResponsesStreamer(srv.URL, "test-token", "gpt-test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.StreamChat(context.Background(), "sys", []Message{{Role: RoleUser, Content: "hi"}}, nil, nil)
	if !errors.Is(err, ErrProviderConfig) {
		t.Fatalf("err = %v, want wrapping ErrProviderConfig", err)
	}
	if strings.Contains(err.Error(), "anthropic") {
		t.Fatalf("err = %q blames anthropic for a Responses-API failure", err)
	}
	if want := "agent: openai-compatible responses returned 401 Unauthorized: invalid_api_key: bad key"; !strings.Contains(err.Error(), want) {
		t.Fatalf("err = %q, want it to contain %q", err, want)
	}
}
