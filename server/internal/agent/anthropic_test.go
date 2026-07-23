package agent

import (
	"strings"
	"testing"
)

// realThinkingThenTextStream is a real SSE response captured from a live
// Anthropic-compatible endpoint (claude-sonnet-4-6, extended thinking
// enabled) — the exact scenario that broke the old langchaingo-based
// implementation, which only ever read Choices[0] and got the empty
// thinking choice instead of the real answer.
const realThinkingThenTextStream = `event: message_start
data: {"message":{"content":[],"id":"06ade102991aa94deb49ac9ac4b7d56d","model":"claude-sonnet-4-6","role":"assistant","stop_reason":null,"stop_sequence":null,"type":"message","usage":{"input_tokens":47,"output_tokens":0}},"type":"message_start"}

event: ping
data: {"type":"ping"}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"The user asks a simple question."}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"7254671620b06ac0624bcebbf123928a95fa4de340fb3ee72109a53a23eee5c8"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: content_block_start
data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Hello! It's wonderful"}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":" to see you here today."}}

event: content_block_stop
data: {"type":"content_block_stop","index":1}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"input_tokens":47,"output_tokens":69}}

event: message_stop
data: {"type":"message_stop"}
`

// realThinkingTextThenToolUseStream is a real capture of a response that
// thinks, narrates a line of text, then calls a tool — three content
// blocks in one turn.
const realThinkingTextThenToolUseStream = `event: message_start
data: {"message":{"content":[],"id":"06ade14529edd3fafca9ad7c366ebf2b","model":"claude-sonnet-4-6","role":"assistant","stop_reason":null,"stop_sequence":null,"type":"message","usage":{"input_tokens":172,"output_tokens":0}},"type":"message_start"}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"Let me check the PRD first."}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: content_block_start
data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Let me first check the current PRD status.\n"}}

event: content_block_stop
data: {"type":"content_block_stop","index":1}

event: content_block_start
data: {"type":"content_block_start","index":2,"content_block":{"type":"tool_use","id":"call_function_4veuf3y571vf_1","name":"get_prd","input":{}}}

event: content_block_delta
data: {"type":"content_block_delta","index":2,"delta":{"type":"input_json_delta","partial_json":"{}"}}

event: content_block_stop
data: {"type":"content_block_stop","index":2}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"input_tokens":172,"output_tokens":111}}

event: message_stop
data: {"type":"message_stop"}
`

func TestParseAnthropicStream_ThinkingThenText(t *testing.T) {
	var deltas []string
	turn, err := parseAnthropicStream(strings.NewReader(realThinkingThenTextStream), nil, func(s string) { deltas = append(deltas, s) })
	if err != nil {
		t.Fatalf("parseAnthropicStream: %v", err)
	}
	if turn.ToolCall != nil {
		t.Fatalf("turn.ToolCall = %+v, want nil", turn.ToolCall)
	}
	const want = "Hello! It's wonderful to see you here today."
	if turn.Text != want {
		t.Fatalf("turn.Text = %q, want %q", turn.Text, want)
	}
	if got := strings.Join(deltas, ""); got != want {
		t.Fatalf("streamed deltas joined = %q, want %q", got, want)
	}
}

func TestParseAnthropicStream_ThinkingTextThenToolUse(t *testing.T) {
	turn, err := parseAnthropicStream(strings.NewReader(realThinkingTextThenToolUseStream), nil, nil)
	if err != nil {
		t.Fatalf("parseAnthropicStream: %v", err)
	}
	if turn.ToolCall == nil {
		t.Fatal("turn.ToolCall = nil, want a get_prd call")
	}
	if turn.ToolCall.Name != "get_prd" {
		t.Fatalf("turn.ToolCall.Name = %q, want %q", turn.ToolCall.Name, "get_prd")
	}
	if turn.ToolCall.ID != "call_function_4veuf3y571vf_1" {
		t.Fatalf("turn.ToolCall.ID = %q, want the real id", turn.ToolCall.ID)
	}
	if turn.ToolCall.Arguments != "{}" {
		t.Fatalf("turn.ToolCall.Arguments = %q, want %q", turn.ToolCall.Arguments, "{}")
	}
}

func TestParseAnthropicStream_Empty(t *testing.T) {
	if _, err := parseAnthropicStream(strings.NewReader(""), nil, nil); err == nil {
		t.Fatal("parseAnthropicStream(empty) should error, got nil")
	}
}

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
