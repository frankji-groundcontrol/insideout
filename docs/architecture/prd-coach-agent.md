# PRD Coach agent

`server/internal/agent` implements a four-stage coaching conversation that
guides a user from a raw idea to a structured PRD, writing sections via tool
calls and streaming replies over Server-Sent Events.

## Provider

`internal/agent/anthropic.go` talks directly to the Anthropic Messages API
over stdlib `net/http`, with a hand-rolled SSE parser in `anthropic_parse.go` —
no LLM framework dependency. This replaced an earlier `langchaingo`-based implementation; see
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
`INSIDEOUT_LLM_API_KEY` is unset — this is also local dev mode. It echoes the
member's last message and makes no tool calls, so no stage or section moves;
the SSE `message_start` frame carries `offline: true` and the client banners it.
`INSIDEOUT_LLM_SCHEMA=messages` (default) posts Anthropic Messages JSON to
`{base}/messages`. `INSIDEOUT_LLM_SCHEMA=responses` posts OpenAI Responses
JSON to `{base}/responses`. The operator includes any `/v1` on
`INSIDEOUT_LLM_BASE_URL`.

### Anthropic wire format

`anthropic.go` builds the request JSON (`model`, `max_tokens`, `system` as a
top-level field, `tools`, `messages`) and `anthropic_parse.go` parses the
streaming response event-by-event: `message_start` → `content_block_start` (`text` / `tool_use`
/ `thinking`) → `content_block_delta` (`text_delta` / `input_json_delta` /
`thinking_delta` / `signature_delta`) → `content_block_stop` →
`message_delta` → `message_stop`. Content blocks are accumulated **by
index**, not assumed to be one-per-response: an extended-thinking model
returns a `thinking` block before the real `text`/`tool_use` block, and the
first content block that carries a tool call wins; otherwise all `text`
blocks are concatenated. `internal/agent/anthropic_test.go` uses real SSE
payloads captured from a live endpoint as test fixtures for exactly this
case.

A turn is whole only when its stream ends the way the provider says it does
(rv4-AIDEV-05). An Anthropic stream must reach `message_stop`; a Responses
stream (parsed in `trust_responses.go`) must reach `response.completed` or
`response.incomplete`. One that ends earlier is `ErrProviderTransient`, retried
once when nothing had been streamed yet. A tool call cut by the token cap
(`max_tokens` / `incomplete`) is never run: the turn is marked truncated and
the stored answer ends with the truncation marker. `response.failed` and
`error` frames are read as provider errors: a server error is retried, an
`invalid_prompt`-class one is `PROVIDER_CONFIG`, and neither is a content
refusal. Responses requests that carry tools send `parallel_tool_calls: false`,
and each call's arguments are filed under the provider's item id, so two calls
are never merged into one: the first announced call runs and later ones are
dropped (rv4-AIDEV-06). The cost is that a gateway which never sends the
closing event now fails every turn; the one-condition escape hatches are in the
[w3-coach-trust record](../changelogs/2026-10-18-rv4-remediation/w3-coach-trust.md).

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

### Confirming, declining and parked resolutions

A stage change the model proposes waits as `pendingStage` on the conversation
until the person answers. Finding resolutions the model reports meanwhile
(`resolve_finding`) are parked as `pendingFindingResolutions` and applied only
when the stage is confirmed (`POST /api/v1/conversations/{id}/stage`). The
conversation view lists them as `pendingResolutions` (id, section, severity,
issue, status) using the same filter confirm applies, so the card never shows a
resolution confirm would skip (rv4-AIDEV-03). `POST
/api/v1/conversations/{id}/stage/decline` (human-only, see
[agent token scope](agent-token-scope.md)) removes both while a stage is
pending: the stage and the findings stay as they were, a second decline or a
confirm after a decline is a 409, and a decline is no longer something only the
client remembers (rv4-CLOSE-03). A second tab keeps its card until it refreshes,
and its confirm then meets the 409.

## Tools

`internal/agent/tools.go` defines six tools and executes them against the
store: `get_prd()`, `update_prd_section(section, markdown)` (validated
against the 8 fixed section keys, emits a `prd_updated` SSE event),
`advance_stage(next)` (validated against the legal transition order), and the
trust-layer tools `record_fact(kind, text, quote)`, `mark_assumption(text)` and
`resolve_finding(id, status)` (the status change is parked until the person
confirms the stage change). Only one tool call per assistant turn is ever
surfaced: both parsers (`parseAnthropicStream` in `anthropic_parse.go`, and the
Responses parser) run the first tool call they find and drop later ones — the
coach's own prompting discipline never asks for more than one anyway.

`record_fact` attests a fact only on the person's own words (rv4-AIDEV-07,
`trust_facts.go`): the quote must appear verbatim in what they said, must hold
at least 12 characters of content once spaces and punctuation are dropped (a
Han, kana or hangul character counts twice), and the fact's text must share at
least 0.2 of its character n-grams with the quote. Both numbers are tunables
chosen on a hand-written calibration set and pinned by
`trust_facts_bounds_test.go`; support is lexical, so it catches text about
something else, not a restatement that merely sounds like the quote. The stored
status stays `attested`; the interface and the gate copy say "recorded fact".

## Streaming contract

`POST /api/v1/conversations/{id}/messages` responds `text/event-stream`:

```
event: message_start   data: {"id": "<assistant message uuid>"}
event: delta            data: {"text": "..."}
event: prd_updated      data: {"section": "goals"}
event: stage_changed    data: {"stage": "draft"}
event: critic_skipped   data: {"reason": "contention", "rounds": 0}
event: message_end      data: {"id": "...", "tokens": 812}
event: error            data: {"error": "...", "code": "..."}
```

`critic_skipped` is sent before `message_end` when a critic pass did not run
(`reason` is `contention` or `critic-output-failure`), so a skipped review is
never silent (rv4-AIDEV-02). It is announced once per run of skips; the marker
clears when a round runs. The conversation view then carries
`criticSkipped` and `criticRounds`, and the finalize card says no critic review
ran. The finalize gate still counts a skip as a pass, on purpose (a busy or
misconfigured critic must not hold a PRD hostage); whether a skip should refuse
the gate is an open owner question, one condition in `tools.go` plus its pinning
test, written up in the w3-coach-trust record under "The gate decision".

Rate-limit and circuit-breaker checks (ported from the original Postgres
functions, `server/db/migrations/20260720135755_ai_ops.sql`) run **before**
the stream opens, so throttling still returns plain JSON 429/503. Every user
message creates an `ai_runs` row (the limiter's counting source); a provider
429 marks the run `failed` rather than leaving it stranded `pending`.

Full history is stored server-side in `agent_messages` (not client-side
`localStorage`) and is retrievable via `GET
/api/v1/conversations/{id}/messages` — verified end-to-end with a real
coaching exchange (a real streamed reply, then confirmed the exchange is
persisted and returned by that endpoint). A turn does not replay all of it:
`coach_loop.go` (the provider/tool loop; `coach.go` owns the turn lifecycle)
loads only the newest messages, and the store applies that limit in SQL.

An SSE-specific bug worth knowing about if a new streaming endpoint is ever
added: response-writer-wrapping middleware must explicitly forward
`http.Flusher`, or streaming breaks silently — see
[BUG-010](../issues/2026-07-21-bug-010-sse-flusher-swallowed-by-logging-middleware.md).
