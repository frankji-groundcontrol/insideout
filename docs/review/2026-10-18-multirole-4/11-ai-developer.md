# 11 · AI developer

**Lens:** prompt trust boundary and data fence, finalize and critic gates, stage handling and human confirmation, provider streaming and error taxonomy, token timeout retry concurrency bounds, bilingual en/zh prompts, offline template coach honesty, MCP tool policy and authority  
**Score:** **6 / 10**  
**Bottom line:** The coach loop is well bounded and its stage gates are server-side and human-confirmed, but the trust boundary has holes (get_prd tool result unfenced, MCP outputs unlabelled), a skipped critic silently opens the finalize gate, and the MCP tool policy is advisory prose rather than enforcement, so an owner-token agent can accept its own proposals.

Tree: `main` @ `9f0b1c7`. Every High this lens filed went to an adversarial skeptic cluster; where the skeptic changed a severity or corrected a claim it is marked **[Correction]** below. The full register is [01-findings-register.md](01-findings-register.md).

## Skeptic verdicts on this lens's Highs

| ID | Filed | Verdict | Final | Reason |
|---|---|---|---|---|
| AIDEV-01 | High | DOWNGRADED | Medium | **[Correction]** The gap is real. tools.go:148 marshals prd.Title and prd.Sections raw. fenceData is not referenced anywhere in tools.go; it appears only in critic.go, prompts.go, prompts_en.go and roadmap_planner.go. The only fence contract (dataFenceClause) is scoped to <prd_data> regions. No test covers tool results: … |
| AIDEV-02 | High | DOWNGRADED | Medium | **[Correction]** The mechanism and the silence are real, but "zero review" overstates it. Confirmed on this tree: (1) tools.go:289 lets the gate pass when Skipped is set. A scratch DB-backed test with maybeRunCritic(waited=true) recorded {RoundCount:0 Skipped:contention} and wrote 0 SSE bytes, and advance_stage critique->finalize then returned … |
| AIDEV-03 | High | DOWNGRADED | Medium | **[Correction]** Every factual claim holds on this tree and I reproduced them on a private Postgres 18.4 with the built server. conversationView has no field for the parked resolutions. GET /conversations/{id} returned only id, locale, pendingStage, prdId, stage, status, updatedAt and workspaceId. I injected the meta a resolve_finding call … |
| AIDEV-04 | High | CONFIRMED | High | Reproduced live on a private Postgres 18.4 with the built insideout-mcp over stdio and a single user token. tools/list returned 52 tools including proposal_decide, commit, signup_mode, invite_create and invite_revoke. In one session, propose (with an item) followed by proposal_decide {decision:accepted, apply:true} succeeded … |

---

## Scope

AI-developer lens on `server/internal/agent`, the LLM client (Anthropic Messages and OpenAI-compatible Responses), prompts and the `<prd_data>` fence, the critic and finalize gates, stage handling, SSE, the run queue, circuit breaker and reaper, `server/internal/mcpserver`, `server/cmd/insideout-mcp`, `SKILL.md` and `skills/`. Tree: clean worktree at main @ 9f0b1c7. Read-only; the stream-parser claims were confirmed by probe tests run in a scratch copy of the server tree with no network. No database cluster was started.

## Scores

| Aspect | Score | Why |
|---|---|---|
| Prompt trust boundary | 6.5 | Fence is applied to every system-prompt surface and planner message, but not to tool results or MCP output |
| Stage and finalize gates | 7.0 | Server-side, human-confirmed, forward-only; a skipped critic silently opens the gate |
| Provider/stream robustness | 5.5 | Good bounds and retry taxonomy; parsers accept truncated tool calls and premature EOF |
| MCP tool policy | 4.5 | One flat token; the "agent vocabulary" is advisory prose, not enforced |
| Bilingual promise | 6.0 | zh/en prompts exist; zh labels and a zh-only fence clause leak into en |
| Offline coach honesty | 7.5 | Offline flag in SSE and client banner; template critic records a clean round |
| Run bounds (deadline, queue, breaker, reaper) | 7.5 | Deadlines, iteration cap, idle watchdog, detached persistence, breaker in SQL |
| Overall | 6.0 | Well-engineered loop with real trust-boundary and honesty gaps |

## What is good

- The fence is a real mechanism, not a slogan: `server/internal/agent/prompt_fence.go` neutralises tag-like text inside `<prd_data>` and `prompt_fence_surfaces_test.go` enumerates the surfaces (stored findings, planner PRD text, expand node text).
- Stage changes are proposals: `tools.go` `advanceStage` only parks `pendingStage`; `stage_confirm.go` and `ConfirmConversationStage` (row lock, `meta->>'pendingStage'` match) apply it only on a human POST. The model cannot self-promote.
- The critic runs in fresh context with a forced tool, drops defect quotes it cannot verify (`critic.go` `parseCriticFindings`), and `critic_merge.go` never lets the findings list shrink.
- Bounds are explicit: 12 min turn deadline, `maxToolIterations=12`, 10,000-byte message cap, 90 s idle watchdog, one retry only before any delta streamed, per-conversation lock, permit pool, detached terminal persistence (`coach.go`, `coach_loop.go`, `dispatch.go`, `coach_fail.go`).
- Offline mode is disclosed rather than disguised: `template.go` flags `message_start` and the client shows `coach.offline`.

## Findings

Worst first. All are confirmed by reading code; AIDEV-05 and AIDEV-06 were also confirmed by running probes.

1. **High, `get_prd` result is unfenced.** `server/internal/agent/tools.go:148: b, err := json.Marshal(map[string]any{"title": prd.Title, "sections": prd.Sections})`. PRD text is user-editable (and, via sharing, writable by others), and the agent that reads it holds `update_section` and `advance_stage`. Every other surface is fenced; this tool result is the one path the fence test does not enumerate.
2. **High, a skipped critic opens the finalize gate.** `tools.go:289: if cs.RoundCount == 0 && cs.Skipped == "" {`. When the critic is skipped for contention or output failure, `Skipped` is set and finalize proceeds with zero review. No SSE event or API field tells the human the PRD was never critiqued; the stage confirmation card looks identical.
3. **High, parked finding resolutions are invisible to the confirming human.** `server/internal/api/conversations.go:29: PendingStage string` is the only pending state exposed. `pendingFindingResolutions` count as resolved in the gate and are applied on confirm ("human-authored by construction"), yet the human never sees them. `_declineStage` in `prd_page.dart` is local-only and the stage is re-set from the server on reload.
4. **High, MCP tool policy is advisory.** `server/internal/mcpserver/product_tools.go:176: addTextTool("proposal_decide", "Human accept/reject of an agent proposal (owner or admin; ...` sits in the same flat registry, under the same token, as `propose` (line 196), `commit`, `signup_mode` set (`admin_tools.go:45`), `invite_*`, deletes, and password-taking `auth_*`. `store.DecideProposal` checks owner or admin only, so an agent running as the owner can propose and accept its own proposal. `SKILL.md` says this is "theirs" and commits are human-only, which nothing enforces.
5. **Medium, truncated tool calls and early EOF are accepted as complete.** `anthropic_parse.go:140` honours `stop_reason=max_tokens` only on the text path; a `tool_use` cut at the 4096 cap returns partial JSON args with `Truncated=false`, and a stream that ends without `message_stop` returns a normal turn. `responses.go` ignores `response.failed` and premature EOF. Probes confirmed all three.
6. **Medium, Responses parallel tool calls merge arguments.** `responses.go:259: case "response.output_item.added":` overwrites tool id and name while arguments keep appending, and the request does not set `parallel_tool_calls:false`. A probe produced `{"kind":"problem"}{"text":"x"}` as one call.
7. **Medium, "attested" facts prove less than they say.** `ledger.go:243: if quoteIsGrounded(args.Quote, msg)` grounds only the quote (70 percent bigram overlap, no minimum length); `Text` is never compared to it. The finalize gate counts such facts, and "done" is model text (`coach.go:331 endsWithMarkerLine`, `NO OPEN QUESTIONS`).
8. **Medium, MCP outputs carry untrusted text unlabelled.** `agentcontext.Assemble` and the roadmap evidence tools emit raw PRD titles, sections, idea and GitHub text to a tool-enabled client with only a `vocabulary` map, no data/instruction marking.
9. **Low, bilingual leaks.** `prompt_fence.go:10` is Chinese-only and is embedded in en prompts; `prompts.go:120` (`章节=`), `:146` (`读者 %s：`) and `原话` leak into en; `prompts_en.go:102` says "12 characters or fewer" for English titles.
10. **Low, `propose` does not declare `items`.** `product_tools.go:201` reads `items` from arguments (and `TestProposeKeepsAtSignInItemTitle` drives it), but the tool schema declares only project_id, kind, summary, detail, so a model reading the schema never sends structured items.
11. **Low, misc.** `taxonomy.go:166` maps any 400 containing "context" to `ErrContextLength`; the critic reads meta before the LLM call and writes after, racing `ConfirmConversationStage`; the reaper stale window (10 min) is shorter than the turn deadline (12 min); the template critic records a clean round as if a review happened.

## Lead verification

| Lead | Result | Evidence |
|---|---|---|
| L1 time bomb | Not re-filed; no sibling in this lens | Only production wall-clock fallback is `agentcontext.go:136`, harmless in production; other 2026 dates are fixed inputs |
| L3 / L4 suite results | Outside lens for the suites; agent package behaviour confirmed by probes | See AIDEV-05, -06 |
| L17 review-3 Highs "Fixed" | Spot-checked, extended | Fence, merge-never-shrinks, finalize gate and offline flag hold; the gate has the skipped-critic hole (AIDEV-02), the resolution path is invisible (AIDEV-03) |
| L2, L5-L16, L18 | Outside lens | Not examined |

## Recommendations

1. Add an enforced agent profile for MCP: a token scope that omits `commit`, `proposal_decide`, `signup_mode` set, `invite_*`, deletes and password auth tools; make `DecideProposal` reject an actor that authored the proposal.
2. Fence `get_prd` output with `fenceData` and add the fence clause to the tool description; extend the surfaces test to cover tool results.
3. Surface `critic_state.Skipped` as an SSE event and API field, and make finalize confirmation say "no critic review ran".
4. Expose parked finding resolutions in `conversationView`, render them on the confirm card, and add a server-side decline.
5. Require `message_stop`/`response.completed` in both parsers, treat `max_tokens` on a tool call as truncated, and set `parallel_tool_calls:false`.
6. Label MCP outputs that carry user-written text as data, and declare `items` in the `propose` schema.
7. Localise the fence clause and the prompt labels per locale; fix the en title-length rule.
8. Bind facts to their quote (require the text to be supported by the quote) and make "done" a server check rather than a marker line.

## What this role would do first

Ship the enforced MCP agent tool policy: a scoped agent token that can read, propose and update the working draft but cannot decide proposals, commit, change signup mode or invites, so the human-only vocabulary in `SKILL.md` becomes server truth. In the same change, ship the prompt contract that every tool result and MCP output carrying user-writable text is fenced and labelled as data, starting with `get_prd`, with a test that enumerates tool results the way the surfaces test enumerates prompts.
