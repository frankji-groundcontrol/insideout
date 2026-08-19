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
	return retryStreamChat(ctx, func(ctx context.Context) (Turn, error) {
		return s.doStream(ctx, system, msgs, tools, "", onDelta)
	})
}

func (s *ResponsesStreamer) StreamChatForcingTool(ctx context.Context, system string, msgs []Message, tool Tool, onDelta func(string)) (Turn, error) {
	return retryStreamChat(ctx, func(ctx context.Context) (Turn, error) {
		return s.doStream(ctx, system, msgs, []Tool{tool}, tool.Name, onDelta)
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
}

func (s *ResponsesStreamer) doStream(ctx context.Context, system string, msgs []Message, tools []Tool, forceTool string, onDelta func(string)) (Turn, error) {
	reqBody := responsesRequest{
		Model:        s.model,
		Instructions: system,
		Input:        toResponsesInput(msgs),
		Tools:        toResponsesTools(tools),
		Stream:       true,
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
		return Turn{}, anthropicHTTPError(resp)
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

func parseResponsesStream(r io.Reader, tick func(), onDelta func(string)) (Turn, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var text strings.Builder
	var toolID, toolName string
	var toolArgs strings.Builder
	var usage Usage
	var truncated bool

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
			return Turn{}, classifyInStreamError(code, msg)
		case "response.completed":
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
	if toolName != "" {
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
