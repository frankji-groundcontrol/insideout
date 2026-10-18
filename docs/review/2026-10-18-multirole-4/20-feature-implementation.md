# 20 · Feature implementation

**Lens:** feature-implementation, product-promise-vs-code, surface-parity, bilingual-support, github-evidence-loop, agent-authority  
**Score:** **6.9 / 10**  
**Bottom line:** The advertised loop (idea, Coach, PRD, Commit, roadmap, GitHub evidence, presence) is built end to end and the gaps PRODUCT.md admits are tracked; the real defects are at the seams: the human-only Commit/decision boundary is a tool description, re-applying a proposal duplicates roadmap nodes (confirmed by running), and the new validation leg is web-only.

Tree: `main` @ `9f0b1c7`. Every High this lens filed went to an adversarial skeptic cluster; where the skeptic changed a severity or corrected a claim it is marked **[Correction]** below. The full register is [01-findings-register.md](01-findings-register.md).

## Skeptic verdicts on this lens's Highs

| ID | Filed | Verdict | Final | Reason |
|---|---|---|---|---|
| FEAT-01 | High | CONFIRMED | High | Reproduced live. I built insideout-mcp and drove it over stdio against a scratch server with one user token. tools/list includes both 'commit' and 'proposal_decide'. In that single session I called MCP `propose`, then `proposal_decide` with decision=accepted and apply=true on that same proposal, then `commit` with name … |
| FEAT-02 | High | CONFIRMED | High | Reproduced on a private Postgres with the real handler and RLS roles. Posting {decision:accepted, apply:true} twice to /agent/proposals/{uid}/decision returned 200 both times, each with a different appliedNodes id, and ListRoadmap then held 2 nodes with the same title. An accept, reject, accept sequence also left 2 nodes. … |

---

## Scope

Role: feature-implementation. Tree: `/private/tmp/insideout-review4` at main @ 9f0b1c7. I walked the README feature list, `PRODUCT.md`, `docs/architecture/product-subsystems.md`, the plans board and the review-3 promise register, then read the handler, store function, CLI verb, MCP tool and client screen behind each advertised item. One claim was confirmed by running code on a private Postgres 18.4 cluster (port 55446, CI-parity roles, stopped before finishing). Read-only on the tree; scratch copy under `/tmp/rv4-scratch/feature-implementation/`. No network beyond localhost, no `.env` read.

Overall: the loop the README sells is built end to end (idea, Coach, PRD, Commit, roadmap, GitHub evidence, presence). PRODUCT.md and the issue tracker are unusually honest about what is missing. The defects are at the seams: the "humans decide" boundary is a convention, re-applying a proposal duplicates nodes, the new validation leg stops at the web, and several advertised behaviours are narrower than their wording.

## Scores

| Aspect | Score | Why |
|---|---|---|
| Core loop (ideas, PRD, Commit, audience views, export) | 8.0 | Real code at every step; export labels are English-only for a zh-first app |
| Coach and critic | 7.5 | Locale-threaded prompts, ranked critic, fact ledger; offline mode is an echo, not a coach |
| Roadmap, time-first, validation | 7.5 | Progress, pressure, owner and the new validated state work; validation has no CLI/MCP verb |
| GitHub evidence loop | 6.5 | Webhook, HMAC, guide match work; commit timeline cannot read private repos; deploy "release" is premature |
| Agent vocabulary and human authority | 5.5 | Vocabulary works; "agents cannot Commit/decide" is not enforced; proposal apply is not idempotent |
| Surface parity (web, CLI, MCP) | 6.5 | 50+ MCP tools, but the Coach chat, project updates and validation have no verb; convert hard-codes zh |
| Promise honesty (README vs code) | 7.0 | PRODUCT.md and the tracker disclose most gaps; a few README lines overstate |
| **Overall** | **6.9** | Works on the documented path; several real defects, none data-destroying |

## What is good

- `server/internal/roadmaptime/roadmaptime.go:80-108`: Now admits deadlined in-progress leaves only, undeadlined ones go to `NeedsDeadline`, Next is capped at three by earliest deadline, `validated` counts in Done. A direct, pure encoding of PRODUCT.md principle 4.
- `server/internal/store/ideas.go:206-273`: `ConvertIdeaWithLocale` claims the conversion with a guarded `UPDATE`, mints PRD plus conversation in one transaction, and carries the idea body in as the first coach message. The comment explains the lock trap it avoids.
- `server/internal/api/roadmap_validation.go:24-110` with `server/internal/store/roadmap_validation.go`: observation and validate are separate routes; `validated` is reachable only through the human confirm with a non-empty reason.
- `server/internal/api/github_webhook.go:52-95` and `github_webhook_evidence.go:58-146`: HMAC verification, repo-to-project resolution through a SECURITY DEFINER helper, idempotent evidence on (node, kind, detail), and failed deployments appended rather than rewritten.
- `server/internal/agent/roadmap_planner.go:21-23,193-221` and `prompts.go:60`: the locale reaches the coach, critic and planner prompts, with the offline fallback kept.
- `docs/issues/2026-10-11-product-promise-gaps.md` plus `scripts/tests/check_product_promise_gaps.py`: every unimplemented PRODUCT.md slice has a dated record, and a checker fails when one is untracked.

## Feature status table

| Advertised item | Status | Implementing code | Gap |
|---|---|---|---|
| Idea inbox, private by default | IMPLEMENTED | `api/ideas.go:14-19`, `store/ideas.go` | none |
| Idea to PRD with body kept as coach context | IMPLEMENTED | `store/ideas.go:206` | Project is created lazily at first roadmap build (`store/roadmap_plan.go:204`), not at convert as PRODUCT.md:68-71 says |
| PRD, eight sections, revisions | IMPLEMENTED | `store/prds.go`, `api/prds.go:20-24` | no autosave (tracked) |
| PRD Coach (SSE, tools, fact ledger) | IMPLEMENTED | `agent/coach_loop.go:62`, `agent/ledger.go` | needs an LLM key |
| Coach offline mode ("template-reply coach") | PARTIAL | `agent/template.go:33` | echoes the user's last message; asks nothing, writes nothing (FEAT-06) |
| Critic with blocking/major/minor | IMPLEMENTED | `agent/critic.go` | none |
| Readiness, never a score | IMPLEMENTED | `readiness/readiness.go:44-70` | "ready" means section non-blank |
| Human Commit with Diff | IMPLEMENTED | `store/prd_commits.go:34` | "human-only" not enforced (FEAT-01) |
| Audience views and export | IMPLEMENTED | `audienceview/audienceview.go`, `api/prds.go:266` | export headings English-only; no secondary audience (FEAT-05) |
| Roadmap tree, canvas, minimap, owner, acceptance | IMPLEMENTED | `api/roadmap.go:17-27`, `client/.../roadmap_canvas.dart` | none |
| AI build/expand roadmap | IMPLEMENTED | `api/roadmap_ai.go:68-71`, `agent/roadmap_planner.go` | none |
| Time-first progress, pressure, SSE | IMPLEMENTED | `roadmaptime`, `api/roadmap.go:24` | no overdue transfer/abandon-with-reason |
| Evidence and validated state | PARTIAL | `api/roadmap_validation.go` | web only; no CLI/MCP verb (FEAT-03) |
| Proposals and decisions | PARTIAL | `api/agent_routes.go:225-270` | duplicate nodes on re-apply (FEAT-02); apply errors swallowed |
| GitHub App and webhook evidence | PARTIAL | `api/github_webhook.go`, `github/appauth.go` | commit timeline unauthenticated, so private repos and shared rate limit (FEAT-04) |
| Presence and live cursors | IMPLEMENTED | `presence`, `client/.../roadmap_page.dart:101-125` | single instance by design (documented) |
| CLI and MCP | PARTIAL | `server/cmd/insideout/cli.go`, `mcpserver/product_tools.go` | no Coach chat, project update, member, validation verbs |
| Invitation and mail funnel | IMPLEMENTED | `api/admin.go:22-27`, `api/auth_email_flows.go` | exercised live per the ledger |
| Bilingual (en, zh) | PARTIAL | `client/lib/l10n`, `agent/prompts_en.go` | exports, audience copy and CLI/MCP convert are zh or en only (FEAT-05) |
| Structured personas, Focus, lenses, dependencies, Product Branch/Merge, scoped feedback, per-Project roles | DOCS-ONLY | none | all tracked in `docs/issues/2026-10-11-*` and disclosed in PRODUCT.md |

## Findings

**FEAT-01 (High). "Humans only" Commit and decision is a tool description, not a rule.** `server/internal/mcpserver/product_tools.go:120`: `addTextTool("commit", "Human Commit: freeze the working PRD ...`. The README labels `insideout commit` `人类专属 / humans only` (README.md:172) and PRODUCT.md:342-344 says agents cannot Commit or apply strategic changes. The MCP server calls the same `/api/v1` routes with the user's own token, and no route distinguishes an agent from a person. `proposal_decide` (line 176) takes `apply`, so an agent can file a proposal, accept it and have the nodes created under the human's RLS context. Fix direction: mint an agent-scoped token (a claim checked by `requireAuth`) that is refused at `/prds/{id}/commit` and `/agent/proposals/{uid}/decision`, and have the MCP server use it. First check: call the MCP `commit` tool against a scratch server and confirm a 201.

**FEAT-02 (High). Re-accepting a proposal with `apply` duplicates its nodes; failure is swallowed.** `server/internal/api/agent_routes.go:254`: `if req.Apply && req.Decision == "accepted" {`. `DecideProposal` upserts (`store/proposal_decisions.go:67`: `ON CONFLICT (update_id) DO UPDATE`), nothing marks items applied, and `ApplyProposalItems` creates each node in its own transaction. Confirmed by running: a scratch store test that accepts and applies the same proposal twice produced 2 nodes for 1 item. The handler also logs an apply error and still returns 200 with an empty `appliedNodes`. A double-click, a retry, or a reversal then re-accept (PRODUCT.md allows reversal) doubles the roadmap. Fix: add `applied_at` on `proposal_items`, apply all items in one transaction with `WHERE applied_at IS NULL`, and return 500 on failure. diff_size: ~40 lines plus one migration.

**FEAT-03 (Medium). The new validation leg has no CLI or MCP verb.** `server/cmd/insideout/cli_roadmap.go:75`: `fs.String("status", "", "locked|pending|in_progress|done")`. `POST /roadmap/{nid}/evidence` and `/validate` exist (`api/roadmap.go:26-27`) and the web client uses them, but `grep validat` over `server/cmd`, `mcpserver` and `apiclient` finds nothing. README.md:33-37 promises the CLI has every verb and MCP the same verbs; `docs/usage/cli-mcp-and-guidance.md:83` still lists four statuses. Fix: add `roadmap observe` and `roadmap validate` plus matching MCP tools and correct the status list.

**FEAT-04 (Medium). Commit timeline sync ignores the GitHub App, so private repos fail and public ones share a 60/hour limit.** `server/internal/api/github_webhook.go:168`: `fresh, err := github.FetchCommitsSince(ctx, repoURL, lastSHA, 20, 5)`. `FetchCommitsSince` (`github/github.go:113`) takes no token, while the installation token is used only for the guide fetch (`github_webhook.go:101-107`). Every push, PR and deployment delivery triggers one to five unauthenticated calls from the server's IP. Private repos get guide evidence but a stale timeline, with only a log line. Fix: pass the installation token through `syncGithubProject` into `fetchCommitPage`.

**FEAT-05 (Medium). Bilingual is uneven outside the UI.** `server/internal/export/render.go:19`: `"background":   "Background & Problem Statement",`. The exported Markdown and print HTML (`lang="en"`) and the audience purposes and whys are English whatever the PRD language, though zh-CN is the shipped default. Separately `server/internal/apiclient/prd.go:23`: `map[string]any{}` sends no locale, so CLI and MCP `idea convert` always start a Chinese coach conversation. Fix: take a `locale` query parameter on export and a `locale` field on the convert call, with a small label table.

**FEAT-06 (Medium). The offline coach is an echo, but the docs say it "works end-to-end".** `server/internal/agent/template.go:33`: `onDelta(lastUser)`. `docs/usage/environment.md:201-203` says the deterministic template streamer makes "the PRD coach work end-to-end" and recommends it for local dev. It asks no question, records no fact, edits no section, and the critic returns no findings. Fix: reword the docs to "echo; edit sections by hand", or return a fixed first question.

**FEAT-07 (Low). A `deployment` creation event is recorded as `release` evidence.** `server/internal/api/github_webhook_evidence.go:91`: `p.kind = "release"`. GitHub fires `deployment` before any outcome, so a queued or later-failed deploy leaves a release row, with a separate "deploy failed" activity row beside it. PRODUCT.md:314-318 says a deployment "may show release". Fix: record `release` on a `deployment_status` of `success` and keep creation as `activity`.

**FEAT-08 (Low). README says GitHub evidence attaches "automatically" after binding a repo.** `README.md:13`: `绑定 GitHub 仓库后，提交、PR 和部署会作为**证据**自动挂到对应的路线图节点上`. Evidence attaches only to leaves a committed `insideout.yaml` matches, and needs the webhook configured on the repo (`docs/usage/cli-mcp-and-guidance.md`). One clause fixes it.

**FEAT-09 (Low). Decision-log comment drift.** `server/internal/store/proposal_decisions.go:13`: `(immutable; re-deciding appends, see`. The row is overwritten by upsert; only the timeline note appends, so the decision log can disagree with the history. Fix the comment or keep a decisions history table.

Wall-clock siblings of L1: I scanned server and client tests for pinned dates against `time.Now` code. The product-significant ones (`roadmaptime`, `agentcontext`) inject `now`; the `time.Now().Add(48h)` fixtures in `prd_to_progress_test.go:135` and `roadmap_acceptance_owner_test.go:41` are relative, so safe. No new sibling found.

## Lead verification

| Lead | Result | Evidence |
|---|---|---|
| L1 | Extended (no new sibling) | `projects.go:84` is the only product code path with a pinned-date test; roadmaptime and agentcontext inject the clock |
| L3 | Not re-run in full | I ran only the store package probe; no contradiction seen |
| L12 | Not checked | outside lens |
| L17 | Partly confirmed for FEAT rows | FEAT-04 (validation) is real in API and web but not CLI/MCP (FEAT-03); FEAT-01 (locale) holds for the UI, not for exports or CLI convert (FEAT-05) |
| L18 | Not checked | data-rls lens |

## Recommendations

1. Make the human boundary real: an agent-scoped token refused at Commit and proposal decision (FEAT-01).
2. Make proposal apply atomic and idempotent, and stop swallowing its error (FEAT-02).
3. Add CLI and MCP verbs for observation and validation, and update the status list in the docs (FEAT-03).
4. Thread the GitHub App installation token through the commit-sync path (FEAT-04).
5. Add a locale to export and to the CLI/MCP convert call (FEAT-05).
6. Correct the offline-coach wording, or give it a scripted first question (FEAT-06).
7. Record `release` evidence only on a successful deployment status (FEAT-07).
8. Add a promise-register row per new route, so a route without a CLI or MCP twin fails a parity test.

## Features to stop advertising until fixed

- "Humans only" Commit and "agents cannot decide": until FEAT-01 is fixed, say it is a convention for well-behaved agents.
- "The PRD coach works end-to-end" offline: it echoes.
- "Every verb in the CLI/MCP": the Coach chat, project updates and validation are web-only.
- "Evidence attaches automatically" for private repos: only the guide match works there.

## What this role would do first

Reproduce FEAT-02 and FEAT-01 in one regression test each, then fix FEAT-02, because it silently damages data a person already trusts: accept a proposal twice and the roadmap doubles. The fix is one migration and one transaction. Then take FEAT-01 as a short design note (which routes an agent-scoped token may call), since it decides whether the "agents report, humans decide" sentence on the README is a promise or a hope.
