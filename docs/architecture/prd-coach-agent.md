# PRD Coach agent

`server/internal/agent` implements a four-stage coaching conversation that
guides a user from a raw idea to a structured PRD, writing sections via tool
calls and streaming replies over Server-Sent Events.

## Provider

`internal/agent/anthropic.go` talks directly to the Anthropic Messages API
over stdlib `net/http`, with a hand-rolled SSE parser — no LLM framework
dependency. This replaced an earlier `langchaingo`-based implementation; see
[BUG-009](../issues/2026-07-21-bug-009-langchaingo-removed.md) for why (the library is
unmaintained, and it silently dropped real answers behind extended-thinking
content blocks). The provider is hidden behind one interface,
`ChatStreamer` (`internal/agent/llm.go`):

```go
type ChatStreamer interface {
    StreamChat(ctx context.Context, system string, msgs []Message, tools []Tool, onDelta func(string)) (Turn, error)
}
```

Swapping providers later is a one-file change. `internal/agent/template.go`
provides an offline, no-network fallback `ChatStreamer` used when
`ANTHROPIC_AUTH_TOKEN` is unset — this is also local dev mode.

### Anthropic wire format

`anthropic.go` builds the request JSON (`model`, `max_tokens`, `system` as a
top-level field, `tools`, `messages`) and parses the streaming response
event-by-event: `message_start` → `content_block_start` (`text` / `tool_use`
/ `thinking`) → `content_block_delta` (`text_delta` / `input_json_delta` /
`thinking_delta` / `signature_delta`) → `content_block_stop` →
`message_delta` → `message_stop`. Content blocks are accumulated **by
index**, not assumed to be one-per-response: an extended-thinking model
returns a `thinking` block before the real `text`/`tool_use` block, and the
first content block that carries a tool call wins; otherwise all `text`
blocks are concatenated. `internal/agent/anthropic_test.go` uses real SSE
payloads captured from a live endpoint as test fixtures for exactly this
case.

Tool results are sent back the way Anthropic expects: a `user`-role message
with a `tool_result` content block — Anthropic has no separate "tool" role.

A non-2xx HTTP response surfaces the provider's actual error message (e.g.
`model_not_found: Model "..." is not supported...`); a 429 maps to
`ErrProviderRateLimited`, which the coach emits as an SSE `error` event with
code `ANTHROPIC_RATE_LIMIT` (the stream is already open by then — see
[backend](backend.md)).

## Stage state machine

One agent, four stages, driven by `agent_conversations.stage` — not a
multi-agent system. `internal/agent/prompts.go` swaps the system prompt per
stage; the model advances the stage itself via the `advance_stage` tool call
when its exit criteria are met.

| Stage | Behavior | Exit |
|---|---|---|
| `clarify` | Interview the user — one focused question at a time. Never drafts yet. | Problem, users, value stated concretely |
| `draft` | Writes every section via `update_prd_section` tool calls. | All 8 sections non-empty |
| `critique` | Scores each section empty/thin/solid, targets the weakest two. | No section below "solid", or user override |
| `finalize` | Summary + completeness checklist, suggests a revision snapshot + submitting for review. | Conversation marked `completed` |

## Tools

`internal/agent/tools.go` defines three tools and executes them against the
store: `get_prd()`, `update_prd_section(section, markdown)` (validated
against the 8 fixed section keys, emits a `prd_updated` SSE event), and
`advance_stage(next)` (validated against the legal transition order). Only
one tool call per assistant turn is ever surfaced (`parseAnthropicStream` in
`anthropic.go` returns on the first tool_use content block it finds) — the
coach's own prompting discipline never asks for more than one anyway.

## Streaming contract

`POST /api/v1/conversations/{id}/messages` responds `text/event-stream`:

```
event: message_start   data: {"id": "<assistant message uuid>"}
event: delta            data: {"text": "..."}
event: prd_updated      data: {"section": "goals"}
event: stage_changed    data: {"stage": "draft"}
event: message_end      data: {"id": "...", "tokens": 812}
event: error            data: {"error": "...", "code": "..."}
```

Rate-limit and circuit-breaker checks (ported from the original Postgres
functions, `server/db/migrations/20260720135755_ai_ops.sql`) run **before**
the stream opens, so throttling still returns plain JSON 429/503. Every user
message creates an `ai_runs` row (the limiter's counting source); a provider
429 marks the run `failed` rather than leaving it stranded `pending`.

Full history is stored server-side in `agent_messages` (not client-side
`localStorage`) and is retrievable via `GET
/api/v1/conversations/{id}/messages` — verified end-to-end with a real
coaching exchange (a real streamed reply, then confirmed the exchange is
persisted and returned by that endpoint).

An SSE-specific bug worth knowing about if a new streaming endpoint is ever
added: response-writer-wrapping middleware must explicitly forward
`http.Flusher`, or streaming breaks silently — see
[BUG-010](../issues/2026-07-21-bug-010-sse-flusher-swallowed-by-logging-middleware.md).
