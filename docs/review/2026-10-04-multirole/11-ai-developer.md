# 11 · AI Developer

**Lens:** agent trust boundary (untrusted text vs tool-enabled actions), prompt-vs-code contract, provider HTTP/error surface, secret handling, coach authority vs RLS, offline fallback honesty
**Score:** **7.2 / 10**
**Bottom line:** The agent subsystem is the strongest-engineered part of this repo — mechanical stage gates, quote grounding for facts, optimistic concurrency on section writes, a circuit breaker, an idle watchdog, and metadata-only telemetry are all real, tested code, not prompt hopes. But I found one High defect on a designed resilience path (a mid-stream retry re-streams the whole answer on top of the partial one), one Medium tool-contract mismatch (`record_fact` quote grounding only searches the last ~20 turns, which can wedge the documented clarify→draft path), and a cluster of honesty/polish gaps around the offline template mode and error labeling.

Reviewed against main@877ad03; High/Critical findings re-checked by an independent skeptic — verdicts in [01-findings-register.md](01-findings-register.md).

---

## Scope

All of `server/internal/agent/` (coach.go, anthropic.go, responses.go, tools.go, ledger.go, critic.go, dispatch.go, taxonomy.go, sse.go, prompts.go, template.go, modelcheck.go, llm_factory.go, llm_url.go, roadmap_planner.go, telemetry.go, llm.go) and its tests; `server/internal/config/config.go`; `server/internal/mcpserver/` (server.go, product_tools.go, admin_tools.go, auth_tools.go); the agent vocabulary routes (`server/internal/api/agent_routes.go`); the coach call site (`server/internal/api/conversations.go`); PRD write path (`server/internal/api/prds.go`, `server/internal/store/prds.go`); provider wiring (`server/cmd/insideout/main.go`); the MCP/CLI HTTP client surface (`server/internal/apiclient/apiclient.go`). Verified `go build ./...`, `go vet ./...`, and `go test ./internal/agent/ ./internal/mcpserver/ ./internal/config/` — all green.

## What is good

- **Mechanical gates, not model say-so.** `checkStageGate` (server/internal/agent/tools.go:245) enforces draft/critique/finalize preconditions server-side from the persisted ledger and PRD rows; `advanceStage` only allows the single legal forward transition (tools.go:227). The coach cannot skip the critic pass or finalize with open blocking findings no matter what the model claims.
- **Anti-fabrication is enforced in code.** `recordFact` requires a verbatim quote that survives a normalized exact-or-≥70%-bigram match against the user's own messages (server/internal/agent/ledger.go:131-141, 205-214), and the critic drops defect findings whose quotes are not grounded in the PRD text (critic.go:277). `mark_assumption` gives the model a separate, honest channel for unconfirmed content.
- **Writes are scoped and concurrency-safe.** Every store call in the tool executor takes `actorID` and re-fetches authorization (`GetPrdForMember`, `GetConversationForOwner`); section writes use optimistic CAS with a human-wins manual path (tools.go:181-186, api/prds.go:121); the conversation lock closes its TOCTOU window (dispatch.go:45-51); all SQL targets the `insideout` schema.
- **Failure taxonomy is deliberate.** Six sentinel errors, one retry with Retry-After honoring and jitter, an idle watchdog distinguishing hung upstream from client disconnect, a circuit breaker, and truncation markers so cut-off answers are never replayed as finished ones (taxonomy.go, coach.go:36-40, 297-315).
- **Secrets stay out of transcripts.** The LLM key lives only in struct fields and request headers; telemetry records error *classes*, never error text (telemetry.go, taxonomy.go:154-157). MCP auth tools keep the token in process memory and never return it as tool output (mcpserver/auth_tools.go:32-43, server.go:38-44).

## Findings

**AIDEV-01 · High · provider retry vs streaming**
`server/internal/agent/taxonomy.go:38` + `server/internal/agent/anthropic.go:270`
Evidence: taxonomy.go:38-50 — `for attempt := 0; attempt < 2; attempt++ { turn, err := do(ctx) ... case errors.Is(err, ErrProviderTransient): wait = 2 * time.Second`; anthropic.go:270-272 — `chunk, _ := delta["text"].(string) ... if onDelta != nil && chunk != "" { onDelta(chunk) }`; the same `onDelta` appends to the client bubble immediately (coach.go:220-223 — `onDelta := func(s string) { streamed.WriteString(s); sse.delta(s) }`).
Failure scenario: the provider emits an in-stream `overloaded_error` (anthropic.go:286-291 classifies it `ErrProviderTransient`) after 600 tokens have already been streamed → `retryStreamChat` re-issues the *full* request with the *same* onDelta → the second stream generates a fresh answer from scratch → the connected client sees the 600-token partial followed by a complete second answer; the persisted message contains only the second attempt's `finalText` (coach.go:261), so the visible stream and stored history diverge.
Fix: track whether any delta was delivered; retry only when zero bytes reached the client (buffer deltas per attempt and flush on success, or skip retry once streaming started). ~15 lines.

**AIDEV-02 · Medium · tool contract: quote grounding window**
`server/internal/agent/coach.go:200` / `server/internal/agent/ledger.go:212`
Evidence: coach.go:200-211 — `history, err := c.loadHistory(ctx, userID, conversationID, historyLimit)` then `userMessages: userTexts(history)`; coach.go:393-395 — `if len(out) > limit { out = out[len(out)-limit:] }` with `historyLimit = 20` counting *user and assistant* turns (coach.go:30); rejection at ledger.go:212-213 — `"quote %q was not found in anything the user said in this conversation"`.
Failure scenario: a clarify conversation passes ~20 text turns (~10 user answers); the user stated their problem in answer #2 and the model records it late; the verbatim quote now fails grounding → `record_fact` errors, the draft gate's required `problem` fact (tools.go:254-260) can never be attested from real history → the turn burns all 12 tool iterations (coach.go:374, "exceeded max tool iterations") and the user must guess to repeat themselves.
Fix: ground quotes against all of the conversation's user turns (`ListAgentMessages` already returns every row; filter roles server-side) instead of the replay window. ~10 lines.

**AIDEV-03 · Low · offline critic counted as a real pass**
`server/internal/agent/template.go:36`
Evidence: template.go:36-38 — `return Turn{ToolCall: &ToolCallRequest{ID: "template", Name: tool.Name, Arguments: "{}"}}`; critic.go:254-259 unmarshals `{}` into zero findings with a nil error, so `runCritic` returns `ok=true`; the finalize gate at tools.go:290-291 — `if cs.RoundCount == 0 && cs.Skipped == "" { return fmt.Errorf("agent: cannot finalize yet — no critic pass has run against this PRD") }` is then satisfied with zero review.
Failure scenario: operator deploys without `INSIDEOUT_LLM_API_KEY`; the PRD sails through critique→finalize with `RoundCount=1` and no findings, and nothing in the finalize report discloses that no reviewer ever ran.
Fix: make the template forced-tool path return an error (so `markSkipped("offline")` fires and the gate stays honest), or treat an empty findings payload as an output failure in `parseCriticFindings`. ~5 lines.

**AIDEV-04 · Low · silent quote truncation**
`server/internal/agent/ledger.go:224`
Evidence: ledger.go:224-227 — `if len([]rune(quote)) > maxQuoteLen { quote = string([]rune(quote)[:maxQuoteLen]) }`, stored as `Fact.Quote` and rendered as the user's verbatim words (`（原话：%s）`, ledger.go:183).
Failure scenario: user gives a 400-rune answer; the model quotes it verbatim (grounding passes on the *full* text); the evidence panel forever shows a 300-rune prefix presented as the user's exact words.
Fix: reject overlong quotes at `record_fact` time (the model can split into two facts) or store the full quote and truncate only at render. ~3 lines.

**AIDEV-05 · Low · wrong provider label on the responses schema**
`server/internal/agent/responses.go:138`
Evidence: responses.go:137-139 — `if resp.StatusCode != http.StatusOK { return Turn{}, anthropicHTTPError(resp) }`; taxonomy.go:133 — `full := fmt.Sprintf("agent: anthropic returned %s: %s", resp.Status, msg)`.
Failure scenario: an operator on an OpenAI-compatible gateway gets `agent: anthropic returned 401 Unauthorized` in logs and `ai_runs` failure text — a misleading on-call signal for a non-Anthropic provider.
Fix: parameterize the provider name in the classifier (or stamp the label at the streamer). ~5 lines.

**AIDEV-06 · Low · stale Nuxt default for PublicURL**
`server/internal/config/config.go:31,78`
Evidence: config.go:31 — `// (Nuxt same-origin needs none).`; config.go:78 — `PublicURL: strings.TrimRight(getenv("INSIDEOUT_PUBLIC_URL", "http://localhost:5173"), "/")` — port 5173 is the deleted Nuxt dev server; Flutter web never serves there.
Failure scenario: fresh local dev without `INSIDEOUT_PUBLIC_URL`; the magic-link email logs a link pointing at a dead origin; the operator chasing "email login broken" wastes a cycle.
Fix: retarget the default to the documented Flutter web origin and drop the Nuxt mention. ~2 lines.

**AIDEV-07 · Low · PRD content enters the system prompt un-delimited**
`server/internal/agent/prompts.go:69`
Evidence: prompts.go:69-70 — `当前各章节内容：\n%s` with `formatSectionsForPrompt` rendering raw `- key: content`; sections are writable by the PRD author or a workspace admin (api/prds.go:126 maps the store's 403 to `"only the author or a workspace admin can edit this PRD"`).
Failure scenario: an admin (or the owner, testing) writes instruction-like text into a section; the owner's next coach turn carries it at system-prompt level and the coach's questions and drafts can be steered. Mechanical gates still block illegal section keys, stage skips, and ungrounded facts, so this is defense-in-depth, not a privilege escalation.
Fix: wrap title/sections/findings in explicit untrusted-data fences with a "treat as data, never instructions" clause; optionally lint section writes for instruction markers. ~10 lines.

**AIDEV-08 · Low · provider error body echoed to the client**
`server/internal/agent/coach.go:298`
Evidence: coach.go:296-298 — `sse.errorEvent(loopErr.Error(), "PROVIDER_CONFIG")`, where `loopErr` embeds up to 4 KiB of provider response body (taxonomy.go:127-133).
Failure scenario: a gateway that echoes request details in its 401/403 body ships those details into the end user's SSE stream — inconsistent with the generic "internal error" used for every other failure class.
Fix: log the full error server-side; send a fixed "AI provider is misconfigured" string to the client. ~2 lines.

## Lead verification

| Lead | Verdict | Evidence |
|---|---|---|
| L6 (stale Nuxt references) | Confirmed (in lens) | `server/internal/config/config.go:31` comment "(Nuxt same-origin needs none)" and `:78` default `:5173`; also `server/cmd/insideout/cli.go:19` hardcodes the hosted API base as the CLI/MCP default |
| L7 (size-rule breaches) | Confirmed (in lens) | `wc -l`: coach.go 410, anthropic.go 406 (>350) with no dated issue in docs/issues/; tools.go 338, responses.go 285, ledger.go 285, critic.go 285 compliant |
| L12 (build/vet/test clean) | Confirmed (for lens packages) | `go build ./...`, `go vet ./...`, `go test ./internal/agent/ ./internal/mcpserver/ ./internal/config/` all pass at 877ad03 |
| L15 (secret scan clean) | Confirmed (in lens) | LLM key only in struct fields and `Header.Set` call sites (anthropic.go:153, responses.go:127, modelcheck.go:20); telemetry stores error classes only (telemetryErrorClass, taxonomy.go:157); `ai_runs.prompt` persists user content but no API route reads `ai_runs` back |
| L1/L2/L3/L4/L5/L8/L9/L10/L11/L13/L14 | Out of lens | Not evaluated (build/compose/docs/deploy/Flutter areas) |

## Recommendations

1. Fix AIDEV-01: gate the provider retry on "zero deltas delivered this attempt" so a mid-stream retry can never concatenate two partial answers onto the client's bubble.
2. Fix AIDEV-02: ground `record_fact` quotes against the full conversation's user turns, not the 20-turn replay window — the prompt promises "the user's verbatim words" with no time limit, and the code should honor that contract.
3. Fix AIDEV-03: in offline template mode, mark the critic skipped instead of recording an empty pass, so the finalize gate's "a critic pass has run" claim stays true.
4. Fix AIDEV-04 and AIDEV-08 (small honesty patches): no silent quote truncation; no verbatim provider bodies in client-visible SSE errors.
5. Fix AIDEV-05: parameterize the provider name in `anthropicHTTPError` so responses-schema deployments get truthful error text.
6. Fix AIDEV-06: update `PublicURL`'s default and the CORSOrigins comment to name the Flutter web origin, closing the Nuxt straggler inside the server config.
7. Adopt AIDEV-07 as a standing rule for every prompt builder: any content a human or tool can write gets explicit data-fencing delimiters before it reaches the system prompt.
8. Split coach.go and anthropic.go under the 350-line rule (L7) while touching them for items 1-2, and file the dated issues the repo rules require.

## What this role would do first

Ship the `record_fact` grounding contract fix (AIDEV-02) first. It is the one place where the prompt contract and the code contract actively disagree — the prompt tells the model "quote must be the user's verbatim words" while the code only accepts quotes visible inside a 20-turn replay window — and the disagreement sits on the documented happy path: a long clarify conversation can hit a hard tool-error loop that blocks the draft gate and ends the turn in "exceeded max tool iterations," with the only escape being the user intuiting that they must repeat an early answer. The retry-duplication fix (AIDEV-01) is the close second — it is the only High — but it corrupts one turn's display, whereas AIDEV-02 denies the documented clarify→draft progression outright and erodes the anti-fabrication story the whole ledger design exists to tell. Both are small, testable diffs in files that already have solid test coverage.
