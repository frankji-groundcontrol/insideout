package agent

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// contentBlockAcc accumulates one content_block's streamed deltas, keyed
// by the block's index in the response.
type contentBlockAcc struct {
	blockType string
	text      strings.Builder
	toolID    string
	toolName  string
	toolInput strings.Builder
}

// parseAnthropicStream reads Anthropic's Messages API SSE stream
// (message_start / content_block_start / content_block_delta /
// content_block_stop / message_delta / message_stop — see
// https://docs.anthropic.com/en/api/messages-streaming) and assembles it
// into one Turn. Extended-thinking models emit a "thinking" block before
// the real "text"/"tool_use" block; thinking deltas are consumed (so
// bufio doesn't choke on long lines) but never contribute to the output.
func parseAnthropicStream(r io.Reader, tick func(), onDelta func(string)) (Turn, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	blocks := make(map[int]*contentBlockAcc)
	var order []int
	var stopReason string
	var usage Usage
	var sawStop bool

	for scanner.Scan() {
		if tick != nil {
			tick()
		}
		line := scanner.Text()
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			continue
		}

		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue // blank lines between events, "event: X" lines, etc.
		}
		eventType, _ := event["type"].(string)

		switch eventType {
		case "message_start":
			if msg, ok := event["message"].(map[string]any); ok {
				readUsage(msg["usage"], &usage)
			}

		case "content_block_start":
			idx, ok := event["index"].(float64)
			if !ok {
				continue
			}
			index := int(idx)
			cb, _ := event["content_block"].(map[string]any)
			acc := &contentBlockAcc{blockType: fmt.Sprint(cb["type"])}
			if acc.blockType == "tool_use" {
				acc.toolID, _ = cb["id"].(string)
				acc.toolName, _ = cb["name"].(string)
			}
			blocks[index] = acc
			order = append(order, index)

		case "content_block_delta":
			idx, ok := event["index"].(float64)
			if !ok {
				continue
			}
			acc := blocks[int(idx)]
			if acc == nil {
				continue
			}
			delta, _ := event["delta"].(map[string]any)
			switch delta["type"] {
			case "text_delta":
				chunk, _ := delta["text"].(string)
				acc.text.WriteString(chunk)
				if onDelta != nil && chunk != "" {
					onDelta(chunk)
				}
			case "input_json_delta":
				chunk, _ := delta["partial_json"].(string)
				acc.toolInput.WriteString(chunk)
			case "thinking_delta", "signature_delta":
				// Not surfaced to the coach — see the doc comment above.
			}

		case "message_delta":
			if delta, ok := event["delta"].(map[string]any); ok {
				stopReason, _ = delta["stop_reason"].(string)
			}
			readUsage(event["usage"], &usage)

		case "message_stop":
			sawStop = true

		case "error":
			errBody, _ := event["error"].(map[string]any)
			errType, _ := errBody["type"].(string)
			errMsg, _ := errBody["message"].(string)
			return Turn{}, classifyInStreamError(errType, errMsg)
		}
	}
	if err := scanner.Err(); err != nil {
		return Turn{}, classifyStreamReadErr(err)
	}

	if stopReason == "refusal" {
		return Turn{}, fmt.Errorf("%w: model declined to respond", ErrContentRefusal)
	}

	// A stream that just stops is not a finished turn — a dropped connection
	// looks exactly like this — so the half it delivered is never taken for
	// the whole (rv4-AIDEV-05). A stream with no content at all stays the
	// empty-response case below.
	if !sawStop && len(order) > 0 {
		return Turn{}, fmt.Errorf("%w: anthropic stream ended before message_stop", ErrProviderTransient)
	}

	var text strings.Builder
	var cutCall bool
	for _, index := range order {
		acc := blocks[index]
		switch acc.blockType {
		case "tool_use":
			if stopReason == "max_tokens" {
				cutCall = true // the cap cut this call's JSON short: it never runs
				continue
			}
			input := acc.toolInput.String()
			if input == "" {
				input = "{}"
			}
			return Turn{ToolCall: &ToolCallRequest{ID: acc.toolID, Name: acc.toolName, Arguments: input}, Usage: usage}, nil
		case "text":
			text.WriteString(acc.text.String())
		}
	}
	if cutCall {
		return Turn{Text: text.String(), Usage: usage, Truncated: true}, nil
	}
	if len(order) == 0 {
		return Turn{}, fmt.Errorf("agent: empty response from provider")
	}
	if text.Len() == 0 {
		return Turn{}, fmt.Errorf("%w: empty text response", ErrContentRefusal)
	}
	// stop_reason=max_tokens means the provider hit its output cap and cut
	// the answer off mid-thought. It's still a usable partial answer, but
	// flag it so the coach doesn't present it as a finished response (F13).
	return Turn{Text: text.String(), Usage: usage, Truncated: stopReason == "max_tokens"}, nil
}

// classifyInStreamError maps an SSE "error" event's error.type onto the same
// sentinels providerHTTPError uses for non-200 responses, so coach.go's
// taxonomy (retry / circuit / user-message policy) treats a mid-stream
// provider error identically to an HTTP-level one — instead of collapsing
// every in-stream error into a single opaque message that loses the type
// (F17). There's no retry here: the returned sentinel flows back through
// streamChat, whose single built-in retry already honors it. The codes are
// Anthropic's; the Responses streamer has its own classifier in responses.go.
func classifyInStreamError(errType, msg string) error {
	full := fmt.Sprintf("agent: anthropic stream error %s: %s", errType, msg)
	switch {
	case errType == "rate_limit_error":
		return &rateLimitError{msg: full} // no Retry-After in-stream; streamChat caps its own wait
	case errType == "overloaded_error" || errType == "api_error":
		return fmt.Errorf("%w: %s", ErrProviderTransient, full)
	case errType == "invalid_request_error" && strings.Contains(strings.ToLower(msg), "context"):
		return fmt.Errorf("%w: %s", ErrContextLength, full)
	case errType == "authentication_error" || errType == "permission_error" ||
		errType == "not_found_error" || errType == "invalid_request_error":
		return fmt.Errorf("%w: %s", ErrProviderConfig, full)
	default:
		return errors.New(full)
	}
}

// readUsage reads {"input_tokens": N, "output_tokens": N} into usage,
// keeping whichever fields are present — Anthropic splits usage across
// message_start (input+output) and message_delta (output only, some
// proxies repeat both), so later non-zero values simply overwrite
// earlier ones rather than needing to know which event carries what.
func readUsage(raw any, usage *Usage) {
	u, ok := raw.(map[string]any)
	if !ok {
		return
	}
	if v, ok := u["input_tokens"].(float64); ok {
		usage.InputTokens = int(v)
	}
	if v, ok := u["output_tokens"].(float64); ok {
		usage.OutputTokens = int(v)
	}
}
