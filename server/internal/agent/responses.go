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

// responsesProvider names this API in error text, so a log line or
// ai_runs.error_message blames the upstream that answered.
const responsesProvider = "openai-compatible responses"

// ResponsesStreamer talks to an OpenAI-compatible Responses API at
// {base}/responses. The operator includes any /v1 on the base URL.
type ResponsesStreamer struct {
	baseURL     string
	authToken   string
	model       string
	client      *http.Client
	idleTimeout time.Duration
}

func NewResponsesStreamer(baseURL, authToken, model string) (*ResponsesStreamer, error) {
	if authToken == "" {
		return nil, fmt.Errorf("agent: llm api key is required")
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
	return &ResponsesStreamer{
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		authToken:   authToken,
		model:       model,
		client:      &http.Client{Transport: transport},
		idleTimeout: idleStreamTimeout,
	}, nil
}

func (s *ResponsesStreamer) Model() string { return s.model }

func (s *ResponsesStreamer) StreamChat(ctx context.Context, system string, msgs []Message, tools []Tool, onDelta func(string)) (Turn, error) {
	streamed := 0
	countDelta := func(d string) {
		if onDelta == nil {
			return
		}
		streamed++
		onDelta(d)
	}
	return retryStreamChat(ctx, func() bool { return streamed == 0 }, func(ctx context.Context) (Turn, error) {
		return s.doStream(ctx, system, msgs, tools, "", countDelta)
	})
}

func (s *ResponsesStreamer) StreamChatForcingTool(ctx context.Context, system string, msgs []Message, tool Tool, onDelta func(string)) (Turn, error) {
	streamed := 0
	countDelta := func(d string) {
		if onDelta == nil {
			return
		}
		streamed++
		onDelta(d)
	}
	return retryStreamChat(ctx, func() bool { return streamed == 0 }, func(ctx context.Context) (Turn, error) {
		return s.doStream(ctx, system, msgs, []Tool{tool}, tool.Name, countDelta)
	})
}

func (s *ResponsesStreamer) CheckModel(ctx context.Context) (ok bool, available []string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LLMModelsURL(s.baseURL), nil)
	if err != nil {
		return false, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.authToken)
	resp, err := s.client.Do(req)
	if err != nil {
		return false, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, nil, fmt.Errorf("agent: GET /models returned %s", resp.Status)
	}
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false, nil, err
	}
	for _, m := range body.Data {
		available = append(available, m.ID)
		if m.ID == s.model {
			ok = true
		}
	}
	return ok, available, nil
}

type responsesRequest struct {
	Model        string           `json:"model"`
	Instructions string           `json:"instructions,omitempty"`
	Input        []map[string]any `json:"input"`
	Tools        []map[string]any `json:"tools,omitempty"`
	ToolChoice   any              `json:"tool_choice,omitempty"`
	Stream       bool             `json:"stream"`
	// MaxOutputTokens caps the answer's length — without it a Responses
	// endpoint defaults to the model's own (for reasoning models very
	// large) ceiling, and hitting our cap is what makes the final status
	// "incomplete", which parseResponsesStream maps to Turn.Truncated (F13).
	MaxOutputTokens int `json:"max_output_tokens,omitempty"`
}

func (s *ResponsesStreamer) doStream(ctx context.Context, system string, msgs []Message, tools []Tool, forceTool string, onDelta func(string)) (Turn, error) {
	reqBody := responsesRequest{
		Model:           s.model,
		Instructions:    system,
		Input:           toResponsesInput(msgs),
		Tools:           toResponsesTools(tools),
		Stream:          true,
		MaxOutputTokens: 4096, // matches the Anthropic streamer's maxTokens (anthropic.go)
	}
	if forceTool != "" {
		reqBody.ToolChoice = map[string]any{"type": "function", "name": forceTool}
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return Turn{}, fmt.Errorf("agent: marshal responses request: %w", err)
	}

	parent := ctx
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, LLMChatURL(s.baseURL, "responses"), bytes.NewReader(body))
	if err != nil {
		return Turn{}, fmt.Errorf("agent: build responses request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+s.authToken)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return Turn{}, ctx.Err()
		}
		return Turn{}, fmt.Errorf("%w: %s", ErrProviderTransient, err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Turn{}, providerHTTPError(responsesProvider, resp)
	}

	idle := s.idleTimeout
	if idle <= 0 {
		idle = idleStreamTimeout
	}
	tick, stop := idleWatchdog(cancel, idle)
	defer stop()

	turn, err := parseResponsesStream(resp.Body, tick, onDelta)
	if err != nil && ctx.Err() != nil {
		if parent.Err() == nil {
			return Turn{}, ErrUpstreamStall
		}
		return Turn{}, parent.Err()
	}
	return turn, err
}

func toResponsesInput(msgs []Message) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case RoleUser:
			out = append(out, map[string]any{"role": "user", "content": m.Content})
		case RoleAssistant:
			if m.ToolCall != nil {
				args := m.ToolCall.Arguments
				if args == "" {
					args = "{}"
				}
				out = append(out, map[string]any{
					"type": "function_call", "call_id": m.ToolCall.ID, "name": m.ToolCall.Name, "arguments": args,
				})
			} else {
				out = append(out, map[string]any{"role": "assistant", "content": m.Content})
			}
		case RoleTool:
			out = append(out, map[string]any{
				"type": "function_call_output", "call_id": m.ToolCallID, "output": m.Content,
			})
		}
	}
	return out
}

func toResponsesTools(tools []Tool) []map[string]any {
	if len(tools) == 0 {
		return nil
	}
	out := make([]map[string]any, len(tools))
	for i, t := range tools {
		out[i] = map[string]any{
			"type": "function", "name": t.Name, "description": t.Description, "parameters": t.Parameters,
		}
	}
	return out
}

// classifyResponsesStreamError maps an in-stream "error" event's code onto the
// sentinels classifyInStreamError uses for Anthropic's vocabulary, so retry and
// the circuit breaker treat a Responses provider like an Anthropic one. Only
// the load codes are mapped: auth, model and context failures are normally
// refused with an HTTP status before the stream opens (providerHTTPError), and
// any other code is a configuration problem on our side of the wire, not
// load — leaving it unclassified fed the global breaker (AIDEV-03).
func classifyResponsesStreamError(code, msg string) error {
	full := fmt.Sprintf("agent: %s stream error %s: %s", responsesProvider, code, msg)
	switch code {
	case "rate_limit_exceeded":
		return &rateLimitError{msg: full} // no Retry-After in-stream; retryStreamChat caps its own wait
	case "server_error":
		return fmt.Errorf("%w: %s", ErrProviderTransient, full)
	default:
		return fmt.Errorf("%w: %s", ErrProviderConfig, full)
	}
}
