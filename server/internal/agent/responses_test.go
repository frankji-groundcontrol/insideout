package agent

import (
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
