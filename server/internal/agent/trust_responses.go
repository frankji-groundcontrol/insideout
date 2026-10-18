package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// parseResponsesStream reads a Responses API SSE stream into one Turn. A stream
// is a whole turn only once the provider says so: response.completed, or
// response.incomplete when it hit max_output_tokens. response.failed is a
// provider error, and a stream that just stops (a dropped connection, a bare
// [DONE]) is a transient failure, never a short answer (rv4-AIDEV-05).
func parseResponsesStream(r io.Reader, tick func(), onDelta func(string)) (Turn, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var text strings.Builder
	var toolID, toolName string
	var toolArgs strings.Builder
	var usage Usage
	var truncated, finished bool

	for scanner.Scan() {
		if tick != nil {
			tick()
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var ev map[string]any
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			continue
		}
		typ, _ := ev["type"].(string)
		switch typ {
		case "response.output_text.delta":
			if d, ok := ev["delta"].(string); ok && d != "" {
				text.WriteString(d)
				if onDelta != nil {
					onDelta(d)
				}
			}
		case "response.output_item.added":
			if item, ok := ev["item"].(map[string]any); ok {
				if item["type"] == "function_call" {
					if id, ok := item["call_id"].(string); ok {
						toolID = id
					}
					if name, ok := item["name"].(string); ok {
						toolName = name
					}
					if args, ok := item["arguments"].(string); ok {
						toolArgs.WriteString(args)
					}
				}
			}
		case "response.function_call_arguments.delta":
			if d, ok := ev["delta"].(string); ok {
				toolArgs.WriteString(d)
			}
		case "error":
			msg, _ := ev["message"].(string)
			code, _ := ev["code"].(string)
			return Turn{}, classifyResponsesStreamError(code, msg)
		case "response.failed":
			return Turn{}, classifyResponsesStreamError(responseFailure(ev))
		case "response.completed", "response.incomplete":
			finished = true
			// response.incomplete is the cap being hit; some providers report the
			// same thing as response.completed with status "incomplete".
			truncated = typ == "response.incomplete"
			if resp, ok := ev["response"].(map[string]any); ok {
				if u, ok := resp["usage"].(map[string]any); ok {
					if v, ok := u["input_tokens"].(float64); ok {
						usage.InputTokens = int(v)
					}
					if v, ok := u["output_tokens"].(float64); ok {
						usage.OutputTokens = int(v)
					}
				}
				if status, ok := resp["status"].(string); ok && status == "incomplete" {
					truncated = true
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return Turn{}, err
	}
	if !finished {
		return Turn{}, fmt.Errorf("%w: %s stream ended before response.completed", ErrProviderTransient, responsesProvider)
	}
	if toolName != "" {
		if truncated {
			// The cap cut the call's JSON short: it never runs, and the person
			// is told the answer was cut off instead.
			return Turn{Text: text.String(), Usage: usage, Truncated: true}, nil
		}
		args := toolArgs.String()
		if args == "" {
			args = "{}"
		}
		return Turn{ToolCall: &ToolCallRequest{ID: toolID, Name: toolName, Arguments: args}, Usage: usage}, nil
	}
	if text.Len() == 0 {
		return Turn{}, fmt.Errorf("%w: empty text response", ErrContentRefusal)
	}
	return Turn{Text: text.String(), Usage: usage, Truncated: truncated}, nil
}

// responseFailure reads the error code and message out of a response.failed
// event. A failure that names no code is the provider's own failure, so it is
// transient (server_error), not the configuration problem an unknown code is.
func responseFailure(ev map[string]any) (code, msg string) {
	if resp, ok := ev["response"].(map[string]any); ok {
		if e, ok := resp["error"].(map[string]any); ok {
			code, _ = e["code"].(string)
			msg, _ = e["message"].(string)
		}
	}
	if code == "" {
		code = "server_error"
	}
	return code, msg
}
