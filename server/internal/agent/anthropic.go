package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// AnthropicStreamer implements ChatStreamer directly against the
// Anthropic Messages API (stdlib net/http + a hand-rolled SSE parser) —
// no LLM library. langchaingo (previously used here, pinned per D6 in
// docs/plans/2026-07-20-go-rewrite/README.md) is an unmaintained project;
// on top of its already-documented bugs (parallel tool calls dropped —
// see ToolCallRequest), it also maps every response content block to a
// separate Choices[i] rather than accumulating them into one message, so
// an extended-thinking model's leading "thinking" block silently ate the
// real answer at Choices[0] and turned every real request into "no
// response". The Messages API itself is a small, stable, well-documented
// surface, so implementing it directly removes both problems at once.
// Retry/error classification lives in taxonomy.go.

type AnthropicStreamer struct {
	baseURL   string
	authToken string
	model     string
	maxTokens int
	client    *http.Client
}

func NewAnthropicStreamer(baseURL, authToken, model string) (*AnthropicStreamer, error) {
	if authToken == "" {
		return nil, fmt.Errorf("agent: anthropic auth token is required")
	}
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
	return &AnthropicStreamer{
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		authToken: authToken,
		model:     model,
		maxTokens: 4096,
		client:    &http.Client{Transport: transport}, // no overall Timeout: SSE streams legitimately run minutes; idleWatchdog covers stalls
	}, nil
}

// --- wire format: only the fields we actually send/read. ---

type anthropicContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
}

type anthropicMessage struct {
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type anthropicToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

type anthropicRequest struct {
	Model      string               `json:"model"`
	MaxTokens  int                  `json:"max_tokens"`
	Stream     bool                 `json:"stream"`
	System     string               `json:"system,omitempty"`
	Tools      []anthropicTool      `json:"tools,omitempty"`
	ToolChoice *anthropicToolChoice `json:"tool_choice,omitempty"`
	Messages   []anthropicMessage   `json:"messages"`
}

type anthropicErrorBody struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// StreamChat retries transient failures once — 429 (honoring Retry-After,
// capped) and 5xx/connect refusal (fixed short backoff) — with jitter, per
// the taxonomy in docs/plans/2026-07-21-prd-agent-harness/plan.md §5.2.
// Every other classified error (context length, config, refusal,
// context.Canceled) is never retried here. tool_choice forces the model
// to call exactly that tool when set (used by the critic — see critic.go).
func (a *AnthropicStreamer) StreamChat(ctx context.Context, system string, msgs []Message, tools []Tool, onDelta func(string)) (Turn, error) {
	return a.streamChat(ctx, system, msgs, tools, "", onDelta)
}

func (a *AnthropicStreamer) StreamChatForcingTool(ctx context.Context, system string, msgs []Message, tool Tool, onDelta func(string)) (Turn, error) {
	return a.streamChat(ctx, system, msgs, []Tool{tool}, tool.Name, onDelta)
}

func (a *AnthropicStreamer) Model() string { return a.model }

func (a *AnthropicStreamer) doStreamChat(ctx context.Context, system string, msgs []Message, tools []Tool, forceTool string, onDelta func(string)) (Turn, error) {
	req := anthropicRequest{
		Model:     a.model,
		MaxTokens: a.maxTokens,
		Stream:    true,
		System:    system,
		Tools:     toAnthropicTools(tools),
		Messages:  toAnthropicMessages(msgs),
	}
	if forceTool != "" {
		req.ToolChoice = &anthropicToolChoice{Type: "tool", Name: forceTool}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return Turn{}, fmt.Errorf("agent: marshal anthropic request: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return Turn{}, fmt.Errorf("agent: build anthropic request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("x-api-key", a.authToken)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return Turn{}, ctx.Err()
		}
		return Turn{}, fmt.Errorf("%w: %s", ErrProviderTransient, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Turn{}, anthropicHTTPError(resp)
	}

	tick, stop := idleWatchdog(cancel, idleStreamTimeout)
	defer stop()

	turn, err := parseAnthropicStream(resp.Body, tick, onDelta)
	if err != nil && ctx.Err() != nil {
		// The derived ctx was canceled (parent canceled, or the idle
		// watchdog fired) — that's the authoritative reason, not
		// whatever wrapped error the transport surfaced for it.
		return Turn{}, ctx.Err()
	}
	return turn, err
}

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

		case "error":
			errBody, _ := event["error"].(map[string]any)
			return Turn{}, fmt.Errorf("agent: anthropic stream error: %v", errBody)
		}
	}
	if err := scanner.Err(); err != nil {
		return Turn{}, classifyStreamReadErr(err)
	}

	if stopReason == "refusal" {
		return Turn{}, fmt.Errorf("%w: model declined to respond", ErrContentRefusal)
	}

	var text strings.Builder
	for _, index := range order {
		acc := blocks[index]
		switch acc.blockType {
		case "tool_use":
			input := acc.toolInput.String()
			if input == "" {
				input = "{}"
			}
			return Turn{ToolCall: &ToolCallRequest{ID: acc.toolID, Name: acc.toolName, Arguments: input}, Usage: usage}, nil
		case "text":
			text.WriteString(acc.text.String())
		}
	}
	if len(order) == 0 {
		return Turn{}, fmt.Errorf("agent: empty response from provider")
	}
	if text.Len() == 0 {
		return Turn{}, fmt.Errorf("%w: empty text response", ErrContentRefusal)
	}
	return Turn{Text: text.String(), Usage: usage}, nil
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

func toAnthropicTools(tools []Tool) []anthropicTool {
	out := make([]anthropicTool, len(tools))
	for i, t := range tools {
		out[i] = anthropicTool{Name: t.Name, Description: t.Description, InputSchema: t.Parameters}
	}
	return out
}

// toAnthropicMessages converts our provider-agnostic Message slice into
// Anthropic's wire format. A tool result (RoleTool) is a "user"-role
// message containing a tool_result block — Anthropic has no separate
// "tool" role (that's an OpenAI convention).
func toAnthropicMessages(msgs []Message) []anthropicMessage {
	out := make([]anthropicMessage, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case RoleUser:
			out = append(out, anthropicMessage{Role: "user", Content: []anthropicContentBlock{{Type: "text", Text: m.Content}}})
		case RoleAssistant:
			if m.ToolCall != nil {
				input := json.RawMessage(m.ToolCall.Arguments)
				if len(input) == 0 {
					input = json.RawMessage("{}")
				}
				out = append(out, anthropicMessage{Role: "assistant", Content: []anthropicContentBlock{{
					Type: "tool_use", ID: m.ToolCall.ID, Name: m.ToolCall.Name, Input: input,
				}}})
			} else {
				out = append(out, anthropicMessage{Role: "assistant", Content: []anthropicContentBlock{{Type: "text", Text: m.Content}}})
			}
		case RoleTool:
			out = append(out, anthropicMessage{Role: "user", Content: []anthropicContentBlock{{
				Type: "tool_result", ToolUseID: m.ToolCallID, Content: m.Content,
			}}})
		}
	}
	return out
}
