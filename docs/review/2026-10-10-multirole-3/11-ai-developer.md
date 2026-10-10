# 11 · AI Developer

**Score:** **7.5 / 10** · 12 findings (2 High+)

**Bottom line:** The agent core is unusually well-girded for its age — mechanical stage gates, server-side quote grounding, prompt fencing with tag-breaking, a real error taxonomy, and terminal-state guarantees on every turn — but two of its load-bearing guarantees are still model-defeatable: a critique-stage reply wholesale-replaces the stored critic findings that the finalize gate reads, and one roadmap path (ExpandNode) ships user-writable text to the model completely unfenced. Everything else I filed is margin work: taxonomy fall-throughs that poison the global breaker, a Responses provider with no output cap, an MCP commit tool that reads an argument its own schema never declares, and a prompt contract that still encodes the fixed-question interview PRODUCT.md explicitly retired.

---

# AI-Developer Review — Agent Core, Providers, SSE, MCP, CLI (multirole review 3)

**Scope read:** every file in `server/internal/agent/` (fence, prompts, tools, coach/loop/fail, ledger, critic, gates, dispatch, SSE, taxonomy, both providers and parsers, planner, telemetry), `server/internal/mcpserver/` (all four tool files), `server/cmd/insideout/cli_roadmap.go` + `cli_version.go`, `server/internal/api/roadmap_ai.go`, `server/internal/readiness/`, `server/internal/mail/mail.go`, and the coaching-contract sections of `PRODUCT.md`.

**Checks I ran:** `go test ./internal/agent/ ./internal/mcpserver/` in `server/` — both **ok** (unit level; `DATABASE_URL` is not set in this shell, so the DB-gated tests inside those packages skipped — I am not claiming the integration suite). Coordinator leads 14 and 15 were mine at the static level: for 14 I compared PRODUCT.md's coaching contract against the stage prompts (result: AIDEV-08, AIDEV-09 — the agent-surface slice; a full end-to-end promise audit was not run); for 15 I verified statically that `mail.Send` no-ops and returns nil when no key is configured (`mail.go:53-59`), that `invite_create` returns the one-time code in its response so a keyless admin path exists (`product_tools.go:222-230`), and that auth email flows answer a uniform 200 (`auth_email_flows.go:27-38`). No live mail-provider exercise was possible (no key) — that item stands as the coordinator stated it.

## What the fencing work actually bought, and what it missed

The trust boundary is now well-drawn on its main surfaces: PRD title, sections, the fact ledger, critic findings, and the round-2 focus sections all ride in `<prd_data>` fences whose tag-like content is mechanically broken (`prompt_fence.go:15-21`), the fence clause is regression-tested for the coach/critic prompts and the second critic round (`prompt_fence_surfaces_test.go`), and the readiness-gap text that deliberately stays outside the fences is static code strings (`readiness/readiness.go:36-66`) — safe.

Two surfaces still ship untrusted text to a tool-enabled model raw:

1. **ExpandNode** (`roadmap_planner.go:204-206`) interpolates project title, node title, and node description — all user/agent-editable via roadmap update — into its user message with no `fenceData` and no `dataFenceClause`, while its sibling `PlanMVP` (lines 184-185) got both and a surfaces test. `api/roadmap_ai.go:193` confirms the store feeds it directly. This is the same exposure class AIDEV-07/13 closed; it was simply missed.
2. **`get_prd` tool results** return PRD title/sections as raw JSON in a tool-result message (`tools.go:143-153`). The fence clause scopes itself to `<prd_data>` regions, so instructions arriving via this channel sit outside its letter. The mechanical gates bound the damage, but the fence contract has a gap exactly where a PRD author can aim. (Folded into AIDEV-02's fix direction.)

## What "verified" means in code vs what the prompts promise

This is the strongest part of the codebase: `record_fact` grounds quotes against every user turn with a CJK-aware bigram matcher and rejects ungrounded ones (`ledger.go:131-141, 216-225`); the critic's defect findings are dropped when their quote isn't in the PRD (`critic.go:310-333`); stage gates are mechanical and never trust the model's say-so (`tools.go:244-297`), with stage changes parked for human confirmation (`stage_confirm.go`). But three contracts are weaker in code than on paper:

- **Completion** keys on `strings.Contains(finalText, "NO OPEN QUESTIONS")` (`coach.go:274`) while the prompt demands it as the *final line*, asserted only when nothing is open — any mention completes the conversation (AIDEV-05).
- **Attestation** is granted by the model's own `record_fact` call at a 70%-bigram fuzzy bar, with no user confirmation and no confirm/correct/remove API anywhere in `internal/api`, though PRODUCT.md:179-187 promises exactly those actions and the gates treat `attested` as human-vouched (AIDEV-09).
- **Override** — `resolve_finding` executes on the model's say-so alone (AIDEV-07), and worse, any later ranked reply *wholesale-replaces* the stored findings the finalize gate reads (AIDEV-01): new IDs, statuses reset, the independent critic's blocking findings erasable by a restatement that omits them. Between the overwrite and unconfirmed resolves, the "gate reads the list, never the model's say-so" comment at `critic.go:133-135` is currently more aspiration than invariant.

## Provider error taxonomy — what still falls through

`providerHTTPError` covers 429, 5xx/529, 400-with-"context", and 400/401/403/404. Everything else — concretely 402 payment-required, 408, 413, 422 — returns an unsentineled `errors.New` (`taxonomy.go:171-172`), and `countsAgainstCircuit` counts every unclassified failure (`coach_fail.go:20-28`, whose own comment admits it). Five consecutive such failures open the one global breaker (`20260720135755_ai_ops.sql:70`), so a lapsed billing card becomes a service-wide 503 — the precise poison-the-breaker outcome `ErrProviderConfig` exists to prevent (AIDEV-03). The new Responses in-stream classifier maps only `rate_limit_exceeded` and `server_error`; any other code (e.g. `invalid_request_error`) is unclassified and likewise breaker-feeding (`responses.go:314-323`).

## Token/cost accounting and output caps

Telemetry is clean: per-call rows with model, tokens, latency, and a stable error class, never content (`telemetry.go:13-27`). But the Responses request carries **no `max_output_tokens`** while the Anthropic side caps at 4096 (`anthropic.go:55`) — per-call output cost is unbounded on a schema the operator selects via `llm_factory.go` — and its truncation flag only fires on `status=="incomplete"`, which mostly accompanies a cap we never set, so the F13 truncated-marker protection is largely inert on that provider (AIDEV-06). Responses usage is also read only from `response.completed`, so errored streams account zero tokens.

## Kill/timeout paths for long turns

Disconnect, upstream silence (90s idle watchdog → `ErrUpstreamStall`), SSE-write failure, and wedged detached writes (10s `persistTimeout`) are all handled and tested. What's missing is a **total** turn bound: `maxToolIterations=12` bounds iterations, not wall time, so a token-every-80s trickle never trips any watchdog while holding one of four permits and the conversation lock (AIDEV-11).

## Schema keys in prompts vs parsers

The MCP `commit` tool reads an `unresolved` array its own schema never declares (`product_tools.go:126` vs 120-123) — schema-abiding clients can't send it, breaking CLI/MCP parity (`cli_version.go:126`) on PRODUCT.md's Commit record (AIDEV-04). Both findings parsers accept any `[A-Za-z]+` section key where the prompts enumerate eight (`prompts.go:138`, `critic.go:318-331`) (AIDEV-12).

## Findings table

| ID | Sev | Bucket | File:line | One-line summary | Fix |
|---|---|---|---|---|---|
| AIDEV-01 | High | FIX | agent/critic.go:121 | Ranked-reply capture wholesale-replaces stored critic findings; finalize gate's input is model-rewritable in one turn | Merge by content, preserve IDs/statuses; full replace only on the stage's first ranked reply |
| AIDEV-02 | High | HARDEN | agent/roadmap_planner.go:206 | ExpandNode sends user-writable project/node text unfenced, no data-fence clause (PlanMVP fixed, this path missed) | Mirror PlanMVP: clause + fenceData; extend the surfaces test |
| AIDEV-03 | Med | HARDEN | agent/taxonomy.go:171 | 402/408/413/422 and non-load in-stream Responses codes fall through unclassified and feed the global breaker (5 consecutive → 503 for everyone) | Map residual 4xx + unknown in-stream codes to ErrProviderConfig |
| AIDEV-04 | Med | FIX | mcpserver/product_tools.go:126 | MCP commit parses `unresolved` its schema never declares — custom unresolved items unreachable via MCP (CLI/API have them) | Declare it (sections_json-style string if no array option) |
| AIDEV-05 | Med | FIX | agent/coach.go:274 | Conversation completes on substring "NO OPEN QUESTIONS" anywhere, not the promised final line | Last non-empty line must equal the marker |
| AIDEV-06 | Med | FIX | agent/responses.go:113 | Responses provider sends no max_output_tokens (Anthropic caps 4096); truncation detection rarely fires | Add max_output_tokens=4096; keep incomplete mapping |
| AIDEV-07 | Med | HARDEN | agent/tools.go:299 | resolve_finding executes with zero user signal though its description promises "once the user has responded" | Park resolutions as pending, apply on human confirmation (pendingStage pattern) |
| AIDEV-08 | Med | IMPROVE | agent/prompts.go:44 | Clarify prompt hardcodes one-question/rotation/8-count; PRODUCT.md:87-100 promises adaptive grouping, and pause/abandon recommendations are unprompted | Rewrite to the PRODUCT.md contract; add the four-way recommendation clause |
| AIDEV-09 | Med | IMPROVE | agent/ledger.go:239 | "Attested" = model-recorded at 70% bigram fuzzy grounding; PRODUCT.md's confirm/correct/remove actions have no API | Build the fact-confirmation API + present-before-Commit, or rename the status honestly |
| AIDEV-10 | Low | HARDEN | agent/ledger.go:236 | Fact text, assumptions (uncounted), section markdown uncapped — bloat meta and every later prompt | Rune-cap + assumption count cap, planner-style |
| AIDEV-11 | Low | HARDEN | agent/coach_loop.go:16 | No wall-clock turn bound; slow-trickle upstream pins 1 of 4 permits and the conversation lock indefinitely | Generous turn deadline as a classified non-circuit error |
| AIDEV-12 | Low | IMPROVE | agent/prompts.go:138 | Both findings parsers accept any alphabetic section key | Validate against store.PrdSectionKeys, drop mismatches |

## Strengths worth keeping

- The fence is mechanical and regression-tested, not prose (`prompt_fence.go`, `prompt_fence_surfaces_test.go`).
- Grounding is code, not prompt discipline: bigram quote matching at record time, ungrounded critic defects dropped (`ledger.go`, `critic.go:325-327`).
- Stage gates + staged writes everywhere: pendingStage confirmation, MCP draft/confirm verbs, conflict-retry on section edits (`stage_confirm.go`, `draft_tools.go`, `tools.go:181-186`).
- A real error taxonomy with sentinels driving retry/breaker/copy, and an idle watchdog that distinguishes stalls from disconnects (`taxonomy.go`, `anthropic.go:182-195`).
- Terminal-state guarantee on every turn, panic-safe, with bounded detached persists (`coach.go:158-177`).

## Ship first

The tool-policy change to ship first: **extend the pending-confirm policy to `resolve_finding`** so a finding can only be closed by the human who confirms it — the same interaction `advance_stage` already uses — and in the same change fix `captureRankedCritique` to merge instead of overwrite. Together they restore the one invariant this product's trust story depends on: the critique→finalize gate reads state that no single model turn can erase or satisfy alone. The ExpandNode fence (five lines plus a test) should ride the same commit; it is the last unfenced door on a tool-enabled prompt.

## What this role would do first

Fix critic.go captureRankedCritique (line 121) to merge with stored findings instead of saveCriticFindings wholesale-replacing extra["critic_findings"] — it is the only place the finalize gate's input is model-rewritable in one turn, and one focused test pins it.
