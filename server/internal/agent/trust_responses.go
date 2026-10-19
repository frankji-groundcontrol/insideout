package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
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
	var calls respCalls
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
			if item, ok := ev["item"].(map[string]any); ok && item["type"] == "function_call" {
				calls.add(item)
			}
		case "response.function_call_arguments.delta":
			calls.delta(ev)
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
		// A read that died because the request was canceled (client gone, idle
		// watchdog) is not the provider's fault and must keep reading as a cancel.
		// Any other read failure is a dropped connection: transient, so a turn that
		// has not streamed yet is retried and the breaker counts it as load
		// (rv4-FR-F82). classifyStreamReadErr is the Anthropic twin, but it words the
		// error for that provider.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return Turn{}, err
		}
		return Turn{}, fmt.Errorf("%w: read %s stream: %s", ErrProviderTransient, responsesProvider, err.Error())
	}
	if !finished {
		return Turn{}, fmt.Errorf("%w: %s stream ended before response.completed", ErrProviderTransient, responsesProvider)
	}
	if call := calls.first(); call != nil {
		if truncated {
			// The cap cut the call's JSON short: it never runs, and the person
			// is told the answer was cut off instead.
			return Turn{Text: text.String(), Usage: usage, Truncated: true}, nil
		}
		args := call.args.String()
		if args == "" {
			args = "{}"
		}
		return Turn{ToolCall: &ToolCallRequest{ID: call.id, Name: call.name, Arguments: args}, Usage: usage}, nil
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

// respCall is one function_call item being streamed.
type respCall struct {
	id, name string
	args     strings.Builder
}

// respCalls keeps the function_call items of one response apart. Each is filed
// under the item id the provider stamps on every event of that item, so two
// calls can never be spliced into one (rv4-AIDEV-06).
type respCalls struct {
	byItem map[string]*respCall
	order  []string // item ids, in the order the provider announced them
	last   string   // the item announced most recently
}

// add starts a call from a response.output_item.added function_call item. A
// provider that sends no item id is keyed by its call id.
func (c *respCalls) add(item map[string]any) {
	key, _ := item["id"].(string)
	if key == "" {
		key, _ = item["call_id"].(string)
	}
	call := &respCall{}
	call.id, _ = item["call_id"].(string)
	call.name, _ = item["name"].(string)
	if args, ok := item["arguments"].(string); ok {
		call.args.WriteString(args)
	}
	if c.byItem == nil {
		c.byItem = map[string]*respCall{}
	}
	if _, seen := c.byItem[key]; !seen {
		c.order = append(c.order, key)
	}
	c.byItem[key] = call
	c.last = key
}

// delta appends an argument fragment to the call it names; a fragment that
// names none belongs to the call announced last.
func (c *respCalls) delta(ev map[string]any) {
	key, _ := ev["item_id"].(string)
	call := c.byItem[key]
	if call == nil {
		call = c.byItem[c.last]
	}
	if d, ok := ev["delta"].(string); ok && call != nil {
		call.args.WriteString(d)
	}
}

// first is the call to run: only one runs per turn (see ToolCallRequest), the
// first the provider announced. The rest are dropped, never merged into it.
func (c *respCalls) first() *respCall {
	for _, key := range c.order {
		if call := c.byItem[key]; call.name != "" {
			return call
		}
	}
	return nil
}
