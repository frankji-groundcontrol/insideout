package agent

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// rv4-AIDEV-05: a provider stream is a whole turn only once the provider says
// so (message_stop on Anthropic, response.completed or response.incomplete on
// Responses). Before this, a stream that just stopped came back as a normal
// turn, a tool call cut off by the output cap reached the tool executor with
// half a JSON object, and response.failed was ignored.

// sseData renders provider stream events the way both parsers read them: one
// "data: " line per event.
func sseData(events ...string) string {
	var b strings.Builder
	for _, e := range events {
		b.WriteString("data: " + e + "\n\n")
	}
	return b.String()
}

const (
	aStart     = `{"type":"message_start","message":{"usage":{"input_tokens":5,"output_tokens":0}}}`
	aTextOpen  = `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`
	aTextDelta = `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Recording that."}}`
	aTextStop  = `{"type":"content_block_stop","index":0}`
	aToolOpen  = `{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"record_fact","input":{}}}`
	aToolStop  = `{"type":"content_block_stop","index":1}`
	aMsgStop   = `{"type":"message_stop"}`
)

func aToolArgs(partial string) string {
	j, _ := json.Marshal(partial)
	return `{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":` + string(j) + `}}`
}

func aStopReason(reason string) string {
	return `{"type":"message_delta","delta":{"stop_reason":"` + reason + `"},"usage":{"output_tokens":9}}`
}

func TestTrustParseAnthropic_ToolUseCutAtTheCapIsNotACall(t *testing.T) {
	stream := sseData(aStart, aTextOpen, aTextDelta, aTextStop, aToolOpen, aToolArgs(`{"kind":"prob`), aToolStop, aStopReason("max_tokens"), aMsgStop)
	turn, err := parseAnthropicStream(strings.NewReader(stream), nil, nil)
	if err != nil {
		t.Fatalf("parseAnthropicStream: %v", err)
	}
	if turn.ToolCall != nil {
		t.Fatalf("turn.ToolCall = %+v: a call cut off by max_tokens carries half a JSON object and must not run", turn.ToolCall)
	}
	if !turn.Truncated {
		t.Fatal("turn.Truncated = false, want the cut-off surfaced")
	}
	if turn.Text != "Recording that." {
		t.Fatalf("turn.Text = %q, want the text that came before the cut kept", turn.Text)
	}
}

// A tool call and nothing else, cut by the cap, is still a truncated turn,
// not a refusal ("empty text response") and not a call.
func TestTrustParseAnthropic_ToolOnlyTurnCutAtTheCapIsTruncated(t *testing.T) {
	stream := sseData(aStart, aToolOpen, aToolArgs(`{"kind":`), aToolStop, aStopReason("max_tokens"), aMsgStop)
	turn, err := parseAnthropicStream(strings.NewReader(stream), nil, nil)
	if err != nil {
		t.Fatalf("parseAnthropicStream: %v", err)
	}
	if turn.ToolCall != nil || !turn.Truncated {
		t.Fatalf("turn = %+v, want a truncated turn with no call", turn)
	}
}

func TestTrustParseAnthropic_StreamWithoutMessageStopIsAnError(t *testing.T) {
	cases := map[string]string{
		"cut mid-answer":                 sseData(aStart, aTextOpen, aTextDelta),
		"end_turn but no message_stop":   sseData(aStart, aTextOpen, aTextDelta, aTextStop, aStopReason("end_turn")),
		"whole call but no message_stop": sseData(aStart, aToolOpen, aToolArgs(`{"kind":"problem"}`), aToolStop, aStopReason("tool_use")),
	}
	for name, stream := range cases {
		t.Run(name, func(t *testing.T) {
			turn, err := parseAnthropicStream(strings.NewReader(stream), nil, nil)
			if !errors.Is(err, ErrProviderTransient) {
				t.Fatalf("err = %v, want a transient provider error: the stream never said it was finished", err)
			}
			if turn.ToolCall != nil {
				t.Fatalf("turn.ToolCall = %+v from a stream that never finished", turn.ToolCall)
			}
		})
	}
}

const (
	rText = `{"type":"response.output_text.delta","delta":"half an ans"}`
	rCall = `{"type":"response.output_item.added","output_index":0,"item":{"id":"fc_1","type":"function_call","call_id":"call_1","name":"record_fact","arguments":""}}`
)

func rArgs(itemID, delta string) string {
	j, _ := json.Marshal(delta)
	return `{"type":"response.function_call_arguments.delta","item_id":"` + itemID + `","output_index":0,"delta":` + string(j) + `}`
}

func rEnd(event, status string) string {
	return `{"type":"` + event + `","response":{"status":"` + status + `","usage":{"input_tokens":3,"output_tokens":4096}}}`
}

func TestTrustParseResponses_FailedIsAProviderError(t *testing.T) {
	cases := []struct {
		code string
		want error
	}{
		{"server_error", ErrProviderTransient},
		{"rate_limit_exceeded", ErrProviderRateLimited},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			failed := `{"type":"response.failed","response":{"status":"failed","error":{"code":"` + tc.code + `","message":"upstream exploded"}}}`
			_, err := parseResponsesStream(strings.NewReader(sseData(rText, failed)), nil, nil)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want it to wrap %v", err, tc.want)
			}
			if !strings.Contains(err.Error(), "upstream exploded") || !strings.Contains(err.Error(), responsesProvider) {
				t.Fatalf("err = %q, want the provider's own message and the provider named", err)
			}
		})
	}
}

func TestTrustParseResponses_StreamWithoutAnEndIsAnError(t *testing.T) {
	cases := map[string]string{
		"text then EOF":    sseData(rText),
		"text then [DONE]": sseData(rText, "[DONE]"),
		"call then EOF":    sseData(rCall, rArgs("fc_1", `{"kind":"problem"}`)),
		"call then [DONE]": sseData(rCall, rArgs("fc_1", `{"kind":"problem"}`), "[DONE]"),
	}
	for name, stream := range cases {
		t.Run(name, func(t *testing.T) {
			turn, err := parseResponsesStream(strings.NewReader(stream), nil, nil)
			if !errors.Is(err, ErrProviderTransient) {
				t.Fatalf("err = %v, want a transient provider error: no response.completed ever arrived", err)
			}
			if turn.ToolCall != nil {
				t.Fatalf("turn.ToolCall = %+v from a stream that never finished", turn.ToolCall)
			}
		})
	}
}

// response.incomplete is how a Responses provider reports hitting
// max_output_tokens; a call cut by it must not run.
func TestTrustParseResponses_ACutOffCallIsNotExecuted(t *testing.T) {
	cut := rArgs("fc_1", `{"kind":"prob`)
	cases := map[string]string{
		"response.incomplete":          sseData(rCall, cut, rEnd("response.incomplete", "incomplete")),
		"completed with status cutoff": sseData(rCall, cut, rEnd("response.completed", "incomplete")),
	}
	for name, stream := range cases {
		t.Run(name, func(t *testing.T) {
			turn, err := parseResponsesStream(strings.NewReader(stream), nil, nil)
			if err != nil {
				t.Fatalf("parseResponsesStream: %v", err)
			}
			if turn.ToolCall != nil {
				t.Fatalf("turn.ToolCall = %+v: a call cut off by the cap carries half a JSON object and must not run", turn.ToolCall)
			}
			if !turn.Truncated {
				t.Fatal("turn.Truncated = false, want the cut-off surfaced")
			}
		})
	}
}

func TestTrustParseResponses_IncompleteTextIsTruncatedWithUsage(t *testing.T) {
	turn, err := parseResponsesStream(strings.NewReader(sseData(rText, rEnd("response.incomplete", "incomplete"))), nil, nil)
	if err != nil {
		t.Fatalf("parseResponsesStream: %v", err)
	}
	if turn.Text != "half an ans" || !turn.Truncated {
		t.Fatalf("turn = %+v, want the partial text flagged truncated", turn)
	}
	if turn.Usage.OutputTokens != 4096 {
		t.Fatalf("usage = %+v, want the usage response.incomplete carries", turn.Usage)
	}
}
