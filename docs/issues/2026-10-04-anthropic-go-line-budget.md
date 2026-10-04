# 2026-10-04 — `server/internal/agent/anthropic.go` over the ~350-line budget

`anthropic.go` is 408 lines and mixes three responsibilities: the wire
types (request/message/tool/error structs), the request path
(`StreamChat`, `StreamChatForcingTool`, `doStreamChat` — HTTP, headers,
the two schemas), and the response parsing (`parseAnthropicStream`,
`contentBlockAcc`, `classifyInStreamError`). It grew past the budget with
the responses-schema support; the 2026-10-04 remediation waves left it
untouched.

## Why not split now

No behavioral change in the waves required touching it, and the parser is
the file's most subtle part (SSE framing, content-block accumulation,
in-stream error classification) — moving it deserves its own verified
diff, not a ride-along.

## Target structure

- `anthropic.go` — wire types + request path (`StreamChat*`,
  `doStreamChat`) (~180 lines).
- `anthropic_parse.go` — `parseAnthropicStream`, `contentBlockAcc`,
  `classifyInStreamError` (~230 lines).

## Bounded fix prompt

"Move the response-side parsing (`parseAnthropicStream`,
`contentBlockAcc`, `classifyInStreamError`) from
`server/internal/agent/anthropic.go` into `anthropic_parse.go` in the
same package — no signature changes, per
[docs/issues/2026-10-04-anthropic-go-line-budget.md](2026-10-04-anthropic-go-line-budget.md).
Gate: `go build ./... && go vet ./... && go test ./...` green — the
stream-parse and error-classification table tests are the behavioral
pins. Both files end under ~350 lines. Record the split in a changelog
entry and close this issue."
