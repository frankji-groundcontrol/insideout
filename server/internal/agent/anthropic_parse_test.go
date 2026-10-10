package agent

import (
	"errors"
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

// textStreamWithStop builds a minimal one-block text stream whose
// message_delta carries the given stop_reason, for exercising the
// max_tokens (truncated) vs end_turn (complete) distinction (F13).
func textStreamWithStop(stopReason string) string {
	return `event: message_start
data: {"type":"message_start","message":{"usage":{"input_tokens":1,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"partial answer"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"` + stopReason + `"},"usage":{"output_tokens":4}}

event: message_stop
data: {"type":"message_stop"}
`
}

func TestParseAnthropicStream_MaxTokensIsTruncated(t *testing.T) {
	turn, err := parseAnthropicStream(strings.NewReader(textStreamWithStop("max_tokens")), nil, nil)
	if err != nil {
		t.Fatalf("parseAnthropicStream: %v", err)
	}
	if turn.Text != "partial answer" {
		t.Fatalf("turn.Text = %q, want the partial text preserved", turn.Text)
	}
	if !turn.Truncated {
		t.Fatal("turn.Truncated = false, want true for stop_reason=max_tokens")
	}
}

func TestParseAnthropicStream_EndTurnNotTruncated(t *testing.T) {
	turn, err := parseAnthropicStream(strings.NewReader(textStreamWithStop("end_turn")), nil, nil)
	if err != nil {
		t.Fatalf("parseAnthropicStream: %v", err)
	}
	if turn.Truncated {
		t.Fatal("turn.Truncated = true, want false for a natural end_turn")
	}
}

// errorEventStream is a one-line SSE "error" event of the given provider
// error.type / message — the in-stream failure shape F17 must classify
// instead of collapsing into an opaque message.
func errorEventStream(errType, msg string) string {
	return "event: error\ndata: {\"type\":\"error\",\"error\":{\"type\":\"" + errType + "\",\"message\":\"" + msg + "\"}}\n"
}

func TestParseAnthropicStream_ErrorEventTaxonomy(t *testing.T) {
	cases := []struct {
		errType string
		msg     string
		want    error
	}{
		{"rate_limit_error", "too many requests", ErrProviderRateLimited},
		{"overloaded_error", "server busy", ErrProviderTransient},
		{"api_error", "boom", ErrProviderTransient},
		{"invalid_request_error", "prompt is too long: context window exceeded", ErrContextLength},
		{"authentication_error", "bad key", ErrProviderConfig},
		{"permission_error", "no", ErrProviderConfig},
	}
	for _, tc := range cases {
		_, err := parseAnthropicStream(strings.NewReader(errorEventStream(tc.errType, tc.msg)), nil, nil)
		if !errors.Is(err, tc.want) {
			t.Fatalf("error.type=%q → %v, want errors.Is %v", tc.errType, err, tc.want)
		}
	}
}

func TestParseAnthropicStream_ErrorEventUnknownType(t *testing.T) {
	// An unrecognized type still surfaces an error, but maps to none of the
	// taxonomy sentinels — it must not be mislabeled transient/rate-limited.
	_, err := parseAnthropicStream(strings.NewReader(errorEventStream("weird_new_error", "?")), nil, nil)
	if err == nil {
		t.Fatal("want an error for an unknown in-stream error event")
	}
	for _, s := range []error{ErrProviderRateLimited, ErrProviderTransient, ErrContextLength, ErrProviderConfig} {
		if errors.Is(err, s) {
			t.Fatalf("unknown error type should not classify as %v", s)
		}
	}
}
