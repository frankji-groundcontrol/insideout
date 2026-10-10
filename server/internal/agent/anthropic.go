package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	// idleTimeout bounds how long the stream may go silent before the
	// watchdog cancels it. Defaults to idleStreamTimeout; overridable so a
	// test can trip the watchdog in milliseconds. Zero means "use default".
	idleTimeout time.Duration
}

func NewAnthropicStreamer(baseURL, authToken, model string) (*AnthropicStreamer, error) {
	if authToken == "" {
		return nil, fmt.Errorf("agent: anthropic auth token is required")
	}
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
	return &AnthropicStreamer{
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		authToken:   authToken,
		model:       model,
		maxTokens:   4096,
		client:      &http.Client{Transport: transport}, // no overall Timeout: SSE streams legitimately run minutes; idleWatchdog covers stalls
		idleTimeout: idleStreamTimeout,
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
// the taxonomy in docs/plans/2026-07-21-prd-agent-harness/plan.md §5.2,
// but only while no delta has been emitted (a retry would replay text the
// client has already seen). Every other classified error (context length,
// config, refusal, context.Canceled) is never retried here. tool_choice
// forces the model to call exactly that tool when set (used by the critic
// — see critic.go).
func (a *AnthropicStreamer) StreamChat(ctx context.Context, system string, msgs []Message, tools []Tool, onDelta func(string)) (Turn, error) {
	return a.streamChat(ctx, system, msgs, tools, "", onDelta)
}

func (a *AnthropicStreamer) StreamChatForcingTool(ctx context.Context, system string, msgs []Message, tool Tool, onDelta func(string)) (Turn, error) {
	// AIDEV-09: the Messages API 400s on an empty "messages" array; a nil
	// payload marshals to exactly that (no omitempty). Fail here with a
	// named cause instead of burning a provider round trip.
	if len(msgs) == 0 {
		return Turn{}, fmt.Errorf("agent: forced-tool call to %q needs at least one message", tool.Name)
	}
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

	parent := ctx
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, LLMChatURL(a.baseURL, "messages"), bytes.NewReader(body))
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
		return Turn{}, providerHTTPError("anthropic", resp)
	}

	idle := a.idleTimeout
	if idle <= 0 {
		idle = idleStreamTimeout
	}
	tick, stop := idleWatchdog(cancel, idle)
	defer stop()

	turn, err := parseAnthropicStream(resp.Body, tick, onDelta)
	if err != nil && ctx.Err() != nil {
		// The derived ctx was canceled — that's the authoritative reason,
		// not whatever wrapped error the transport surfaced for it. The two
		// causes are told apart by the parent (request) ctx: if it's still
		// alive, the idle watchdog fired on a healthy connection — a hung
		// upstream (ErrUpstreamStall, a provider-health signal). If the
		// parent is done too, the client really disconnected — a routine
		// context.Canceled that coach.go treats as no-one's fault.
		if parent.Err() == nil {
			return Turn{}, ErrUpstreamStall
		}
		return Turn{}, parent.Err()
	}
	return turn, err
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
