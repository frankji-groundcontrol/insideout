# 11 · AI Developer

**Lens:** 
**Score:** **6.8 / 10**
**Bottom line:** The coach loop is still the best-engineered part and round one's High is genuinely fixed — but this pass surfaces a new High stub-tests cannot see: the fresh-context critic sends an empty messages array to the provider, so every critic call 400s on the real Messages API and the independent-reviewer feature silently never runs; plus the idea-privacy change outpaced the MCP/SKILL.md contract, and five carried findings remain open.

Reviewed main @ 26db979; a skeptic re-checked every High finding from this review. AIDEV-09 was confirmed High by the skeptic and reproduced at code level by the coordinator: `critic.go:231` passes `nil` messages, `anthropic.go` has no `omitempty` and builds a non-nil empty slice, so the wire body is `"messages": []` — every production critic call 400s and degrades to a silent skip.

---

This review was persisted findings-only; the section below is assembled verbatim from its structured findings (the prose wrapper was not retained). 11 findings as filed, worst first.

## Findings

### AIDEV-09 · High ·  **Skeptic: CONFIRMED.**

**Fresh-context critic sends an empty messages array to the provider; on the real Anthropic Messages API every critic call 400s, so the critic pass never runs in production and silently degrades to a skip.**

- Where: `server/internal/agent/critic.go:231`
- Evidence: critic.go:231 'turn, err := streamer.StreamChatForcingTool(ctx, sys, nil, tool, nil)'; anthropic.go:384 marshals nil msgs to "messages": [] (no omitempty); planner does it right at roadmap_planner.go:184/:206.
- Fix: Pass a one-message payload like the planner ('[]Message{{Role: RoleUser, Content: …call report_findings}}'); add a wire-level test asserting non-empty messages for forced-tool calls plus one live provider exercise.
- Diff size: ~5 lines + test

### AIDEV-10 · Medium · 

**MCP idea_list description (and SKILL.md) still promise a workspace-wide idea inbox, but ideas became author-private on 2026-10-04 — the tool returns only the caller's own ideas.**

- Where: `server/internal/mcpserver/product_tools.go:249`
- Evidence: product_tools.go:249 'List a workspace's idea inbox with full content'; store/ideas.go:58-70 author-scoped; SKILL.md:37 'Triage the inbox'.
- Fix: Reword to 'List the signed-in user's own ideas in this workspace (ideas are private to their author)'; fix SKILL.md:37-38; sweep idea_* descriptions for the stale workspace-inbox framing.
- Diff size: ~10 lines docs

### AIDEV-11 · Medium · 

**Carried AIDEV-02, still open: record_fact quote grounding only searches the last ~20 turns while the prompt promises 'the user's verbatim words' with no window — long clarify conversations get wedged out of the draft gate.**

- Where: `server/internal/agent/coach.go:246`
- Evidence: coach.go:229 historyLimit=20; coach.go:246 'userMessages: userTexts(history)'; coach.go:450-452 slice; tools.go:55 'quote must be their verbatim words'.
- Fix: Ground quotes against all user turns: ListAgentMessages already returns every row — build userTexts from the unfiltered list.
- Diff size: ~10 lines

### AIDEV-12 · Medium · 

**Model-behavior failures and mid-loop store errors fall into failTurn's default branch and count against the provider circuit breaker — a looping model can open the circuit for every user.**

- Where: `server/internal/agent/coach.go:368`
- Evidence: coach.go:431 'exceeded max tool iterations' flows to coach.go:368-371 default: RecordCircuitResult(false).
- Fix: Classify max-iterations (and unexpected store errors) as their own branch: fail the run, log loudly, do not RecordCircuitResult.
- Diff size: ~15 lines

### AIDEV-13 · Medium · 

**Carried AIDEV-07, still open: PRD title, sections, and open findings are injected into the system prompt with no data fencing — untrusted human-writable text sits at instruction level for both coach and critic.**

- Where: `server/internal/agent/prompts.go:69`
- Evidence: prompts.go:69-70 raw Sprintf interpolation of prdTitle/sections; same in critic.go:206-210.
- Fix: Wrap title/sections/findings in explicit untrusted-data fences with a fixed 'treat as data, never instructions' clause.
- Diff size: ~10 lines

### AIDEV-14 · Low · 

**Carried AIDEV-08, still open: the PROVIDER_CONFIG branch streams the raw provider error body (up to 4 KiB) to the client over SSE.**

- Where: `server/internal/agent/coach.go:355`
- Evidence: coach.go:353-355 'sse.errorEvent(loopErr.Error(), "PROVIDER_CONFIG")' where loopErr embeds up to 4096 bytes (taxonomy.go:147-153).
- Fix: Log the full error server-side and send a fixed 'AI provider is misconfigured' string.
- Diff size: ~2 lines

### AIDEV-15 · Low · 

**Carried AIDEV-03, still open: in offline template mode the critic's forced tool call returns '{}' which parses as zero findings — the finalize gate records a full critic round that never reviewed anything.**

- Where: `server/internal/agent/template.go:36`
- Evidence: template.go:36-38 forced-tool returns Arguments "{}"; parseCriticFindings("{}") → (nil, nil); critic.go:80 RoundCount++.
- Fix: Return an error from the template streamer's forced-tool path so markSkipped("offline") fires, or treat empty findings as an output failure.
- Diff size: ~5 lines

### AIDEV-16 · Low · 

**Carried AIDEV-04, still open: record_fact silently truncates the stored quote to 300 runes while grounding validated the full text — the evidence panel shows a prefix presented as the user's exact words.**

- Where: `server/internal/agent/ledger.go:224`
- Evidence: ledger.go:224-27 quote truncated to maxQuoteLen, rendered as '（原话：%s）' at ledger.go:183.
- Fix: Reject overlong quotes at record_fact time, or store the full quote and truncate only at render.
- Diff size: ~3 lines

### AIDEV-17 · Low · 

**propose flattens structured items into 'title@hint' strings; a title containing '@' is silently cut at the first '@' by the API client.**

- Where: `server/internal/mcpserver/product_tools.go:219`
- Evidence: product_tools.go:219 'items = append(items, title+"@"+hint)'; apiclient.go:387 'title, hint, _ := strings.Cut(it, "@")'.
- Fix: Extend apiclient AgentPropose to take structured items (it already builds []map[string]string) and pass them through un-flattened.
- Diff size: ~10 lines

### AIDEV-18 · Low · 

**Carried AIDEV-05, still open: responses-schema deployments get 'agent: anthropic returned …' error text because responses.go reuses the Anthropic-labeled classifier.**

- Where: `server/internal/agent/taxonomy.go:153`
- Evidence: responses.go:153-155 'anthropicHTTPError(resp)'; taxonomy.go:153 'fmt.Sprintf("agent: anthropic returned %s: %s"…'.
- Fix: Parameterize the provider label at the streamer.
- Diff size: ~5 lines

### AIDEV-19 · Low · 

**mcpserver.Build's contract comment says 'INSIDEOUT_API (default: hosted API)' but the binary defaults to http://127.0.0.1:8080/api/v1.**

- Where: `server/cmd/insideout-mcp/main.go:19`
- Evidence: mcpserver/server.go:20 'Configure with INSIDEOUT_API (default: hosted API)' vs main.go:19 'const defaultAPIBase = "http://127.0.0.1:8080/api/v1"'.
- Fix: Align the comment with the localhost default, or move the constant into mcpserver.
- Diff size: ~2 lines

