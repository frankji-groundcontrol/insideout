# 50 · PRODUCT.md promise register (lead 14, run)

**Why this exists:** COMP-01 (High). Review 3's feature-implementation
dimension promised a per-advertised-feature IMPLEMENTED / PARTIAL / DEAD /
DOCS-ONLY classification and never delivered it: coordinator lead 14 was
marked not-run, and the five partial gap-lists (architect roll-up, FULL-03,
CQ-02, SEC-04, TEST-08) disagreed — autosave sat in exactly one of five.
This register is lead 14 run as the remediation's first artifact (wave 4).

**Stamped at:** `rv3/w4-promise-register` base `2c5631b` (2026-10-11) — main
after remediation waves 1–3. The review's claims were made at `754c502`;
where a wave already moved an answer, the row says so.

**Method:** PRODUCT.md 64–390 read top to bottom; every promise verified
against the tree with greps run this session (named per row). Stamps:
**IMPLEMENTED** (working code end to end) · **PARTIAL** (named half missing)
· **DOCS-ONLY** (promise exists in prose, zero code) · **DEAD** (advertised,
code removed — none found). Derived delivery defaults (391–407) are excluded
by PRODUCT.md's own "not separate product commitments"; brand commitments are
out of the walked range except the one row the review confirmed broken.

## Canonical journey (PRODUCT.md 64–81)

| Promise | Stamp | Verified surface / absence |
|---|---|---|
| One-step idea capture, private to its author by default (PRODUCT.md:68) | **IMPLEMENTED** | `api/ideas.go` routes; privacy migration `20261004120000_ideas_private_by_default.sql` + `20261010120000_ideas_require_membership.sql` |
| Start-progress creates the Project and preserves title **and body** as coaching context (PRODUCT.md:69-71) | **IMPLEMENTED** | Title → `insertPrd`; body → first conversation message `store/ideas.go:252-253` via `insertFirstUserMessage` (`agent_conversations.go:53`). Was FEAT-02 at `754c502`; wave 1 fixed it |
| Existing project / GitHub repository entry without a retrospective Idea (PRODUCT.md:73) | **IMPLEMENTED** | Manual project create + repo link and manual sync `api/github.go:63-71` (`syncGithubProject`), client `project_github_card.dart` |
| Git required when code work becomes someone's main Focus in team Now (PRODUCT.md:79-81, 309-310) | **DOCS-ONLY** | No gate anywhere; hangs off the unbuilt Focus surface (row PRODUCT.md:247-249) |

## The Coach and the first workable version (PRODUCT.md 83–150)

| Promise | Stamp | Verified surface / absence |
|---|---|---|
| Coaching loop: SSE turns with offline template fallback (PRODUCT.md:87) | **IMPLEMENTED** | `agent/coach.go` SSE, `agent/template.go` offline replies; offline identity disclosed in-band (`sse.go:56-60`, wave 2); stage proposals human-ratified via `conversations.go:16` + client `coach_panel.dart:43` (wave 1, was USER-01) |
| Adaptive depth: no fixed question count, no forced sequence (PRODUCT.md:87-90) | **PARTIAL** | Clarify discipline is adaptive since wave 3 (`prompts.go:52`: 不设固定数量/顺序), but the session still runs the fixed four-stage machine over the eight storage sections (`prompts.go` StageDraft writes 全部 8 个章节; COMP-02 re-scoped those to the storage scaffold) |
| Every question carries priority, reader, why-now (PRODUCT.md:92-97) | **IMPLEMENTED** | `prompts.go:53` — 优先级（必须现在澄清 / 本版应澄清 / 之后再验证）+ who the answer serves + why now |
| Grouping allowed; one-at-a-time only when the answer branches (PRODUCT.md:99-100) | **IMPLEMENTED** | `prompts.go:52` — 相互独立的问题可以合并成一组；只有答案改变下一分支时才单独先问 (wave 3) |
| "Form a version now" any time; gaps marked; editable version (PRODUCT.md:102-104) | **IMPLEMENTED** | Gap disclosure `readiness/readiness.go` + `api/prd_readiness.go:12`; suggested unresolved items carried into the Commit; drafts stay editable |
| Critical product argument before the first version (PRODUCT.md:107-119) | **IMPLEMENTED** | `prompts.go:52` asks what-problem / who / current alternatives / why-now; the doc's own "critical questions, not mandatory headings" covers era/team/future-bet depth |
| Structured personas: core/key/later, four fields, value exchanges, grounded/synthesized/assumed labels (PRODUCT.md:121-134) | **DOCS-ONLY** | `grep -rni persona` hits only copy strings (`readiness.go:47`, `export/render.go:20`, `audienceview.go:39`); the `users` section is free text |
| Severity-ranked critique; blocking/major individual with keep-as-is; minors folded in a batch (PRODUCT.md:136-143) | **IMPLEMENTED** | `critic.go` ranked parser + `blocking/major/minor` validation (`critic.go:321`), blocking gate (`critic.go:199`), resolve/override statuses |
| Independent AI review may be unavailable: disclosed, retryable, no trap (PRODUCT.md:143-147) | **IMPLEMENTED** | Under load the critic degrades with a recorded skip, not a stall (`critic.go:17-18,41`) |
| Correct the Coach by editing directly or continuing (PRODUCT.md:149-150) | **IMPLEMENTED** | PRD section update routes any time; conversation continues |

## One PRD core, multiple audience views (PRODUCT.md 152–174)

| Promise | Stamp | Verified surface / absence |
|---|---|---|
| One human-confirmed primary audience per version; Coach may recommend; secondary audiences attachable (PRODUCT.md:156-158) | **PARTIAL** | Primary audience on every Commit (`prd_commits.go:34`, `20260821150000_prd_commits.sql:11`) + client `audience_selector.dart`; no secondary-audience surface (`grep secondary` → 0) |
| The four projections (Decision / Management / Delivery / Co-creation & validation) (PRODUCT.md:160-169) | **IMPLEMENTED** | `audienceview.go` projection map (audiences listed `api/prd_view.go:72`); served by `api/prd_view.go:23-40` |
| Projections of one core; audience-specific readiness; no universal completeness score (PRODUCT.md:171-173) | **IMPLEMENTED** | `readiness.go` per-audience rules with priority gaps — no percent score exists |
| Authorized readers switch views without gaining permissions (PRODUCT.md:174) | **IMPLEMENTED** | One `requireAuth` route serves all audiences (`api/prd_view.go`) |

## Facts, evidence, and privacy (PRODUCT.md 176–195)

| Promise | Stamp | Verified surface / absence |
|---|---|---|
| Coach records candidate facts without interrupting (PRODUCT.md:178-179) | **IMPLEMENTED** | `agent/ledger.go:260` record_fact with quote grounding (`quoteIsGrounded`, `ledger.go:139`) |
| Critical facts presented together for confirmation before a meaningful Commit (PRODUCT.md:179-180) | **DOCS-ONLY** | No facts route exists (`grep -rn facts server/internal/api` → 0); nothing surfaces the facts at commit time (parked PM-17 / AIDEV-09 slice) |
| Facts surface: interpretation, source, status, and confirm/correct/remove/discuss actions (PRODUCT.md:182-187) | **PARTIAL** | Recording + attested/assumed statuses implemented (`ledger.go:26-27,111`); the four user actions have no API and no client UI |
| Raw quotes private by default; views expose conclusions; no silent cross-PRD reuse (PRODUCT.md:189-195) | **IMPLEMENTED** | Facts and quotes live inside conversation meta; the projections pick sections only; no cross-PRD fact machinery exists to leak |

## Product version control (PRODUCT.md 197–225)

| Promise | Stamp | Verified surface / absence |
|---|---|---|
| Working version **autosaved** (PRODUCT.md:201) | **DOCS-ONLY** | `grep -rniE "autosave\|auto-save\|debounce" client/lib server/internal server/cmd` → **0**; saving is manual only (`prd_page.dart` save action, `prd_desk_actions.dart` onSave). The one promise four of five gap-lists missed |
| Commit: immutable human-confirmed checkpoint recording name, audience, change summary, unresolved items, Diff (PRODUCT.md:203-206) | **IMPLEMENTED** | Route `api/prds.go:16`; `store/prd_commits.go:34` CreatePrdCommit; `internal/prdcommit/diff.go`; immutability pinned by `store/prd_commit_lock_test.go` |
| Product Branch / Explore direction (PRODUCT.md:207-208, 222-225) | **DOCS-ONLY** | `grep -rniE "product[ _-]?branch" server/ client/lib` → **0**; no branch table among the 36 migrations |
| Diff / Compare changes: what changed and why (PRODUCT.md:209) | **IMPLEMENTED** | `prdcommit/diff.go` + change summary on the Commit; versions UI `versions_page.dart`, `versions_commit_details.dart` |
| Merge / Combine conclusion (PRODUCT.md:210-211) | **DOCS-ONLY** | Same zero-match grep; merge decisions have no machinery |
| Coach recommends Commit moments, never gates or performs them (PRODUCT.md:212-216) | **IMPLEMENTED** | The draft prompt proposes forming the version (`prompts.go:56`); the Commit route is a direct human action, readiness is a separate GET (`prd_readiness.go`) |
| Saved versions immutable; later changes form the next working version (PRODUCT.md:215-216, 220-221) | **IMPLEMENTED** | `prd_commit_lock_test.go` + the revisions model (`store/prd_revisions.go:43`) |
| Decision Log: who/why/evidence/alternatives for Commits, Merges, overrides, reversed decisions (PRODUCT.md:218-221) | **PARTIAL** | A free-text note on Commit (`prd_commits.go:34` decisionNote; `cli_version.go:124`; `product_tools.go:123`) and an immutable agent-proposal decision record (`proposal_decisions.go:13`, `20260822100000_proposal_decisions.sql`); no structured record or view, nothing for Merges/overrides/reversals |

## Collaboration and authority (PRODUCT.md 227–259)

| Promise | Stamp | Verified surface / absence |
|---|---|---|
| Workspace roles manage organization access (PRODUCT.md:229-230) | **IMPLEMENTED** | `workspace_memberships` admin/member (`20260720135751_workspaces.sql:25`); `users.system_role` (`20260825120000_invitation_signup.sql:8-9`) |
| Product authority per Project: Decision Owner / Driver / Contributor / Reviewer / Viewer (PRODUCT.md:230-241) | **DOCS-ONLY** | `grep -rniE "project[ _-]?role"` → **0**; self-disclosed at PRODUCT.md:431-434 |
| One current Driver per Product Branch, transferable (PRODUCT.md:242-243) | **DOCS-ONLY** | No branches exist (row PRODUCT.md:207-208) |
| Contributions remain proposals until accepted (PRODUCT.md:243-244) | **PARTIAL** | Agent proposals record with human decision (`api/agent_routes.go:20-21,213` accepted:false; `proposal_decisions.go`); human members edit the shared mainline directly in v1 |
| Each Now item has exactly one human responsible owner; an Agent cannot be the only owner (PRODUCT.md:246-248) | **IMPLEMENTED** | `owner_user_id uuid REFERENCES users` (`20261005120000_roadmap_acceptance_owner.sql:8`), surfaced as OwnerName by `roadmaptime` |
| Focus: non-exclusive pointer, one main Focus per session, team Now aggregates (PRODUCT.md:247-249) | **DOCS-ONLY** | Focus exists only as an agent-context parameter (`agentcontext.go:76`); no human focus surface (FEAT-06) |
| GitHub or Agent advance leaf evidence when acceptance is explicit; Driver confirms milestones; Decision Owner confirms results and Merges (PRODUCT.md:250-252) | **PARTIAL** | Leaf evidence auto-attach implemented (evidence rows below); milestone = human Commit; Decision-Owner and Merge confirmation have no machinery (rows PRODUCT.md:230-241, 207-208) |
| Scoped feedback invitations without joining; feedback retains author/time/context; respondents see only their own; owner may publish a synthesis (PRODUCT.md:254-259) | **DOCS-ONLY** | No feedback route anywhere (`grep -rni feedback server/internal/api server/cmd client/lib` → no product hits); the invitations table is workspace signup only |

## The shared Roadmap Graph (PRODUCT.md 261–303)

| Promise | Stamp | Verified surface / absence |
|---|---|---|
| One shared, versioned, hierarchical graph (PRODUCT.md:263-265) | **PARTIAL** | `roadmap_nodes` parent hierarchy (`20260722120000_roadmap_nodes.sql`) per project; the "tied to an explicit PRD Commit baseline" link is not modeled — no commit reference on the graph |
| Hierarchy expresses outcomes → work → evidence; up reveals Why, down acceptance/work/evidence (PRODUCT.md:267-270) | **IMPLEMENTED** | Canvas bands + tree (`roadmap_canvas.dart`, `roadmap_tree.dart`); acceptance and evidence columns on nodes |
| Progress view: Now dominant, at most three justified Next, Done with evidence (PRODUCT.md:272-273) | **IMPLEMENTED** | `roadmaptime.Assemble` (`roadmaptime.go:77-88`) + client `roadmap_progress_view.dart` |
| Canvas reveals the larger hierarchy and execution frontier (PRODUCT.md:273-274) | **IMPLEMENTED** | `roadmap_canvas.dart` with `roadmap_minimap.dart` |
| History shows versions, decisions, evidence (PRODUCT.md:274-275) | **IMPLEMENTED** | `versions_page.dart` (versions + Commit detail incl. the decision note) + `project_updates_feed.dart` (activity/evidence feed) |
| Lenses: Decision / Product / Delivery / Validation / Agent context; role, lens and permission stay separate (PRODUCT.md:277-289) | **PARTIAL** | The agent-context lens is implemented with mode-specific emphasis (`agentcontext.go:18,53-76`); the four human roadmap lenses have no parameter or view (`grep -rni lens` → only the `audienceview.go:16` comment). Audience switching already grants no permission, so the separation principle holds where a view exists |
| Product outcomes carry target time ranges (PRODUCT.md:291-292) | **PARTIAL** | Node deadlines only; no outcome-level time-range field |
| Deadlines explicit; an item without a deadline cannot enter Now (PRODUCT.md:292-293) | **IMPLEMENTED** | Deadline column + `NeedsDeadline` band (`roadmaptime.go:63-88`) |
| Pressure ladder normal → near → high risk → overdue (PRODUCT.md:295-297) | **IMPLEMENTED** | `roadmaptime.Pressure` (`roadmaptime.go:21-28`) |
| Overdue never silently closed; continue / change deadline / transfer / abandon with recorded reason (PRODUCT.md:299-301) | **PARTIAL** | Pressure surfaces overdue; no transfer or abandon-with-reason action (`grep -rniE "overdue\|abandon"` over product code → comments only); parked review-2 FEAT-09/PM-18 |
| Dependency delay shows the affected path and a proposed time Diff; human confirms the rebaseline (PRODUCT.md:301-303) | **DOCS-ONLY** | No dependency edges (`grep -rni depend` over migrations/store/client → section copy only) |
| Agents include deadline pressure when recommending (PRODUCT.md:303) | **IMPLEMENTED** | Agent context carries deadline + pressure (`agentcontext.go:138-139`) |

## Git, CLI, MCP, and Agents (PRODUCT.md 305–344)

| Promise | Stamp | Verified surface / absence |
|---|---|---|
| GitHub recommended after Idea save / Project create, skippable (PRODUCT.md:307-309) | **IMPLEMENTED** | Link card + manual sync (`api/github.go`; `project_github_card.dart`); nothing forces the connection |
| Matched GitHub activity advances leaf evidence: commit→activity, opened PR→review, merged PR→implementation, deployment→release (PRODUCT.md:312-318) | **IMPLEMENTED** | `planGuideEvidence` (`github_webhook_evidence.go:58-140`) maps all four; evidence kind set activity/review/implementation/release |
| Feedback and metric observations arrive through their own evidence sources, never as GitHub events (PRODUCT.md:320-321) | **DOCS-ONLY** | No validation evidence source exists at all (FEAT-04, wave 4); the never-as-GitHub-events half holds by construction |
| No Git event silently proves a higher outcome; unmatched activity stays visibly unassociated (PRODUCT.md:323-325) | **IMPLEMENTED** | Guide match required before leaf attach; unmatched commits still land in the updates feed (`syncGithubProject` → `SyncRepoCommits`, `github_webhook.go:160-188`) |
| Reverts and failed deployments append evidence; they do not erase history (PRODUCT.md:325-326) | **IMPLEMENTED** | `deployment_status` failure appends its own activity row (`github_webhook_evidence.go:96-115`, FEAT-07 landed wave 2); evidence rows are insert-only |
| Change Proposal for delivery-revealed product change; human accepts before the baseline changes (PRODUCT.md:326-328) | **PARTIAL** | The generic agent proposal + human decision flow exists (`agent_routes.go:20-21`, kinds structure/scope/priority at `:137-155`); no change-proposal surface showing impact on views/journeys/Roadmap (`grep change[ _-]?proposal` → 0) — a gap **no review list carried** |
| Web, GitHub, CLI, MCP, Agents on one source of truth; agent vocabulary context/focus/checkpoint/propose/version (PRODUCT.md:330-332) | **IMPLEMENTED** | All ride `/api/v1`: `agent_routes.go:18-21`, CLI dispatch `cmd/insideout/cli.go`, 52 MCP tools (41 `addTextTool` + 11 `addTool` registrations) |
| Compact focus-scoped agent context; brainstorming/implementation/review modes (PRODUCT.md:334-340) | **IMPLEMENTED** | `agentcontext.go` — mode-specific emphasis, focus-node context at `:76` |
| Agents can checkpoint and propose; cannot apply strategic changes, Commit, or Merge (PRODUCT.md:342-344) | **IMPLEMENTED** | Checkpoint/propose write drafts and proposal rows only (`agent_routes.go:19-21,213`); Commit is a human route |

## Completion and learning (PRODUCT.md 346–355)

| Promise | Stamp | Verified surface / absence |
|---|---|---|
| implemented / released / validated are distinguished (PRODUCT.md:348) | **DOCS-ONLY** | Node statuses are `locked/pending/in_progress/done` only (`20260722120000_roadmap_nodes.sql:17`); no validated state (FEAT-04 / REL-07, wave 4) |
| Completion links to verifiable evidence; human override only with a reason (PRODUCT.md:349-350) | **PARTIAL** | Evidence attaches to nodes (`AddRoadmapEvidence`, `github_webhook.go:145`); status changes record no reason |
| After release: compare outcomes with committed personas/assumptions/success criteria; begin the next version or Branch (PRODUCT.md:352-355) | **DOCS-ONLY** | The compare loop has no surface; depends on the persona and validation rows |

## Outside the walked range, confirmed by the review

| Promise | Stamp | Verified surface / absence |
|---|---|---|
| English and Simplified Chinese are first-class product languages (PRODUCT.md:441-442) | **PARTIAL** | UI catalogs complete en/zh (379/379 keys, review-3 measurement); every LLM prompt is Chinese-only (`prompts.go:43`; no locale/language parameter in `internal/agent`) — FEAT-01, wave 4 in flight when this register was stamped |

## The end-state loop, item by item (PRODUCT.md 357–383)

| Loop item | Status | Load-bearing gap |
|---|---|---|
| 1. private Idea or existing GitHub project, context preserved (PRODUCT.md:362-364) | **IMPLEMENTED** | — |
| 2. prioritized, explained, adaptive coaching; form-now with explicit gaps (PRODUCT.md:364-366) | **IMPLEMENTED** | prompts are zh-only (row PRODUCT.md:441-442) |
| 3. editable product argument with prioritized concrete personas (PRODUCT.md:366-368) | **PARTIAL** | personas are free text (row PRODUCT.md:121-134) |
| 4. choose an audience; view one fact core through the projection (PRODUCT.md:368-369) | **IMPLEMENTED** | — |
| 5. immutable Commit with Diff, summary, open validation items, Decision Log (PRODUCT.md:369-371) | **PARTIAL** | Decision Log is a free-text note only (row PRODUCT.md:218-221) |
| 6. Roadmap baseline: Now / justified Next / evidenced Done, dependencies, one human owner, deadlines (PRODUCT.md:371-373) | **PARTIAL** | dependency edges DOCS-ONLY (row PRODUCT.md:301-303) |
| 7. Decision Owner and Driver separated; baselines and Merges human-confirmed (PRODUCT.md:373-374) | **DOCS-ONLY** | per-Project roles and Product Branch/Merge (rows PRODUCT.md:230-241, 207-208) |
| 8. Git required at Focus; matched evidence without claiming unproven user value (PRODUCT.md:375-378) | **PARTIAL** | the requirement gate is DOCS-ONLY (row PRODUCT.md:247-249) |
| 9. Web and one Agent interface share Focus-scoped Roadmap context (PRODUCT.md:378-379) | **PARTIAL** | agent context yes; human Focus surface no (row PRODUCT.md:247-249) |
| 10. implemented / released / validated distinguished; next version without erasing history (PRODUCT.md:379-381) | **DOCS-ONLY** | no validated state or validation evidence (rows PRODUCT.md:348, 320-321) |

## Reconciling the five gap-lists

Every claim re-verified against this tree. "Confirmed" means the claim held
at `754c502` and still holds here unless a wave is named.

| List | Claim | Verdict |
|---|---|---|
| Architect roll-up (10-architect.md:36) | Product Branch/Merge, structured Decision Log, facts layer, personas, lenses, feedback invitations, dependencies all unimplemented; per-Project roles self-disclosed | **Confirmed**, one correction: "facts layer" is imprecise — recording is implemented (`ledger.go`); the missing half is the user-facing facts surface (the IMPLEMENTED-vs-PARTIAL distinction this register exists to make) |
| FULL-03 (12-fullstack.md:29) | autosave, structured personas, Product Branch/Merge, Decision Log, facts surface, roadmap lenses, dependency modeling, external feedback have no code and no dated issue; only the roles gap is admitted | **Confirmed** — all eight verified absent above; the only list of the five that carried autosave |
| CQ-02 (21-code-quality.md:40) | Product Branch/Explore/Merge, Decision Log as a surface, scoped feedback invitations, persona grounding labels, per-role Roadmap lenses unimplemented; critique ladder, deadline bands, audience readiness implemented | **Confirmed** on both halves (implemented set re-verified: `critic.go`, `roadmaptime.go`, `readiness.go`) |
| SEC-04 (22-security.md:26) | scoped feedback invitations, Product Branch/Merge + Decision Log, five Roadmap Lenses, per-Project roles unimplemented | **Confirmed** — all four stamps above |
| TEST-08 (23-testing.md:46) | Branch/Merge, Decision Log, scoped feedback invitations have zero implementation and therefore zero tests; the status paragraph disclaims only the roles gap; the four audience views are implemented | **Confirmed** — the PRODUCT.md status enumeration is this wave's TEST-08 slice |

Rows **no list carried** (this walk's additions, all verified above): the
Change Proposal surface (PARTIAL via generic proposals only), secondary
audiences (absent), outcome-level target time ranges (absent), the
Commit-baseline tie on the Roadmap graph (not modeled), and the
override-with-reason on completion (absent).

## What this hands forward

The DOCS-ONLY and PARTIAL rows are the seed for the `w4-gap-issues` slice
(one dated issue per parked promise, Branch/Merge first — the Decision Log,
personas and the facts surface all hang off it). Walk totals at this tree —
63 promise rows: **36 IMPLEMENTED · 13 PARTIAL · 14 DOCS-ONLY · 0 DEAD** —
so the ADD bucket of the remediation plan is now sized from a verified
inventory, which is what COMP-01 asked for.
