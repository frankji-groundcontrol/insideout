# PRD agent harness — design plan

Status: **reviewed and revised** — plan-ceo-review (13 findings, 1
blocking) and plan-eng-review (20 findings) both executed; every
blocking/major finding is resolved in this revision. See
[reviews.md](reviews.md) for verbatim findings and resolutions.

## 1. Problem

PRD writing is a subtle task whose failure mode is **confident fabrication**:
a naive agent fills the template with plausible users, metrics, and
requirements the human never said. The current harness
([architecture](../../architecture/prd-coach-agent.md),
`server/internal/agent/`) is a good skeleton — four stages, three tools, SSE
— but nothing structurally prevents fabrication, and the critique stage is
the same model critiquing itself in the same context (documented
sycophancy risk, [research](research.md)). Operationally, the harness also
has verified production gaps (checked against code, not hypothesized):

- `internal/agent/anthropic.go` uses `http.DefaultClient` — no timeouts; a
  hung upstream pins a goroutine and the user's stream forever.
- No per-conversation serialization: two concurrent sends to one
  conversation interleave history writes and double-bill.
- Partial streamed text is lost on failure (`UpdateAgentMessageContent`
  runs only on success); empty assistant placeholder rows accumulate —
  already visible in our own smoke-test history.
- No stale-run reaper: a crash mid-turn leaves `ai_runs` rows `running`
  forever, and the rate limiter counts them — a crashy hour can lock a
  user out for the sliding window.
- `ai_runs.idempotency_key` (column + unique index) and the
  `ai_run_events` telemetry table exist in SQL but are dead code.
- No global bound on concurrent upstream calls; no token budget beyond
  `max_tokens=4096` per call; no context-window-overflow handling; no
  startup model check (BUG-009's `model_not_found` surfaced only at the
  first user request).

## 2. What the research says (see [research.md](research.md))

1. **Elicitation is the highest-leverage stage.** STORM's
   perspective-guided questioning ~tripled unique sources gathered before
   writing; GATE's preregistered human study found LLM interviewing beats
   user-written prompts. Every praised PRD-AI product (ChatPRD et al.) has
   a mandatory clarifying-question round; one-shot generation grades C–F.
2. **Good PRDs have mechanical properties** worth encoding as gates, not
   vibes: problem-not-solution phrasing (Cagan's "what vs how"), a named
   narrow segment (reject "everyone"), current alternatives + why they
   fall short (Amazon PR/FAQ), measurable objectives with rank-ordered
   traceable requirements, explicit assumptions with validated/unvalidated
   tags, "top 3 reasons this fails" — truth-seeking, not selling.
3. **Simplest harness that works** (Anthropic guidance): fixed workflow >
   autonomous loop; multi-agent only where parallel/fresh-context genuinely
   pays. Self-critique measurably helps only with concrete rubrics and
   bounded rounds; unstructured debate converges to confident-wrong.
4. **Anti-hallucination mechanics from gstack** transplant directly: a
   quote-the-motivating-line verification gate (unquotable findings are
   demoted), one-issue-one-question interaction with an opinionated
   recommendation and a "keep as is" option, no re-litigating settled
   decisions, fixed rubric sections with "No issues found" required rather
   than skipping, a machine-checkable terminal artifact.

## 3. Design principles

1. **Elicit, never invent**: everything in the PRD traces to something the
   user said, or is explicitly labeled an assumption/open question.
2. **Mechanical gates over model judgment** where possible: stage advances
   and "done" claims validated server-side, not by the model's say-so.
3. **Fresh context for critique**: the critic never sees the chat history
   that produced the draft — only the artifact and the evidence.
4. **Simplest harness**: one primary coach + one fresh-context critic
   pass. No persona crews. Ceilings documented with upgrade paths.
5. **Fail visibly, degrade gracefully**: every provider failure has a
   user-visible outcome and a terminal DB state; under pressure the
   harness sheds optional work (critic pass) before core work (coaching).

## 4. Harness design — coaching quality

### 4.1 Fact ledger (the grounding backbone)

A conversation-scoped, structured list of user-attested facts, stored in
`agent_conversations.meta` (jsonb, exists today, unused — no migration):

```json
{"facts": [{"id": "f3", "kind": "problem|segment|alternative|whynow|goal|constraint|evidence|decision",
            "text": "...", "quote": "<verbatim user words>",
            "status": "attested|assumed|needs-validation"}]}
```

- New tool `record_fact(kind, text, quote)` — the coach must record a fact
  (with the user's verbatim words as `quote`) before it may rely on it.
  Server-side validation: `quote` must be a fuzzy substring of some prior
  user message in this conversation, else the tool errors — the
  quote-the-motivating-line gate applied to *generation*.
- New tool `mark_assumption(text)` for things the coach proposes and the
  user has not confirmed; they render in the PRD as `[ASSUMPTION]` items.
- Bounds: max 20 facts per kind, quotes truncated at 300 chars —
  enforced in `record_fact`, so the ledger can't become the context bloat
  its own auto-tighten path preserves.
- Quote validation is CJK-aware: NFKC-normalize both sides, strip
  whitespace/punctuation, then character-level substring / n-gram overlap
  (the product is Chinese-first; word-token fuzzy matching doesn't work on
  CJK). Unit tests use Chinese fixtures.
- The ledger is injected into every system prompt (it survives the last-20
  history window — fixing the "clarify facts fall out of context by
  draft time" hole) and returned by `get_prd`.
- SSE: new `fact_recorded` event so the UI can show a growing evidence
  panel; the user sees exactly what the coach thinks it knows.

### 4.2 Stage gates (server-side, mechanical)

`advance_stage` stops trusting the model. `tools.go` validates:

- `clarify → draft`: ledger contains ≥1 `attested` fact for each of
  `problem`, `segment`, `alternative`, `whynow`. Segment text failing an
  "everyone" lint (a short deny-list check) blocks with a targeted error
  the model relays as a question.
- `draft → critique`: all 8 sections non-empty (already checkable via
  `get_prd`; now enforced server-side).
- `critique → finalize`: at least one critic pass completed **or a
  recorded `critic-skipped` marker** (degradation, 5.1 — the skip
  satisfies the gate but forces a `[CRITIC SKIPPED]` line into the
  finalize report; degradation never hostage-takes finalize), and no open
  `blocking` findings, or the user has explicitly overridden (override
  recorded as a `decision` fact — no re-litigating).

Escape hatch: the user can always say "just draft it" — the coach records
a `decision` fact and the gate honors it, marking skipped prerequisites as
`[ASSUMPTION]`s in the PRD. Caution must not become hostage-taking.

### 4.3 Stage prompt upgrades (`prompts.go`)

- **clarify**: perspective-guided questioning (STORM): the prompt cycles
  lenses — target user, skeptical exec, engineer, QA — one question at a
  time, ≤2 questions per lens, hard cap ~8 questions before offering to
  draft with assumptions. Each answer → `record_fact`.
- **draft**: per-section evidence discipline: the coach reports which
  ledger fact ids ground each section via a `section_facts` argument on
  `update_prd_section`, persisted as a per-section map in conversation
  `meta` (`{section: [factIds]}`) — **out-of-band, never embedded in the
  section markdown** (eng-review fix: HTML comments would sit visibly in
  the plain-textarea editors and required export stripping no phase
  implemented). The evidence panel renders the mapping. Anything without
  a fact becomes `[ASSUMPTION]` or an `OPEN QUESTION:` line — never
  silent plausible filler. Requirements phrased as problems,
  not solutions (Cagan lint in-prompt; the critic re-checks it).
- **critique**: becomes a thin relay stage — it presents critic findings
  (4.4) one at a time, gstack-style: one issue per turn, an opinionated
  recommendation, a "keep as is" option, then applies the user's decision
  via `update_prd_section` and records it as a `decision` fact.
- **finalize**: emits a fixed-format completeness report (per-section
  solid/thin/empty, assumption count, open questions listed verbatim) and
  ends with a sentinel line: either `NO OPEN QUESTIONS` or the enumerated
  list — machine-checkable, never silently defaulted.

### 4.4 The critic (the "multi-agent" decision)

One additional role, not a crew: a **fresh-context critic pass** run at
critique-stage entry and after each revision round.

- Implementation: a second `ChatStreamer` call (same provider/model, no new
  infra) with a critic system prompt; input = PRD sections + fact ledger
  **only** — no chat history, killing same-context sycophancy.
- Rubric (fixed sections, anti-skip, "No issues found" required): the
  Amazon PR/FAQ 7 reviewer questions + Cagan checks (problem-not-solution
  per requirement; measurable objectives; buildability/testability: "could
  an engineer build and a QA test from this?"). Output is structured JSON
  (parsed, not streamed to the user): findings with
  `{section, severity: blocking|major|minor, quote, issue, suggestion}`.
- **Verification gate**: a finding whose `quote` is not a substring of the
  actual PRD text is dropped server-side. Findings referencing facts the
  ledger doesn't support are tagged `unsupported-by-evidence`.
- **Findings persist server-side** (review-blocking fix): in
  `agent_conversations.meta.critic_findings` as
  `{id, section, severity, quote, issue, status: open|resolved|overridden}`,
  written when the critic pass parses, updated by the relay stage as the
  user decides each finding — the critique→finalize gate reads this list,
  never the model's say-so. The critic runs in the request handler
  immediately after a successful `advance_stage` into critique and after
  each applied revision round.
- **Structured output is forced, not scraped**: the critic gets exactly one
  tool, `report_findings`, with `tool_choice: {type:"tool"}` added to the
  request struct (two-line wire change) — schema-validated, no JSON
  parsing of prose. Output-shape failures have their own taxonomy rows
  (5.2): one re-prompt retry with the parse error appended; on second
  failure, degrade to single-context critique with the same visible
  notice as contention — never a circuit failure, never a failed turn.
- Bounded: max 2 critic rounds per critique stage (self-refine thrash
  guidance); round 2 only re-examines previously-flagged sections.
- Why not more agents: Anthropic's guidance and the debate literature —
  cost multiplies, quality gains need genuinely independent context or
  parallel breadth, which one fresh-context critic already captures.
  Ceiling + upgrade path: if evals later show critic blind spots, add ONE
  adversarial "red team user" pass at finalize, not a persona crew.

## 5. Production hardening — dispatch, queueing, failure

All primitives are single-process + Postgres; no Redis/queue service.
`// ponytail:` ceilings are stated where they bite at multi-instance.

### 5.1 Dispatch and queueing

- **Per-conversation serialization**: an in-process `map[uuid]*sync.Mutex`
  (keyed by conversation) with `TryLock`; a second concurrent send gets
  HTTP 409 `{"code":"CONVERSATION_BUSY"}` immediately — no queuing user
  turns (a coaching chat has no meaningful "queued second message"; the
  UI already disables send while streaming, this closes the two-tab race).
  `// ponytail: in-process lock is single-instance; move to pg_advisory_lock on the conversation id when a second instance exists.`
- **Global upstream concurrency semaphore**: buffered-channel semaphore,
  size `AI_MAX_CONCURRENT` (default 4), acquired **once per turn, before
  the SSE stream opens** (review fix: per-call acquisition after
  `message_start` made the documented 503 impossible and let turns stall
  mid-tool-loop). Bounded pre-stream wait `AI_QUEUE_WAIT` (default 15s);
  on timeout, a true JSON 503 `CIRCUIT_OPEN`-shaped response — genuinely
  reusing the existing frontend countdown. No queued-position events (a
  15s max wait doesn't need a ticker).
- **Degradation ladder** under pressure: skip the critic pass (critique
  falls back to single-context review with a visible notice) before ever
  queuing or failing coach turns. The critic runs sequentially *inside*
  an already-permitted turn and adds no concurrency, so it **inherits the
  turn's permit** — it never acquires its own (eng-review fix: a
  `TryAcquire` would always fail against its own turn's permit at low
  `AI_MAX_CONCURRENT`, silently starving the critic under normal load).
  The degradation trigger is observed queue pressure at turn start
  (waiters > 0) or a half-open circuit, recorded as the `critic-skipped`
  marker.

### 5.2 Provider error taxonomy and retry policy

One classification point in `anthropic.go`, one policy in `coach.go`:

| Condition | Handling |
|---|---|
| 429 (with `Retry-After`) | One retry after min(Retry-After, 20s) with jitter; then SSE `error` `ANTHROPIC_RATE_LIMIT` + circuit `failure` |
| 529/5xx/connect refusal | One retry with 2s jitter backoff; then fail the turn, circuit `failure` |
| Timeout / mid-stream drop | No retry (tokens may have been consumed; a retried half-turn double-writes tools). Persist partial text (5.3), fail run, circuit `failure` |
| 400 context-length | No retry. Auto-tighten: rebuild prompt with ledger + last 6 messages only; one attempt; else surface "conversation too long — snapshot and start a fresh session" |
| 400 other / 401 / 403 / `model_not_found` | No retry; fail run with the provider's actual message (config error, not load) — **not** a circuit failure (it would poison the breaker for all users) |
| Content refusal (`stop_reason` refusal / empty) | No retry; surface honestly as coach message |
| `context.Canceled` (client disconnect) | Persist partial text, mark run `failed` (`error_message: canceled`), **no circuit result**, no retry — tab closes are routine, not provider health |
| Critic output unparseable / truncated / empty | One re-prompt with the parse error appended; second failure → degrade to single-context critique with visible notice; **not** a circuit failure |

- **HTTP client**: replace `http.DefaultClient` with explicit
  `Transport` timeouts (connect 10s, TLS 10s, response-header 30s) plus an
  **idle-stream watchdog**: no SSE bytes for `AI_STREAM_IDLE_TIMEOUT`
  (default 90s, thinking pauses are long) cancels the request.
- Existing DB circuit breaker (5-fail-open/2-success-close/30s cooldown)
  stays the shared cross-instance backstop; the taxonomy above feeds it.

### 5.3 Turn lifecycle: crash safety, partial work, idempotency

- **Persist partial text**: `runLoop` accumulates streamed text; on any
  failure the handler writes what streamed into the placeholder message
  with an `[interrupted]` marker before emitting SSE `error` — the user
  never loses a half-written answer, and history stops accumulating empty
  rows.
- **Terminal-state guarantee**: `defer` in the handler marks the run
  `failed` on any non-success path (panic included, via the existing
  recover middleware ordering — verified in delivery tests).
- **Stale-run reaper**: on startup and every 5 min, mark runs stuck in
  `pending`/`running` with no heartbeat for 10 min as `failed`.
  **Heartbeat**: `ai_runs.updated_at` is touched at the start of every
  provider call (coach, tool-loop continuation, critic round) — without
  it the reaper would kill healthy long turns (90s thinking silences ×
  tool loop × critic rounds legitimately exceed 10 min). Review
  correction: reaping does **not** relax rate limiting —
  `ai_check_rate_limit` counts by `created_at` regardless of status, by
  design (anti-abuse). The reaper fixes contradictory run states and dead
  placeholders, nothing more.
- **Idempotency**: deferred out of scope (review consensus): the
  per-conversation lock already answers the in-flight duplicate (409),
  the UI refetches history on reconnect for the after-completion case,
  and returning "the original run's terminal state" on an SSE endpoint
  is a new, unspecified response shape. The dormant `idempotency_key`
  column loses nothing by waiting for real evidence of need.
- **Client disconnect**: request context cancellation already propagates
  to the provider call (stop paying for tokens); the turn is then
  finalized exactly like a timeout (partial text + `failed` run) — and the
  UI refetches history on reconnect (existing behavior).

### 5.4 Budgets, observability, config safety

- **Token budgets** (fast-follow after telemetry lands, not H4): running
  total cached in conversation `meta` (O(1) check, updated with each
  telemetry write) rather than a per-turn aggregate over `ai_run_events`;
  the 150k default is a placeholder until a week of real usage data
  exists. Exceeded → coach suggests snapshot + fresh conversation; critic
  calls draw from the same budget.
- **Telemetry**: actually call `InsertAIRunEvent` (dead today) per
  provider call: model, input/output tokens (from `message_delta.usage`),
  latency, error class, coach-vs-critic. **Plus product telemetry**
  (review addition — infra metrics can't tell you whether the coaching
  mechanics work): facts recorded per conversation, gate-denial counts,
  escape-hatch ("just draft it") rate, `[ASSUMPTION]` density in
  finalized PRDs, critic findings accepted vs kept-as-is. If 80% of users
  escape-hatch past clarify, the ledger is theater — these counters are
  how we'd know.
- **Startup model check**: on boot with a token configured, `GET
  /v1/models`; if `AI_MODEL` is absent, log a loud warning (don't crash —
  gateways lie) — turns BUG-009's silent config drift into an operator
  signal.
- **SSE keepalive**: emit `: ping` comment lines every 15s during model
  silence; delivery includes verifying end-to-end streaming through the
  Nitro proxy (buffering would break everything and only shows up live).
- **PRD edit conflict**: the coach's `update_prd_section` and a user's
  manual section save currently race (silent last-write-wins). Fix via
  compare-and-swap on `prds.updated_at` (`UPDATE ... WHERE id=$1 AND
  updated_at=$2`) — no migration, no new column; the "rev" is the
  timestamp the tool read via `get_prd`. On mismatch the tool errors and
  the coach re-reads — the human always wins.

## 6. Delivery

Phased; each phase independently shippable and verified. No schema
migrations required (ledger uses `meta`; everything else is code).

| Phase | Scope | Files (primary) |
|---|---|---|
| H1 Hardening core | HTTP timeouts + idle watchdog, error taxonomy + retry, partial-text persistence, terminal-state defer, reaper + heartbeat, per-conversation lock, pre-stream semaphore | `agent/taxonomy.go` (new), `agent/dispatch.go` (new), `anthropic.go`, `coach.go`, `store/agent_messages.go`, `cmd/insideout/main.go` |
| H2 Ledger + gates | fact tools + validation, meta storage, prompt injection of ledger, server-side stage gates, `fact_recorded` SSE, frontend evidence panel | `tools.go`, `prompts.go`, `coach.go`, `store/agent_conversations.go`, `app/src/composables/useCoachStream.ts`, PRD page |
| H3 Critic | critic prompt + structured-output call, verification gate, bounded rounds, critique-stage relay protocol, degradation ladder | `agent/critic.go` (new), `prompts.go`, `coach.go` |
| H4 Telemetry + polish | ai_run_events wiring + product telemetry, startup model check, finalize completeness report + sentinel (token budgets = fast-follow on real data; idempotency dropped) | `anthropic.go`, `coach.go`, `store/agent_messages.go`, `api/conversations.go` |

Testing per repo practice (no mocks; real APIs):

- Unit: error-taxonomy classification (table-driven, real captured error
  bodies as fixtures), gate validation, quote-verification (CJK fixtures
  mandatory), reaper SQL + heartbeat, critic finding filter (verbatim /
  paraphrased / fabricated quotes; supported and unsupported facts),
  critic parse-failure → graceful degradation, auto-tighten prompt
  rebuild. `maxToolIterations` raised to 12 (multi-fact answers + the
  8-section draft burst exhaust 6).
- Integration (DATABASE_URL-gated): stage-gate deny paths, 409 on
  concurrent send, reaper against seeded stale rows, **section-edit
  conflict** (critical per review: interleave a manual save with a
  stale-rev tool write; assert the human edit survives and the coach
  retries cleanly), **dispatch contention** (`AI_MAX_CONCURRENT=1` + a
  slow fake ChatStreamer built on the template.go pattern: assert the
  503 contract, that the critic completes while its own turn holds the
  sole permit, and that the skip marker appears only under genuine queue
  pressure).
- Live (ANTHROPIC_AUTH_TOKEN-gated): full clarify→finalize session
  against the real API; kill-the-connection mid-stream and assert partial
  text persisted + run failed; SSE keepalive observed through the Nitro
  proxy.
- Eval (new, small): 3 scripted fixture conversations (one with a
  multi-fact answer and a full draft burst): (a) fabrication metric —
  zero PRD sentences lacking a fact citation or `[ASSUMPTION]` tag;
  (b) **outcome metric** (review addition — don't measure only the
  seatbelt): the same fixtures through old and new harness, graded by a
  decomposed binary checklist (named narrow segment? measurable
  objectives? current alternative named? requirements problem-phrased?);
  (c) **critic precision**: fixtures with planted defects + known-good
  sections — recall on planted, false-positive rate on good (also
  answers the critic-model open question).

## 7. Risks

- **Question fatigue**: elicitation-first can feel like an interrogation.
  Mitigated by the lens cap (~8 questions), the "just draft it" escape
  hatch, and the visible evidence panel making questions feel productive.
- **Cost**: critic pass adds ~1 call per critique round (bounded at 2);
  budgets + degradation ladder cap the blast radius.
- **Quote validation false-negatives**: paraphrased answers may fail the
  fuzzy-substring check; threshold tuned permissively, and failure mode is
  a tool error the model can recover from (re-quote), never a lost turn.
- **Mutex-map growth**: conversation lock entries are deleted on
  uncontended unlock (map guarded by one mutex, ~10 lines); the residual
  leak is ~100B per touched conversation, accepted.
- **Single-instance primitives**: the lock and semaphore are in-process;
  documented ceilings with named upgrades (advisory locks, DB-backed
  semaphore) — acceptable while deployment is one server container.

## 8. NOT in scope

- Multi-provider support (stays behind `ChatStreamer`, Q5 unchanged).
- A persona-crew multi-agent writer (rejected on evidence, §4.4).
- Rich-text PRD editing, avatar upload, cookie theme persistence (existing
  known limitations, unrelated).
- Distributed queueing infra (Redis/NATS) — ceilings documented instead.
- Retrieval/RAG over external product docs — a future evidence source for
  the ledger, not v1.
- Idempotency-key wiring — deferred per review (see 5.3); the column
  stays dormant.
- **Roadmap as a timed branched tree** (user idea, recorded so it isn't
  lost): evolve the project board's flat timeline into a time-axised
  branching tree — ideas and PRDs branching off projects, branches
  merging or dying, giving the group-leader view real history-shape. A
  product-model change to the tracking pillar, not the coaching harness;
  deserves its own plan (the tree is largely derivable from existing
  `ideas.prd_id` / `prds.project_id` links plus timestamps — likely a
  frontend-first experiment).

## 9. Open questions (unresolved decisions)

Merged from both reviews; none silently defaulted:

1. Critic model: same as coach vs cheaper — now decidable by the critic
   precision eval (6c) instead of guesswork; run it on both.
2. `AI_MAX_CONCURRENT` (4) and `AI_QUEUE_WAIT` (15s) defaults are guesses
   pending telemetry; accept the guesses or ship the semaphore disabled
   until data lands.
3. Evidence panel: chat-only fact editing (plan) vs UI CRUD — product
   call.
4. Gate strictness under load: contention-skipped critic satisfies the
   finalize gate with a visible marker (plan), vs critic block-acquiring
   when gate-critical (stricter, slower under load).
5. H4 slim-down (budgets → fast-follow, idempotency dropped) is adopted
   in this revision — flag if you prefer shipping budgets in H4 on
   placeholder defaults.

## 10. Outside-voice revisions (2026-07-21, codex cross-model pass)

A Codex outside-voice review (15 findings) ran after the eng-review
sections; per user delegation the clearly-correct findings are adopted
here as binding plan amendments:

1. **Omission findings must survive the quote gate**: the critic's
   verification gate drops findings with no PRD substring to quote —
   exactly the omissions the rubric targets ("no measurable objective").
   Amendment: findings carry `kind: defect|omission`; omission findings
   quote nothing but must name the section + missing rubric element, and
   are validated against the rubric list instead of the PRD text.
2. **Terminal-state writes use a detached context**: all cleanup
   (partial-text persist, MarkAIRunFailed, circuit record) runs on
   `context.WithoutCancel(ctx)` — with `r.Context()` a client disconnect
   cancels the very writes that record the cancellation.
3. **Reaper vs RLS**: `ai_runs` is RLS-forced and user-scoped, so a
   global reaper UPDATE matches zero rows. Amendment: one small migration
   amends the `ai_runs` policy to treat `current_user_id() IS NULL` as
   the trusted system context (same pattern the `users` table already
   uses) — the "no migrations" claim is dropped (now: one policy
   migration).
4. **Acquire lock + permit before any durable turn writes**: a 503 must
   not leave a `running` run row, a persisted user message, and an empty
   placeholder behind.
5. **Frontend keeps partial text on SSE `error`**: `useCoachStream`
   currently discards `streamingText`; amendment: retain it and refetch
   history — backend persistence alone doesn't satisfy "never lose a
   half-written answer".
6. **Telemetry is metadata-only**: `ai_run_events` is non-RLS; the wiring
   stores tokens/latency/error-class/role only — never prompts, quotes,
   PRD text, or provider identifiers.
7. **Conflict retry may not clobber the human**: on CAS failure the coach
   re-reads and may retry only sections whose content it did not just see
   change; a section the human altered gets a relayed question, not a
   rewrite.
8. **H2/H3 sequencing**: the finalize gate's critic requirement activates
   with H3; H2 ships the gate checking only sections + facts (else H2
   alone strands conversations in critique).
9. **In-turn stage switch**: `runLoop` holds one system prompt per turn,
   so `advance_stage` mid-turn doesn't change behavior until the next
   turn. Amendment: after a successful `advance_stage`, the loop rebuilds
   the system prompt before the next provider call; critic-entry runs
   after the turn completes.
10. **Finalize is enforced, not prose**: the server validates the
    completeness report shape and calls `CompleteConversation` (currently
    dead) — the sentinel is checked, not trusted.
11. **Baseline before replacement**: fixture outputs are captured on the
    current harness *before* H2 lands, so the outcome eval has a real
    old-harness baseline.
12. **Acknowledged residual risks** (not mechanically closable): a
    genuine quote can still be paired with a distorted `text` (mitigated
    by the evidence panel showing quote+text pairs and the critic
    cross-checking them — human remains the verifier); the pre-existing
    CheckRateLimit/CreateAIRun race stays an accepted ceiling at 10/min
    granularity; `tool_choice` needs a small `ChatStreamer` option, not a
    wire-only change.

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 1 | issues_open | 13 findings (1 blocking), all folded into the revision |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 1 | issues_open | 38 issues across 4 sections; majors resolved in §5/§6/§10 |
| Outside Voice | codex (cross-model) | Independent 2nd opinion | 1 | issues_found | 15 findings; 11 adopted as §10 amendments, 4 residual risks acknowledged |
| Design Review | `/plan-design-review` | UI/UX gaps | 0 | — | — |

- **CODEX:** the cross-model pass caught three enforcement gaps the section review missed — the reaper vs forced-RLS conflict, cleanup writes dying on the cancelled request context, and the in-turn stage-switch no-op — all now in §10.
- **CROSS-MODEL:** eng-review and codex agreed on the load-bearing structural fixes (permit inheritance, lock-before-durable-writes, critic omission-findings). No unreconciled tension: where they overlapped they agreed; codex added enforcement-correctness findings the rubric sections didn't reach.
- **VERDICT:** ENG reviewed, not CLEARED — this is a design plan, not a diff; the review's job was to harden it, and every blocking/major finding is resolved or explicitly accepted as a residual risk (§10.12). Ready to implement once the plan §9 user decisions are made.

**UNRESOLVED DECISIONS:**
- Critic model: same as coach vs cheaper (decidable by eval 6c).
- `AI_MAX_CONCURRENT` (4) / `AI_QUEUE_WAIT` (15s) defaults pending telemetry.
- Evidence panel: chat-only fact editing vs UI CRUD.
- Gate strictness under load: skip-marker satisfies finalize vs critic block-acquires when gate-critical.
- H4 slim-down (budgets → fast-follow, idempotency dropped) — ratify or ship budgets in H4 on placeholder defaults.
- Whether to accept the residual quote-pairing risk (§10.12) or add a stronger fact-attestation step.
