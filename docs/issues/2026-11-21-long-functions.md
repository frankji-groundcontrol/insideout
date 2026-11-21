# 2026-11-21 — sixteen server functions over the 80-line function budget

> **Resolved 2026-11-21** in the round that closed the review 5 gaps: every
> listed function is split, and the guard's `longFunctions` map is empty. A new
> function over the threshold now fails the guard outright.

**Date:** 2026-11-21
**Kind:** structural debt (function length), no behaviour change
**Found by:** review 5 ([COMP-05](../review/2026-11-21-focused-review-5/30-completeness-critic.md),
row 4.1 of the [remediation plan](../review/2026-11-21-focused-review-5/40-remediation-plan.md)),
when the line rule was replaced by a responsibility rule with a function-length guard.
**Guard:** `server/internal/linebudget/func_budget_test.go`, threshold
`functionLines` in [config/code-budget.json](../../config/code-budget.json).

## What is over

*2026-11-21, wave 4: `Coach.HandleMessage`, `runLocalSetup` and `parseAnthropicStream` are split and out of the guard's list; `parseResponsesStream` moved to `internal/llm`. Thirteen remain. Wave 3, built beside the guard, grew `addAuthTools` to 156 lines and `runClientCommand` (`cmd/insideout/cli.go`) to 92: both are listed at those lengths and assigned to the wave 4 MCP and CLI splits.*

*Wave 4 CLI split: `runClientCommand` (now `cmd/insideout/cli_dispatch.go`), `runCrudCommand`, `runVersionCommands` and `runServe` are split and out of the guard's list.*

Length is counted from the `func` line to the closing brace, at `87c95f5`:

| Function | File (under `server/`) | Lines |
|---|---|---|
| ~~`addProductTools`~~ | `internal/mcpserver/product_tools.go` | 294: split into per-resource `tools_*.go` in wave 4 |
| ~~`Coach.HandleMessage`~~ | `internal/agent/coach.go` | 219: split in wave 4 |
| ~~`runLocalSetup`~~ | `cmd/insideout/cli_setup_local.go` | 160: split in wave 4 |
| ~~`addAuthTools`~~ | `internal/mcpserver/auth_tools.go` | 152: split in `tools_auth.go` in wave 4 |
| ~~`parseAnthropicStream`~~ | `internal/agent/anthropic_parse.go` | 136: split in wave 4 |
| ~~`runVersionCommands`~~ | `cmd/insideout/cli_version.go` | 117: split per verb in wave 4 (`cli_prd.go`, `cli_prd_versions.go`, `cli_agent.go`) |
| ~~`runCrudCommand`~~ | `cmd/insideout/cli_crud.go` | 103: split per verb in wave 4 (`cli_idea.go`, `cli_workspace.go`, `cli_prd.go`) |
| ~~`Store.Migrate`~~ | `internal/store/migrate.go` | 100: split into bootstrap, list and apply steps in the closing-gaps round |
| ~~`Store.ReplaceRoadmapTree`~~ | `internal/store/roadmap_plan.go` | 99: split into guard, insert and seed steps in the closing-gaps round |
| ~~`parseResponsesStream`~~ | `internal/llm/responses_parse.go` (moved by wave 4) | 98: split into a per-event handler in the closing-gaps round |
| ~~`Server.handleUpdateRoadmapNode`~~ | `internal/api/roadmap.go` | 97: split in the gaps round (`roadmapNodeFields`) |
| ~~`Store.MoveRoadmapNode`~~ | `internal/store/roadmap_move.go` | 94: split into authorize, cycle-check and sibling-shift steps in the closing-gaps round |
| ~~`Server.handleAgentContext`~~ | `internal/api/agent_routes.go` | 89: split in the gaps round (`agentContextQuery`, `agentContextNodes`, `addAgentContextPrd`) |
| ~~`runServe`~~ | `cmd/insideout/main.go` | 87: split in wave 4 (`newModel`) |
| ~~`Server.handlePresenceStream`~~ | `internal/api/presence_routes.go` | 85: split in the gaps round (`streamPresence`, `presenceName`) |
| ~~`runRegister`~~ | `internal/setup/ui/register.go` | 84: submit step split out in the closing-gaps round |

Each is listed in `longFunctions` with its current length. A listed function
may not grow; an entry whose function is gone or back under the limit fails,
so the list only shrinks.

## Why it matters

A function this long does several steps of work in one body: the MCP tool
registrars hold one closure per tool, the stream parsers one branch per event
type, the coach turn the whole load-call-persist sequence. Each step is hard to
test alone and every change to one step re-reads all of them.

## Target structure

- Registrars (`addProductTools`, `addAuthTools`): one small `addXTool`
  function per tool, the registrar a list of calls.
- Stream parsers: one handler per event type behind the read loop.
- Command dispatchers (`runCrudCommand`, `runVersionCommands`, `runServe`,
  `runLocalSetup`, `runRegister`): one function per verb or setup step.
- Store and handler functions: lift validation, the transaction body and the
  response shaping into named helpers.

## Bounded fix prompt

Pick one function from the table. Split it by step into named functions in the
same package (a new file if the current one would pass the file budget), with
no behaviour change: the package's tests stay green before and after, and the
commit says so. Delete its `longFunctions` entry in the same commit (the guard
fails if the entry outlives the split) and strike its row here. Close this
record when the table is empty.
