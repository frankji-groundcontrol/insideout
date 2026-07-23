// Package agent implements the PRD Coach: a four-stage coaching
// conversation that guides a user from a raw idea to a structured PRD,
// writing sections via tool calls and streaming replies over SSE. See
// docs/plans/2026-07-20-go-rewrite/03-agents.md.
package agent

import (
	"context"
	"errors"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolCallRequest is the model asking to invoke one tool. Only one tool
// call per assistant turn is supported — see §5 of 03-agents.md: the
// coach prompts for exactly one tool call per turn anyway (see
// prompts.go), so anthropic.go only ever surfaces the first tool_use
// block in a response.
type ToolCallRequest struct {
	ID        string
	Name      string
	Arguments string // raw JSON object
}

// Message is one turn in a conversation, in our own provider-agnostic
// shape (never the wire format of any specific provider) so only
// anthropic.go knows about Anthropic's Messages API.
type Message struct {
	Role       Role
	Content    string
	ToolCall   *ToolCallRequest // set on an assistant message that is a tool call
	ToolCallID string           // set on a tool-role message: which call this answers
}

// Tool is a tool definition offered to the model, JSON-Schema parameters.
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// Turn is what the model produced: either final text, or a request to
// call one tool.
type Turn struct {
	Text     string
	ToolCall *ToolCallRequest
	Usage    Usage
}

// Usage is token accounting for one provider call — metadata only, never
// the prompt/response content itself (plan §10.6 privacy constraint on
// telemetry).
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// Sentinel errors a ChatStreamer implementation classifies its failures
// into, so coach.go can decide retry/circuit/user-message policy without
// knowing provider-specific status codes (docs/plans/2026-07-21-prd-agent-harness/plan.md §5.2).
// A ChatStreamer is expected to retry transient conditions (rate limit,
// 5xx) internally once before returning; these sentinels represent the
// final, exhausted-retry outcome.
var (
	// ErrProviderRateLimited: 429, retries exhausted. Circuit failure.
	ErrProviderRateLimited = errors.New("agent: provider rate limited")
	// ErrProviderTransient: 5xx/connect refusal, retries exhausted. Circuit failure.
	ErrProviderTransient = errors.New("agent: provider transient error")
	// ErrContextLength: 400 context-length-exceeded. Not retried by the
	// streamer; coach.go gets one auto-tighten attempt. Not a circuit failure.
	ErrContextLength = errors.New("agent: context length exceeded")
	// ErrProviderConfig: 400 other/401/403/model_not_found — a
	// configuration problem, not provider load. Not a circuit failure
	// (would poison the breaker for every user over one bad config).
	ErrProviderConfig = errors.New("agent: provider configuration error")
	// ErrContentRefusal: the model refused or returned empty content.
	// Not a circuit failure.
	ErrContentRefusal = errors.New("agent: content refused")
)

// ChatStreamer sends one turn to a model: system prompt, message history,
// available tools, and a callback for text deltas as they stream in. This
// is the ONLY interface the coaching logic depends on — swapping the
// Anthropic direct client (anthropic.go) for another provider later is a
// one-file change (see docs/plans/2026-07-20-go-rewrite/03-agents.md §5).
type ChatStreamer interface {
	StreamChat(ctx context.Context, system string, msgs []Message, tools []Tool, onDelta func(string)) (Turn, error)

	// StreamChatForcingTool is like StreamChat but forces the model to
	// call exactly the given tool — used by the critic (critic.go) to
	// get schema-validated structured output instead of scraping JSON
	// out of freeform text (plan §4.4).
	StreamChatForcingTool(ctx context.Context, system string, msgs []Message, tool Tool, onDelta func(string)) (Turn, error)

	// Model identifies which model this streamer talks to, for telemetry.
	Model() string
}
