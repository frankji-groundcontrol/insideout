# 20 · Feature Implementation

**Score:** **7 / 10** · 8 findings (4 High+)

**Bottom line:** The shipped vertical slice — idea inbox → PRD coach (SSE, offline fallback) → critic with quote-grounding → roadmap tree with deadlines/pressure/one-human-owner → human Commit with diff → GitHub evidence via guide-matched webhook → audience views/readiness/export, with genuine 1:1 CLI/MCP parity — is real, tested, and honest about what it is; the parked Wave-4 halves (branches/merge, per-project roles, personas, lenses, feedback invitations) are known and ledgered. What this review adds: two un-ledgered defects on documented paths (the coach/critic/planner prompts are Chinese-only against PRODUCT.md's first-class-English commitment, and ConvertIdea silently drops the idea body it promises to preserve as coaching context), the never-exercised hosted mail funnel (magic-link/code/reset cannot deliver to an outside inbox today), and one build-next gap — the validation half of the product thesis has zero implementing surface. 7.0: I would use this product myself in Chinese today; I would not point an English-speaking team or a paying workshop at it yet.

---

# Review 3 — Feature Implementation (main @ 754c502)

**Scope:** PRODUCT.md read feature by feature; every promise traced to implementing code across `server/`, `client/`, `cmd/`, `skills/`, `scripts/`; prior reviews cross-checked against `docs/review/2026-10-10-disposition-ledger/` so parked rows (FEAT-06..12, PM-10..18) are classified here but **not** re-filed. **Ran:** `go test -count=1 ./internal/roadmaptime/... ./internal/readiness/... ./internal/audienceview/... ./internal/prdcommit/... ./internal/agentcontext/... ./internal/presence/... ./internal/mcpserver/...` — all 7 packages `ok` (uncached). Everything else is read/grep evidence cited inline.

## Feature-by-feature classification (lead 14)

| PRODUCT.md promise | Status | Implementing code / gap |
|---|---|---|
| Idea capture (one-step, private) | **IMPLEMENTED** | `api/ideas.go` routes; privacy migration `20260722120000…`→`20261004120000_ideas_private_by_default.sql` |
| Idea → PRD conversion preserving title **and body** | **PARTIAL (defect)** | `store/ideas.go:235` carries title only — FEAT-02 below |
| Coaching loop (stages, SSE, offline fallback) | **IMPLEMENTED** (as the *old* fixed model) | `agent/coach.go`, `agent/prompts.go`, `api/conversations.go`; 4 stages + 8 sections — adaptive contract parked (FEAT-07/PM-15, not re-filed) |
| Critic loop (fresh-context, ranked, resolve/override gate) | **IMPLEMENTED** | `agent/critic.go:21-93`; finalize gate reads persisted findings, never model say-so |
| Fact ledger (quotes grounded, statuses) | **IMPLEMENTED** | `agent/ledger.go` `recordFact`/`markAssumption`; `quoteIsGrounded` CJK bigram matcher (ledger.go:131) |
| Fact actions (confirm/correct/remove/discuss) | **DOCS-ONLY** | Parked PM-17 |
| Personas (core/key/later, 4 fields, labeling) | **DOCS-ONLY** | Parked FEAT-10/PM-14 |
| Roadmap tree, acceptance, one human owner | **IMPLEMENTED** | `store/roadmap.go`; migrations `20260722120000`, `20261005120000` |
| AI build/expand (draft→confirm) | **IMPLEMENTED** | `api/roadmap_ai.go` + `agent/roadmap_planner.go`; writes nothing until confirm |
| Progress view (Now/NeedsDeadline/Next≤3/Done) + pressure | **IMPLEMENTED** | `roadmaptime.Assemble` (roadmaptime.go:77), `Pressure` (line 21); client `roadmap_progress_view.dart` |
| Deadlines gate Now | **IMPLEMENTED** | in-progress without deadline → `NeedsDeadline` band |
| Dependencies + rebaseline | **DOCS-ONLY** | **New finding FEAT-05** — no edges exist; not in the parked ledger |
| Overdue continue/transfer/abandon-with-reason | **PARTIAL** | Parked FEAT-09/PM-18 |
| Commit versions (immutable, diff, audience, unresolved, note) | **IMPLEMENTED** | `api/prds.go` `handleCommitPrd`; `prdcommit.Diff`; `store/prd_commits.go` (decision_note column) |
| Decision Log (merges, overrides, alternatives) | **PARTIAL** | One free-text note on Commit only; parked PM-10 |
| Audience views (4 projections, server-side) | **IMPLEMENTED** | `audienceview.go`; `api/prd_view.go:42-48` returns projection + readiness + latestCommit |
| Readiness per audience, no universal % | **IMPLEMENTED** | `readiness.go` gaps with priority + carry-into-commit |
| Revisions/snapshots (working version) | **IMPLEMENTED** | `store/prd_revisions.go` |
| Product Branch / Merge | **DOCS-ONLY** | Parked PM-11 (grep confirms zero surface) |
| Per-project roles | **DOCS-ONLY** | Parked FEAT-06/PM-12; PRODUCT.md self-discloses workspace-scoped v1 |
| Presence + cursors | **IMPLEMENTED** | `presence.go` registry + SSE; client `roadmap_page.dart`/`roadmap_canvas.dart` |
| Human Focus / team Now | **DOCS-ONLY** | **New finding FEAT-06** |
| Roadmap lenses (Decision/Product/Delivery/Validation) | **DOCS-ONLY** | Parked FEAT-08/PM-16; agent-context modes are the one lens that exists |
| Exceptions-first lead home | **DOCS-ONLY** | Parked PM-16; `dashboard/home_field.dart` is a workspace list |
| GitHub: link, manual sync, webhook, guide matching | **IMPLEMENTED** | `api/github.go`, `github_webhook.go` (HMAC-first, per-owner RLS-respecting sync), `guide/` |
| Evidence kinds commit/PR/deploy→activity/review/implementation/release | **IMPLEMENTED** | `attachGuideEvidence` (github_webhook.go:92-239) |
| Validation evidence (feedback, metrics, validated state) | **DOCS-ONLY** | **New finding FEAT-04** — the loop's second half |
| Failed deployments append evidence | **PARTIAL** | FEAT-07 — creation-only mapping, refinement deferred in code comment |
| Unmatched activity stays visible | **IMPLEMENTED** | `syncGithubProject` appends every commit to the timeline regardless of guide match |
| Agent vocabulary (context/checkpoint/propose/decide/versions) | **IMPLEMENTED** | `api/agent_routes.go`, `agentcontext` |
| CLI + MCP parity | **IMPLEMENTED** | Verified verb-by-verb incl. auth `--code`/`--magic` (cli_auth.go:73-128) vs `auth_*` tools; 52 MCP tools |
| Mail flows (register/code/magic/forgot/reset, invitations) | **PARTIAL (never live)** | Handlers complete (`auth_email_flows.go`), dev-mode degradation honest — FEAT-03: unexercised against the provider |
| Install/setup (scripts, bootstrap, setup TUI, skill) | **IMPLEMENTED** | `scripts/install.sh`/`setup.sh`, `internal/setup`, `skills/insideout-setup` |
| Bilingual en/zh | **PARTIAL** | UI fully bilingual; **LLM prompts Chinese-only** — FEAT-01 |

## Findings (worst first)

| ID | Sev | Bucket | Where | Summary |
|---|---|---|---|---|
| FEAT-01 | High | FIX | `server/internal/agent/prompts.go:43` | Coach/critic/planner prompts are Chinese-only; PRODUCT.md:439-441 makes English first-class. No language parameter exists in `internal/agent`. An English-first user gets a Chinese coach behind a bilingual UI. **Fix:** thread the client locale into conversation start, select/direct prompt language; red test for the en prompt. |
| FEAT-02 | High | FIX | `server/internal/store/ideas.go:235` | `insertPrd(ctx, tx, …, idea.Title)` — the body is scanned and dropped; PRD starts `emptySections()`, conversation starts empty. PRODUCT.md:69-72 promises "preserves the original title and body as coaching context." **Fix:** seed a first conversation message with the body verbatim inside the convert transaction. |
| FEAT-03 | High | HARDEN | `server/internal/mail/mail.go:52` | Mail-off logs and returns nil; hosted key unset; default From only reaches the Resend account's own email (environment.md:104). Magic-link/code/reset are log-only today; the invite funnel survives only because the code is returned inline (admin.go:93-95). **Fix:** set key + verified sender on the host, run one live invite→register→code→magic→reset pass; consider hiding (not annotating) emailed options when `mailEnabled=false`. |
| FEAT-04 | High | ADD | `server/internal/api/github_webhook.go:140` | Evidence kinds end at release; no validation evidence source, no implemented/released/validated distinction — PRODUCT.md:26's `→ validation → next version` has no code. **Build this next:** `validation` evidence kind + human-entered observation on a leaf + validated state with reason. |
| FEAT-05 | Med | IMPROVE | `server/db/migrations/` (roadmap set) | No dependency edges exist anywhere, so "dependency delay → affected path → time-Diff rebaseline" (PRODUCT.md:301-303) cannot be built on top; not in the parked ledger. First slice: depends-on edges + blocked indicator in `Assemble`. |
| FEAT-06 | Med | IMPROVE | `server/internal/agentcontext/agentcontext.go:18` | Focus is only an agent-context parameter; no human focus pointer, no one-main-Focus-per-session, team Now is just node status + presence (PRODUCT.md:246-249). Smallest slice: per-session focus reusing the presence TTL. |
| FEAT-07 | Low | IMPROVE | `server/internal/api/github_webhook.go:164` | "success/failure refinement belongs to a later deployment_status slice" — failed deployments never append evidence (PRODUCT.md:326-327 promises they do). Residual of dispositioned CC-01; map deployment_status failures as appended activity rows. |
| FEAT-08 | Low | IMPROVE | `README.md:35` | Advertises "50 tools"; the registry defines 52 (counted from `addTextTool`/`addTool` registrations). Update or derive the number in the docs check. |

## Strengths

- **Time-first roadmap is exactly the doc:** `roadmaptime.go:77` `Assemble` implements Now-dominant / at-most-three deadlined Next / Done-with-owner, and `NeedsDeadline` for deadline-less in-progress work — pure, clock-injected, tested.
- **The critic is engineered, not prompted:** fresh-context pass bounded at 2 rounds (critic.go:32), quote-grounding drops ungrounded defect findings rather than the batch (critic.go:325-327), and under load it degrades with a recorded skip instead of trapping the user.
- **Human authority is structural:** every AI write is draft→confirm (`roadmap_ai.go` routes, `mcpserver/draft_tools.go`); Commit is a separate human route with an immutable diff (`prd_commit_lock_test.go` pins it).
- **Parity is real:** all 52 MCP tools are thin projections of the same `/api/v1` the CLI drives, including the full auth surface with the token kept out of transcripts (`auth_tools.go:42`).
- **The webhook stays inside RLS** by resolving projects and re-running the per-owner sync (github_webhook.go:29-34), signature verified before any parse (line 47).

## Stop advertising until fixed / build next

**Stop advertising:** (1) the bilingual coaching experience — until FEAT-01 lands, say the coach is Chinese-first; (2) "deployments attach as release evidence" should be scoped to deployment-creation until FEAT-07; (3) any hosted-demo implication that emailed sign-in works — FEAT-03.

**Build next:** the validation half (FEAT-04). It completes the product thesis's differentiating promise, is the cheapest wedge into the parked persona/feedback work, and every other candidate (branches, roles, lenses) already has a parked plan waiting on it.

## What this role would do first

Run the lead-15 funnel check against the hosted stack: `curl -s $HOST/api/v1/auth/config` — if `mailEnabled` is false, decide whether to configure the mail key + verified sender now (unlocking the first real-user pass) or hide the emailed sign-in options in the client.
