# 40 · Remediation plan — workflow-executable

> **Executed 2026-10-11** — all four waves delivered (`5c8f54f..4464330`),
> every cluster verifier-accepted round 1; outcomes per row live in the
> [disposition ledger](../2026-10-10-disposition-ledger/04-multirole-3.md),
> changes in the [changelog](../../changelogs/2026-10-11-rv3-remediation/index.md).
> The text below is the plan as executed, unchanged.

This plan turns the review-3 register into four waves of file-disjoint
clusters, each executable by the worker → adversarial-verifier → repair loop
that carried the close-open-gaps sweep (24 batches, 86 commits, zero
post-landing defects). Nothing here requires the owner; the owner-blocked
rows are listed separately at the end.

## How to run it

One dynamic workflow per wave (or one workflow, four sequential phase
groups). Clone the proven script
`.zcode/workflow-drafts/close-review-gaps-continue.ts` and swap its
`ClusterSpec` array for the wave's table below; the loop
(`worker → verifier → repair ≤3 → ACCEPTED`) needs no changes. The invariants
that made the sweep work, restated:

- **Per cluster**: a fresh `git worktree` on branch `rv3/<cluster>` off the
  wave's base SHA. Workers get the sweep's worker rules (ownership, secrets,
  no-mocks real PostgreSQL, gates, red-test-first commit order, mutation
  check, ≤350-line files, explicit-path commits, no push).
- **Red first**: every code item lands its failing test in its own commit
  before the fix; docs/CI items land a shell check that fails first instead.
  A finding the worker cannot make fail is reported NOT REAL with evidence —
  never "fixed" blind.
- **Adversarial verifier**: a fresh agent at the branch head re-runs the named
  red test at the red commit (must fail), at HEAD (must pass), re-runs the
  cluster's gates, mutates the fix's key line (must go red), and writes
  `verdict-<cluster>-r<N>.json`: `ACCEPTED | REPAIR (issue list) | REJECTED`.
  Per the critic (COMP-05), IMPROVE-bucket rows get the mutation check even
  though no skeptic verdict covers them.
- **Repair loop** ≤3 rounds; verifier names stay unique per round.
- **Reports and verdicts are JSON files on disk** (resumable orchestration);
  the workflow's board artifact tracks cluster state.
- **Integrator** (coordinator, not workers): cherry-pick `-x` in listed
  cluster order onto `main`, resolve add/add conflicts by content, full gates
  (`go test ./... -race`-equivalent locally, client analyze/format/test),
  one whole-change adversarial review, then the records pass — changelog
  folder, register dispositions, board rows, HANDOFF.

Cluster tables below are the `ClusterSpec` source of truth. "Files" are the
ownership boundary; a worker needing anything else stops and reports.

## Wave 1 — FIX the confirmed Highs (code + toolchain)

All clusters are red-test-first unless marked *(shell)*.

| Cluster | Findings | Files owned | Brief (what the worker proves red, then fixes) |
|---|---|---|---|
| `w1-gate-integrity` | AIDEV-01, AIDEV-07 | `server/internal/agent/critic.go`, `tools.go`, adjacent `_test.go` | Test: a second ranked reply cannot drop an open blocking finding or a recorded resolved/overridden status — make `captureRankedCritique` merge by (section, severity, issue) preserving ID+status; full replace only on the stage's first ranked reply. Then: `resolve_finding` parks `(id,status)` as pending and applies on the same human confirmation `advance_stage` uses (the pendingStage pattern) |
| `w1-planner-fence` | AIDEV-02 | `server/internal/agent/roadmap_planner.go`, `prompt_fence_surfaces_test.go` | Test: `TestExpandNode_FencesUntrustedNodeText` fails because ExpandNode interpolates title/node/desc raw; fix mirrors PlanMVP — append `dataFenceClause`, wrap all three fields in `fenceData` |
| `w1-email-copy` | FULL-01, FULL-06, FULL-08, TEST-02 | `server/internal/api/admin.go`, `auth_email_flows.go`, `auth_email_flows_test.go` | Test: HTML and text expiry copy agree for a >24h invite TTL (currently "168 minutes"); make `simpleEmailHTML` unit-aware. Rebrand the shared button to cobalt `#3B5BFF`, squared (FULL-06). Then the TEST-02 round trips: request→parse token from dev-mode body log→consume→session cookie, for magic-link, reset, and admin invite |
| `w1-toolchain-floor` | ARCH-01, ARCH-02 *(shell)* | `server/Dockerfile`, `.github/workflows/ci.yml`, `README.md:58`, `docs/usage/installation.md:33`, `docs/architecture/backend.md:3`, `scripts/install.sh:22`, `skills/insideout-setup/references/modules.md` | **Decision is settled by the coordinator's reproduction**: `x/sys` and `x/term` require Go 1.26.0, so bump `server/Dockerfile` to `golang:1.26-alpine` — do NOT lower go.mod. Add a CI lane that builds `server/Dockerfile` (mirror the client-image principle at ci.yml:141-143). Sync the five "Go 1.25+" texts to 1.26; make `modules.md` instruct reading `server/go.mod` instead of quoting it. Shell checks: yml parse + the five greps |
| `w1-sse-timeout` | REL-01 | `client/lib/api/client_core.dart`, `client_stream.dart`, `client_stream_test.dart` | Test: the coach POST carries a per-request `receiveTimeout: Duration.zero` override (dio treats ≤0 as off) so the native stream is governed by the client's own 75s idle timer; keep all other requests at 30s |
| `w1-idea-body` | FEAT-02 | `server/internal/store/ideas.go`, `agent_conversations.go`, store tests (DB-gated) | Red test (integration): after ConvertIdea, the conversation's first user message contains the idea body verbatim; fix inserts it inside the existing transaction (seed the fact ledger from it only if mechanical — else report back) |
| `w1-stage-confirm` | USER-01 | `client/lib/features/prd/coach_panel.dart`, `prd_page.dart`, the conversation API client file, widget tests | Red widget test: when `pendingStage` exists the panel shows confirm/decline; confirm calls `POST /conversations/{id}/stage`, decline clears it; the header renders the proposed stage as "proposed (unconfirmed)" until confirmed, so refresh no longer reverts silently |

## Wave 2 — HARDEN (providers, bounds, degradation honesty)

| Cluster | Findings | Files owned | Brief |
|---|---|---|---|
| `w2-provider-hardening` | AIDEV-03, AIDEV-06 | `server/internal/agent/taxonomy.go`, `responses.go`, `coach_fail.go`, tests | Tests: a 402 response classifies as `ErrProviderConfig` and does NOT count against the breaker (currently unclassified counts, 5 opens it service-wide); Responses request carries `max_output_tokens: 4096`; unknown in-stream Responses codes map to config-classified, not breaker-feeding. Stretch (CQ-04): de-triplicate the streamed-delta counter |
| `w2-turn-bounds` | AIDEV-05, AIDEV-11, CQ-07, USER-08 | `server/internal/agent/coach.go`, `coach_loop.go`, tests | Tests: completion requires the LAST non-empty line to equal `NO OPEN QUESTIONS` exactly (substring anywhere no longer completes); a total-turn deadline (10–15 min, classified non-circuit) ends trickling turns; fix the two pre-wave comments (CQ-07); the interrupted marker is localized/bilingual-neutral (USER-08) |
| `w2-github-hardening` | SEC-02, SEC-06, FEAT-07 | `server/internal/github/appauth.go`, `github.go`, `server/internal/api/github_webhook.go`, tests | Tests: FetchGuideFile enforces the 1 MiB cap while streaming (`io.LimitReader`), not after buffering; owner/repo/ref segments are URL-escaped before interpolation; a `deployment_status` event after a recorded release updates that evidence instead of being dropped (FEAT-07) |
| `w2-store-bounds` | REL-02, REL-03, SEC-08 | `server/internal/store/pool.go`, `agent_messages.go`, `users.go`, tests | Set a `statement_timeout` on pool config and in migrations (bounded, e.g. 30s; DB-gated test that a `pg_sleep` query is cut off); extend the reaper to purge old agent message rows / telemetry / sessions on the same schedule (REL-03 — pick retention from docs, state assumption); delete dead `SharesWorkspaceWith` (SEC-08) |
| `w2-client-guards` | FULL-02, CQ-03 | `client/lib/features/prd/prd_page.dart`, `revisions_page.dart`, `features/workspace/settings_page.dart`, tests | Widget tests: controller writes after awaited loads are mounted-guarded on both pages; settings reload preserves in-progress edits. Extract the triplicated snapshot-dialog flow into one shared widget (CQ-03) as a pure-move commit first |
| `w2-honest-degradation` | USER-03, PM-04 | `server/internal/agent/template.go`, `sse.go`, `client/lib/features/prd/coach_panel.dart`, `client/lib/api/client_auth.dart`, tests | Server: `message_start` carries the model identity (or `offline: true`); the offline template reply stops embedding env-var instructions and is locale-neutral. Client: coach panel badges offline mode; absent `mailEnabled` reads as unknown (no affirmative mail claim) rather than `?? true` |
| `w2-url-schemes` | SEC-07 | `server/internal/api/me.go`, workspaces/projects handlers, tests | Red test: `avatarUrl`/`coverUrl` accepting `javascript:` or non-http(s) schemes is rejected 400; allow only `https:` (and `http:` in dev) at the validation seam |
| `w2-mail-signal` | ARCH-03 | `server/internal/api/admin.go`, admin tests | Admin test-send probe: POST a typed recipient → Send → surface the provider status/error; record the last Send error where an operator sees it (`insideout setup --check` reads it). Builds on `w1-email-copy`'s landing |
| `w2-throttle-caveat` | SEC-03, REL-05 | `server/internal/api/auth_throttle.go`, deployment docs | Bounded: a startup log + docs paragraph stating the single-replica contract (throttle/cooldown/SSE registries are per-process), and a runtime guard that warns loudly if a second process is detected (advisory heartbeat row). Redis-backed registry is explicitly out of scope — issue if ever multi-replica |
| `w2-mcp-parity` | AIDEV-04 | `server/internal/mcpserver/product_tools.go`, tests | Declare the `unresolved` argument the commit handler already parses (sections_json-style string option if the lib lacks arrays) — red test: the advertised schema names it and a schema-abiding client can pass it |
| `w2-ledger-caps` | AIDEV-10 | `server/internal/agent/ledger.go`, tests | Rune-cap fact/assumption text and section markdown (maxQuoteLen pattern) and cap assumptions per conversation — red test: oversized model-authored strings are truncated and the count cap rejects beyond it |

## Wave 3 — IMPROVE (docs truth + mechanical cleanups)

| Cluster | Findings | Files owned | Brief |
|---|---|---|---|
| `w3-docs-truth` | PM-02, PM-03, ARCH-04, ARCH-05, FEAT-08, USER-05, COMP-02, COMP-04 *(shell/docs)* | `README.md`, `client/README.md`, `docs/plans/README.md`, `docs/architecture/backend.md`, `docs/usage/cli-mcp-and-guidance.md`, `docs/HANDOFF.md`, `PRODUCT.md` (one sentence), `server/cmd/insideout/main.go` (doc comment only) | One docs pass with shell checks: client README says "serves a release, currently behind main"; close the stale Android In-flight board row; refresh backend.md's layout tree (both binaries, all 20 internal packages) and main.go's CLI pointer; tool count 50→52 or de-numberize pointing at `tools/list`; README install one-liner demoted under the release conditional; PRODUCT.md:420 re-scoped to "eight keys remain the storage scaffold"; HANDOFF records lead-15's verified state (COMP-04); backup-restore.md gains the unscheduled-manual-script honesty paragraph (REL-04) |
| `w3-env-catalog` | FULL-07 | `.env.example`, env-catalog test | Red test: every compose-interpolated `INSIDEOUT_*` var appears in the skeleton (currently fails on `INSIDEOUT_GH_PRIVATE_KEY_FILE`); add the commented skeleton line |
| `w3-client-polish-a` | FULL-04, CQ-01, CQ-06, USER-09 | `client/lib/chrome/product_dock.dart`, `loop_focus.dart`, `features/auth/auth_door.dart`, `features/workspace/workspace_page.dart`, `features/dashboard/dashboard_page.dart`, tests | Ideas pulse for Capture on the ideas wall; Version/Commit dead-tap resolved (pulse or disabled on `/prd/:id/versions`); the 31 raw numeric SizedBox gaps in auth_door become tokens; the duplicated chevron tile collapses to one widget; the zero-workspace Capture button routes to idea capture with workspace creation implied, not a form wall |
| `w3-client-polish-b` | FULL-05, USER-07, USER-06 | `client/lib/features/prd/export_page.dart`, `coach_column.dart`, `lib/l10n/l10n.dart`, `lib/l10n/en_prd.dart`, `zh_prd.dart` (this cluster owns the prd l10n tables in this wave) | Neutral "Export" page title + desk-menu string in both locales; the coach masthead strapline stops leaking design-spec language ("Large pull-quotes") for user-facing copy; first-visit locale honors the browser language before the stored choice (USER-06) |
| `w3-prompts-contract` | AIDEV-08, AIDEV-12 | `server/internal/agent/prompts.go`, `critic.go` (parser sites), `prompts_test.go` | Rewrite clarify discipline to PRODUCT.md's adaptive contract (grouping allowed, one-at-a-time only when branching, no fixed count/rotation, four-way continue/validate/pause/abandon recommendation clause); both findings parsers validate section keys against `store.PrdSectionKeys` and drop mismatches |
| `w3-test-hygiene` | TEST-04, TEST-06, TEST-07, CQ-05 | `client/test/feature_wiring_test.dart`, `server/internal/agent/breaker_pins_test.go`, new server line-budget test, `server/internal/setup/envwrite/envwrite.go` | Pin the client's called API paths against the server's registered routes (the drift contract TEST-04 wants, bounded to path sets); make breaker pins stop leaning on another package's test internals (own fixtures); add the server-side ~350-line budget test mirroring `line_budget_test.dart` (max file today: 344 — pin under 400 to start); delete dead exported `ClearVar` with its parity test adjusted |

## Wave 4 — ADD (bounded additions)

| Cluster | Findings | Files owned | Brief |
|---|---|---|---|
| `w4-promise-register` | COMP-01, TEST-08 *(docs/grep)* | new `docs/review/2026-10-10-multirole-3/50-promise-register.md` + README link | Walk every PRODUCT.md promise (sections 64–390) and stamp IMPLEMENTED / PARTIAL / DEAD / DOCS-ONLY with the implementing file — the classification the feature-implementation dimension never delivered. Seeds `w4-gap-issues` |
| `w4-locale-threading` | FEAT-01 | migration (conversation locale column), `server/internal/agent/prompts.go` (+ en variants), conversation-create API path, client locale send, tests | Red test: a conversation started with locale `en` carries an English system prompt end to end. Migration follows repo conventions; en prompt variants mirror the zh ones (coach stages, critic, planner). Largest bounded slice in this plan — its own verifier round |
| `w4-upgrade-lane` | TEST-01 *(shell)* | `.github/workflows/ci.yml`, checked-in seed SQL | CI job: migrate a scratch DB only up to the `20261008*` set, load the seed (users, workspaces, ideas, PRDs, updates, a legacy short invite code), migrate to HEAD, run smoke against it |
| `w4-hosted-smoke` | TEST-03, TEST-05 *(shell)* | `.github/workflows/smoke.yml`, `server/scripts/smoke.sh` | A `workflow_dispatch` hosted mode pointing `SMOKE_BASE` at the public URL (smoke.sh:20 already supports it); provider legs behind explicit env gates so the LLM/mail paths run when keys exist and say so when they don't |
| `w4-attestation-honesty` | AIDEV-09 | `server/internal/agent/ledger.go`, gate copy, client label, tests | Bounded slice (decision recorded): keep the stored enum, make every user-facing surface say "model-recorded" instead of "attested", and gate copy stops implying human vouching. The full confirm/correct/remove facts API is a parked slice — `w4-gap-issues` files it |
| `w4-validation-slice` | FEAT-04, REL-07 | new evidence-kind migration + API + client leaf form | `validation` evidence kind (human-entered observation: what, source, date) + `validated` node state requiring human confirm with reason. Largest item in the plan; if it overruns its budget it splits into its own dated plan |
| `w4-contribution-surface` | PM-05 *(docs)* | new `CONTRIBUTING.md`, `SECURITY.md`, README table row | Thin: point at AGENTS.md, the changelog convention, CI gates; private disclosure path for the hosted instance |
| `w4-gap-issues` | CQ-02, FULL-03, SEC-04, USER-04, FEAT-05, FEAT-06 *(docs)* | `docs/issues/` + issues README + PRODUCT.md status link | One dated issue consolidating the parked product slices (Product Branch/Merge + Decision Log, scoped feedback invitations, tokenized audience-share link, persona grounding labels, roadmap lenses, dependency edges, Focus surface) with per-slice fix prompts and the recommended order (Branch first — Merge/Decision Log/personas hang off it); linked from PRODUCT.md's status section |

## Owner-blocked (not workflow work)

Deploy, keys, and first release — already in [HANDOFF](../../HANDOFF.md), now
with register rows: **SEC-01** + **TEST-01**'s production half (apply the four
migrations), **USER-02**/**FEAT-03**/**PM-01**/**ARCH-03**'s live half (set +
verify mail key, one live funnel exercise), **PM-02**/**PM-04**'s resolution
(the deploy itself), **USER-05**/**REL-06**/**SEC-05** (first `v*` tag, first
real LLM exercise). The workflow waves above make every one of these
one-command verifiable once the owner acts.

## Not in scope yet

Redis-backed throttle registries (only if multi-replica ever happens), the
full facts confirm/correct/remove trust layer (parked slice, `w4-gap-issues`),
roadmap dependency rebaselining beyond an issue row, and any PRODUCT.md slice
beyond the validation evidence wedge — the convergent recommendation of three
roles is to ship and deploy what exists before building the next layer.

## Coverage note

All 83 register rows are dispositioned by this plan: 77 executed across the waves above
(including AIDEV-04, AIDEV-10, REL-04, USER-06, folded in after a coordinator coverage
audit on execution day), 6 owner-blocked (below), and 3 satisfied by the report itself
(COMP-03 score recalibration, COMP-05 bucket statement + this plan's verify contract,
COMP-06 the 379/379 correction — all in the review README).

## Sequencing note

Waves are sequential (wave N+1 rebases on wave N's integrated main); clusters
inside a wave are parallel and file-disjoint by construction — the only
shared-file pairs across the whole plan (`admin.go`, `critic.go`, `ci.yml`,
`prd_page.dart`, l10n tables) never co-occur inside one wave. Expected
totals: wave 1 ≈ 7 clusters / ~14 commits, wave 2 ≈ 11 / ~20, wave 3 ≈ 6 /
~12, wave 4 ≈ 8 / ~16 — comfortably inside one workflow run per wave at the
sweep's observed pace.
