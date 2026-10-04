# 20 · Feature Implementation

**Lens:** advertised-feature truth (README core features vs implementing code, PRODUCT.md canonical journey vs shipped behavior); covers server routes/store, Flutter feature pages, MCP tool registry, CLI verbs, and docs/usage/cli-mcp-and-guidance.md.
**Score:** **6.5 / 10**
**Bottom line:** The core loop is real and unusually well-built: idea → four-stage coach → PRD → review → versions → roadmap → GitHub evidence all have implementing code behind them, server-enforced state machines, and green tests (57/57 Flutter, Go build/vet clean — I ran both). The gap is between the marquee README promises and the shipped UI: the shared board shows only title + status, the "flagged when quiet" behavior has literally zero implementation anywhere in the tree, the web export page renders text instead of downloading it, and the server-side canonical Progress view (deadline-gated Now) has no web consumer at all. Nothing here is unsalvageable — most fixes are tens of lines — but as documented today, the team-lead board story overpromises.

Reviewed against main@877ad03; High/Critical findings re-checked by an independent skeptic — verdicts in 01-findings-register.md.

---
## Scope

README.md core features; PRODUCT.md canonical journey and superseded-assumptions list; server implementing code: `server/internal/api/{projects,project_updates,ideas,prds,prd_commits,prd_readiness,prd_view,conversations,roadmap,roadmap_progress,roadmap_evidence,roadmap_ai,github,github_webhook,agent_routes,presence_routes,guide}.go`; `server/internal/store/{projects,ideas,prds,roadmap_evidence}.go`; `server/internal/agent/{prompts,tools,critic,coach}.go`; `server/internal/roadmaptime/`; MCP registry `server/internal/mcpserver/{server,product_tools,auth_tools,admin_tools}.go`; CLI dispatch `server/cmd/insideout/cli*.go`; Flutter pages under `client/lib/features/{workspace,project,prd}/` plus `client/lib/api/{client,models,export_format}.dart` and `client/lib/router.dart`; `docs/usage/cli-mcp-and-guidance.md`. Verification: `go build ./...` + `go vet` (pass), `flutter analyze` (0 errors/0 warnings, 20 info lints), `flutter test` (57/57 pass), grep sweeps for flag/stale/quiet/progress wiring.

## What is good

- **The coach is a real state machine, not a prompt prayer.** Stages are linearly enforced server-side (`server/internal/agent/prompts.go:14-17`), the clarify→draft gate demands attested facts for `problem/segment/alternative/whynow` (`server/internal/agent/tools.go:254`), draft→critique refuses empty sections (`tools.go:274-281`), and critique→finalize requires an independent critic pass with no open blocking findings (`critic.go`, `tools.go:290-295`). Facts carry verbatim user quotes (`ledger.go`), matching PRODUCT.md's evidence-trust intent.
- **Review flow authorization is enforced in the store, not the UI.** The transition table `draft→reviewing→approved/rejected, rejected→draft` (`server/internal/store/prds.go:184-189`) plus the author-only/admin-only split (`prds.go:215-227`) implements exactly what the README promises: author submits and resubmits, only a workspace admin approves or rejects.
- **Product version control exists as advertised.** `handleCommitPrd` (`server/internal/api/prd_commits.go:20-63`) freezes an immutable named version with primary audience, change summary, unresolved items, and decision note — PRODUCT.md's Commit shape — with a web versions page and commit dialog (`client/lib/features/prd/versions_*.dart`), plus snapshot revisions (`handleCreateRevision`).
- **GitHub evidence sync is end-to-end and honest.** The webhook verifies HMAC before anything else (`server/internal/api/github_webhook.go:41`), resolves projects by repo, re-runs the per-project sync under each owner's RLS context, matches only leaf nodes via `insideout.yaml`, and leaves unmatched activity unassociated. The docs description (`docs/usage/cli-mcp-and-guidance.md:107-115`) matches the code.
- **Surface parity is real.** I counted 50 MCP tools (39 product + 9 auth + 2 admin in `mcpserver/`), matching the doc's "50 today"; CLI dispatch (`cli.go:39-62`) covers the same verb set; reads return raw API JSON by design so surfaces cannot drift.
- **Presence/cursors work** on the roadmap canvas — SSE snapshot + cursor broadcast (`server/internal/api/presence_routes.go:70-139`) and a client loop with backoff that renders viewer chips and labeled cursors (`client/lib/features/project/roadmap_page.dart:50-71`).

## Findings

**FEAT-01 · High · board/flagging · README.md:21**
The board's headline promise — quiet projects get flagged — has zero implementing code anywhere in the server or client.
Evidence: README.md:21 — "Projects that have gone quiet too long get flagged — so you know what's happening without having to ask." A case-insensitive sweep of `server/` and `client/lib` for flag/stale/quiet finds only the Go `flag` package, the AI-run reaper, and presence TTL pruning — no staleness computation, no `flagged` field in `projectView` (`server/internal/api/projects.go:24-36`), no rendering in the board page (`client/lib/features/workspace/workspace_page.dart:119-133` renders only title and status).
Failure scenario: a project goes silent for three weeks; the workspace leader opens the board expecting the advertised flag and sees a plain card — the exact "chasing people to ask" scenario the README claims is solved.
Fix: compute staleness from `LatestUpdateAt` (already returned by `ListProjectsForWorkspace`, `server/internal/store/projects.go:99-100`) against a threshold, expose a `flagged` boolean, render a badge on the board card; or cut the sentence from the README until it ships. diff_size: ~40 lines (model field + API field + badge + test).

**FEAT-02 · High · board/live updates · client/lib/features/workspace/workspace_page.dart:128**
The "one shared board with live progress, blockers, and notes" is a title-and-status list; the API already returns the latest update but the client never renders it and nothing refreshes.
Evidence: workspace_page.dart:128-129 — `Text(p.title, style: theme.textTheme.titleLarge), Text(l10n.t('project.status.${p.status}'), style: theme.textTheme.bodySmall),`; the model parses `latestUpdateKind`/`latestUpdateContent` (`client/lib/api/models.dart:124-125`) yet no widget consumes them; the page loads once via FutureBuilder with reload only on user action (`workspace_page.dart:26-31`). Kinds progress/blocker/note are validated server-side (`server/internal/api/project_updates.go:29`) and visible only on the per-project page.
Failure scenario: a teammate posts "blocked: waiting for API keys" as a blocker update; everyone on the workspace board sees an unchanged card and only discover the blocker by opening every project one by one — "live progress, blockers" as documented does not exist on the board.
Fix: render the latest-update kind chip + one-line content + relative time on each board card (fields already in the response; add `latestUpdateAt` to the model, which currently drops it) and refresh on page focus or a short poll. diff_size: ~30 lines.

**FEAT-03 · Medium · export · client/lib/features/prd/export_page.dart:71**
Web "export to Markdown for download, or browser print to PDF" displays the text inline; no file download and no print path exist in the client.
Evidence: export_page.dart:71 — `if (body != null) Padding(padding: const EdgeInsets.only(top: 16), child: SelectableText(body!)),`; `client/lib/api/client.dart:382` — `return res.data.toString();` captures the body through Dio, discarding the server's `Content-Disposition: attachment` (`server/internal/api/prds.go:296-304`). Nothing calls `window.print()` or creates a download anchor.
Failure scenario: a user clicks Export → Markdown expecting a downloadable `.md` file and gets a wall of selectable text they must copy-paste by hand; on a large PRD this is the difference between one click and error-prone manual work.
Fix: for markdown, trigger an `<a download>` save of the returned bytes (or navigate the browser to the export URL directly so the attachment header applies); for print, open the `format=print` HTML in a print window. diff_size: ~30 lines.

**FEAT-04 · Medium · idea privacy · server/internal/store/ideas.go:65**
Idea capture is workspace-wide visible, contradicting PRODUCT.md's canonical journey ("private to its author by default") — a listed superseded assumption is still the shipped behavior.
Evidence: store/ideas.go:65 — `WHERE workspace_id = $1` (no author filter); the web inbox renders everything the endpoint returns (`client/lib/features/workspace/ideas_page.dart:31`); PRODUCT.md:68 — "Capture remains one-step and private to its author by default." PRODUCT.md:421 lists "captured Ideas are automatically visible to the Workspace" among superseded assumptions.
Failure scenario: a member jots a half-formed idea in the inbox; every workspace colleague sees it in the shared wall immediately — the product's canonical one-step private capture is not what the code does.
Fix: this is a product decision first — if PRODUCT.md still holds, filter `ListIdeas` by `author_id = actor` (plus admin visibility) and add an explicit "share" step; otherwise update PRODUCT.md to declare the shared inbox canonical. first_check: confirm with the product owner which side of the superseded list is intended for the current slice. diff_size: unknown (depends on the decision).

**FEAT-05 · Medium · roadmap progress view · client/lib/features/project/roadmap_bands.dart:27**
PRODUCT.md's deadline-gated Progress view is implemented server-side but unreachable from the web UI, which buckets every in-progress node into "Now" regardless of deadline.
Evidence: `server/internal/api/roadmap_progress.go:13-16` — "Now dominant (deadlined work only), at most three justified Next … in-progress work without a deadline surfaces as needsDeadline"; the client wrapper exists (`client/lib/api/client.dart:506`, `roadmapProgress`) but grep shows zero call sites in `client/lib/features`; instead roadmap_bands.dart:27-29 — `'in_progress' || 'done' => 'now', 'pending' => 'next', _ => 'later',` — and the Next band is uncapped. PRODUCT.md: "An item without a deadline cannot enter Now."
Failure scenario: a team marks three items in_progress with no deadlines; the web Now band shows all three as current work while the product contract — and the CLI's `roadmap progress` output — says they cannot be Now until they carry a deadline; web and CLI project different truths from the same store.
Fix: render `GET /roadmap/progress` as the default roadmap view in Flutter (the endpoint, model class `RoadmapProgress`, and pressure l10n keys all already exist), keeping Bands/Canvas as alternate lenses; otherwise align `RoadmapBands.bucket` with the deadline rule. diff_size: ~80 lines if the endpoint is surfaced; ~10 lines if bucketing is aligned.

**FEAT-06 · Low · review flow UI · client/lib/features/prd/prd_desk.dart:166**
The PRD status menu offers approve/reject to every viewer; non-admin clicks die as a raw 403.
Evidence: prd_desk.dart:166-167 — `for (final s in const ['draft', 'reviewing', 'approved', 'rejected']) PopupMenuItem(value: s, …)`; enforcement is server-side only (`server/internal/store/prds.go:220-227` returns ErrForbidden for non-admin approve/reject), surfaced in the UI as an unhandled exception string.
Failure scenario: a workspace member exploring the menu clicks "approved"; they get an opaque error instead of the menu simply not offering the action.
Fix: hide or disable approve/reject items unless the session's workspace role is admin (role is already available on the workspace payload). diff_size: ~5 lines.

**FEAT-07 · Low · product-model drift · README.md:29**
The README's Coach and Review sections advertise the exact model PRODUCT.md declares superseded; the docs never tell a reader which story is current.
Evidence: README.md:29 — "an AI coach walks you through four stages: **clarify** … → **finalize** with a completeness summary"; README.md:33 — "a workspace admin approves or rejects it"; PRODUCT.md:419-421 — "every coaching session follows four rigid stages, a Workspace admin approves all PRDs, captured Ideas are automatically visible to the Workspace" are listed under "Superseded assumptions".
Failure scenario: a new workshop participant reads README.md as the product contract, then meets the coach's hard stage gates and admin approval — and files feedback against behavior PRODUCT.md has already retired.
Fix: add a one-line superseded marker on README's Coach/Review sections pointing at PRODUCT.md's adaptive contract and human-Commit model (PRODUCT.md itself concedes this foundation "does not yet implement this whole target experience"). diff_size: ~4 lines.

## Lead verification

| Lead | Verdict | Evidence |
| --- | --- | --- |
| L3 (README advertises superseded model) | **Confirmed** | README.md:29,32-33 (four stages; admin approves) vs PRODUCT.md:416-422 "Superseded assumptions" list containing both. |
| L7 (size breaches incl. client/api files) | **Confirmed** | `wc -l`: client/lib/api/client.dart 529, models.dart 472 — over the ~350 rule, with no dated issue for either (only the two apiclient files are tracked in docs/issues/). |
| L13 (flutter analyze/test not run) | **Refuted — now run by this reviewer** | `flutter analyze --no-pub`: 0 errors, 0 warnings (20 info-level lints, all `use_build_context_synchronously`). `flutter test`: 57/57 pass. Plans-board claim of "54 tests green" is slightly stale (57 today) but directionally true. |
| L12 (Go clean) | **Confirmed** | `go build ./...` and `go vet` on agent/api/mcpserver/cmd pass (re-run in this review). |
| MCP "50 tools" claim in docs | **Confirmed** | Counted `addTextTool`/`addTool` registrations: 39 product + 9 auth + 2 admin = 50, matching docs/usage/cli-mcp-and-guidance.md:55. |

## Recommendations

1. Stop advertising "flagged when quiet" until staleness is computed and rendered — the README's flagship team-lead promise is currently empty (FEAT-01).
2. Render the latest progress/blocker/note on each board card; the API already returns it and the model already parses two of the three fields (FEAT-02).
3. Make Export actually deliver a download and a print dialog in the web client (FEAT-03).
4. Decide the idea-visibility model against PRODUCT.md and either filter `ListIdeas` by author or amend the canonical journey (FEAT-04).
5. Surface the server Progress view (`/roadmap/progress`) as the default web roadmap view, or align the band bucketing with the no-deadline-no-Now rule (FEAT-05).
6. Role-gate the PRD status menu so approve/reject appear only for admins (FEAT-06).
7. Mark README's Coach/Review sections with the superseded-model pointer to PRODUCT.md (FEAT-07).

## What this role would do first

Strip the false board promises from the README before anything else — specifically "Projects that have gone quiet too long get flagged" and "live progress, blockers" — or implement them this week, because both are small: `LatestUpdateAt` already reaches the API, the model is one field away, and a staleness flag is a date comparison. A team leader onboarding from the README currently inherits a mental model of the product that the codebase cannot fulfill, and that is the one gap that costs trust on day one. Everything else (export download, Progress view surfacing, idea privacy, role-gated menu) is real but secondary, and the underlying coach/review/version/evidence machinery is solid enough that these are UI-level completions rather than rewrites.
