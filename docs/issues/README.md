# Issues

Concrete, dated problem records: bugs with root causes and fixes, structural
debt, deferred splits, and open questions that need a decision. Compact
issues are single dated files (`YYYY-MM-DD-title.md`); issues that accumulate
evidence get a dated folder. English only.

Bug records keep their historical `BUG-NNN` identity in the filename and
prose (they are cross-referenced by number throughout `docs/`); new bug
records continue the numbering (next: BUG-021).

## Bug records

- [BUG-001 — happy-dom localStorage](2026-07-20-bug-001-happydom-localstorage.md)
  — test-infra localStorage capability check broke under happy-dom.
- [BUG-002 — BaseInput SSR hydration](2026-07-20-bug-002-baseinput-ssr-hydration.md)
  — `Math.random()` default id mismatched between server and client render.
- [BUG-003 — Tailwind dynamic class interpolation](2026-07-20-bug-003-tailwind-dynamic-class-interpolation.md)
  — template-literal class names are invisible to Tailwind's scanner.
- [BUG-004 — Compose nested interpolation](2026-07-20-bug-004-compose-nested-interpolation.md)
  — docker-compose can't nest `${...}` inside `:-` defaults; `DATABASE_URL`
  became explicitly required.
- [BUG-005 — go.mod toolchain mismatch](2026-07-20-bug-005-gomod-toolchain-mismatch.md)
  — `go mod init` inherited the dev machine's toolchain version, breaking
  Docker builds.
- [BUG-006 — pnpm ignored build scripts](2026-07-20-bug-006-pnpm-ignored-build-scripts.md)
  — `pnpm-workspace.yaml` (holding the build allow-list) was missing from
  the Dockerfile's manifest COPY.
- [BUG-007 — RLS against real Postgres](2026-07-20-bug-007-rls-against-real-postgres.md)
  — five distinct bugs (schema bootstrap privileges, policy self-recursion,
  SECURITY DEFINER under FORCE RLS, row-locking vs cross-table policies,
  simple-protocol jsonb encoding) found only by running against a live
  database.
- [BUG-008 — shared-instance DB provisioning](2026-07-20-bug-008-shared-instance-db-provisioning.md)
  — dedicated-instance assumptions (role ownership, connection paths,
  pooler modes) broke against the real shared-instance target.
- [BUG-009 — langchaingo removed](2026-07-21-bug-009-langchaingo-removed.md)
  — the unmaintained library hid real answers behind extended-thinking
  content blocks; replaced with a direct Anthropic client.
- [BUG-010 — SSE Flusher swallowed by middleware](2026-07-21-bug-010-sse-flusher-swallowed-by-logging-middleware.md)
  — interface-embedding in the logging middleware dropped `http.Flusher`,
  silently breaking every streaming response.
- [BUG-011 — locale SSR hydration mismatch](2026-07-23-bug-011-locale-ssr-hydration-mismatch.md)
  — SSR always renders `zh-CN` (no `localStorage` on the server) while the
  client hydrates to the saved `en-US`, so every localized page logs a
  hydration text mismatch. Fix: read the locale from a cookie during SSR.
- [BUG-012 — project list NULL latest-update scan](2026-07-23-bug-012-project-list-null-latest-update-scan.md)
  — `ListProjectsForWorkspace`'s `LEFT JOIN LATERAL` yields `NULL` `lu.*`
  columns for projects with no updates; scanning them into value
  `string`/`time.Time` 500'd the workspace board. Fix: pointer fields.
- [BUG-013 — GitHub-sync repo link is a relative URL](2026-07-25-bug-013-github-sync-relative-repo-link.md)
  — the sync card binds the stored `owner/repo` straight into the anchor
  `href`, so an `owner/repo` link (the placeholder invites it) resolves
  relative to the app origin and 404s. Fix: normalize to
  `https://github.com/…` for display only.
- [BUG-014 — dio's web adapter never streams SSE](2026-08-23-bug-014-dio-web-sse-never-streams.md)
  — XHR + `arraybuffer` delivers a "stream" body only at connection
  close, so web SSE over dio is silently dead; presence fixed 2026-08-23
  with fetch + ReadableStream; Coach web POST joined that transport
  2026-09-18.
- [BUG-015 — SetSignupMode sent POST to a PUT-only route](2026-09-03-bug-015-signup-mode-wrong-method.md)
- [BUG-016 — Web boot gate hangs when Google Fonts is unreachable](2026-10-06-bug-016-web-boot-gate-hangs-on-font-load.md)
  — `document.fonts.load()` never settles on a dead font host, so the
  boot gate's deadline (checked only between awaits) never fires and
  the app spins forever. Fixed with a per-await timeout.
  — CLI `signup-mode <mode>` 405'd against the real API since the
  invitation slice; httptest pinned the client's consistency, not its
  correctness. Fixed (PUT) same day; lesson: projected verbs need one
  live call before closing.
- [BUG-017 — roadmap move collides with a sibling position](2026-10-06-bug-017-roadmap-move-collides-sibling-position.md)
  — `move --position 0` leaves the previous occupant at 0, so two
  siblings share a position. Fixed 2026-10-06: the move shifts siblings.
- [BUG-018 — roadmap pressure renders as a raw localization key](2026-10-06-bug-018-roadmap-pressure-label-untranslated.md)
  — the web progress view prints `roadmap.pressure.normal` beside the
  deadline. Fixed 2026-10-06: the key is translated. The release bundle
  on :5174 predates it.
- [BUG-019 — accepting a proposal applies its items out of order](2026-10-06-bug-019-applied-proposal-reorders-items.md)
  — Intake, Verification, Handoff, Archive landed as Intake, Handoff,
  Verification, Archive. Fixed 2026-10-06: items carry an explicit
  position.
- [BUG-020 — a workspace member can write the plan](2026-10-06-bug-020-workspace-member-writes-the-plan.md)
  — membership alone lets a reader add roadmap nodes and file
  proposals. Fixed 2026-10-06: only the project owner or a workspace
  admin may change the roadmap or file a proposal.

- [2026-10-06 — roadmap.go over the ~350-line budget](2026-10-06-roadmap-go-line-budget.md)
  — 403 lines after the sibling-shift fix; target split is
  `roadmap.go` + `roadmap_move.go` at the next structural touch.

## Other open issues

- [2026-10-04 — Multirole review findings register](../review/2026-10-04-multirole/01-findings-register.md)
  — 75 verified findings (12 High confirmed) with fixes and a five-wave
  remediation plan; the review of record for prioritization. Duplicates
  already tracked here stay tracked in place (e.g. auth rate limiting,
  2026-08-24 below).
- [2026-10-04 — coach.go over the ~350-line budget](2026-10-04-coach-go-line-budget.md)
  — 467 lines mixing turn lifecycle and loop mechanics; target split into
  `coach.go` + `coach_loop.go`. Filed with the CQ-01 wave-4 pass.
- [2026-10-04 — anthropic.go over the ~350-line budget](2026-10-04-anthropic-go-line-budget.md)
  — 408 lines mixing wire types/request path and stream parsing; target
  split into `anthropic.go` + `anthropic_parse.go`. Filed with CQ-01.
- [2026-10-04 — models.dart over the ~350-line budget](2026-10-04-models-dart-line-budget.md)
  — 478 lines of every API model; target four domain files behind the
  `models.dart` barrel. Filed with CQ-01.
- [2026-09-18 — Flutter ApiClient over the ~350-line budget](2026-09-18-flutter-apiclient-line-budget.md)
  — `client/lib/api/client.dart` ~529 lines; target core / product /
  stream split. Deferred from the Cobalt live-API wiring slice.
- [2026-09-03 — apiclient.go over the ~350-line modular-file budget](2026-09-03-apiclient-line-budget.md)
  — transport + roadmap/GitHub + PRD/agent verbs mixed at ~478 lines;
  target three-way split with a bounded, test-pinned fix prompt.
- [2026-08-24 — Auth endpoints have no rate limiting or failed-attempt lockout](2026-08-24-auth-endpoints-no-rate-limiting.md)
  — **resolved 2026-10-04** in remediation wave 2: per-IP + per-account
  throttle (`auth_throttle.go`; 8 failures → 15-min lockout, uniform 429 +
  `Retry-After`) plus a 60 s per-email + per-IP resend cooldown
  (`auth_cooldown.go`). Was: unlimited password guesses and link emails.
- [2026-07-29 — env.sh over the ~350-line modular-file budget](2026-07-29-envsh-line-budget.md)
  — **resolved 2026-07-30**: split at the next structural touch, as planned,
  into `env.sh` / `env-lib.sh` / `env-write.sh` (227 / 110 / 278 lines). The
  duplicated `KNOWN_NAMES` list was deleted rather than moved — it had already
  drifted from the skeletons it copied.
- [2026-07-26 — Roadmap canvas: adversarial-verify follow-ups (Workstream D)](2026-07-26-roadmap-canvas-adversarial-followups.md)
  — from the T9 adversarial pass: the popover-trapped-under-sibling defect is
  **fixed** (one-line `.rm-card:focus-within` lift; it could silently reparent a
  node); two pre-existing low items remain open — no keyboard directional pan and
  two pre-existing motions not reduced-motion-gated — each with a bounded fix prompt.
- [2026-07-26 — Canvas fit-to-view zooms wide roadmaps out too small](2026-07-26-canvas-fit-too-small-on-wide-trees.md)
  — `fitTo` clamps a wide tidy-tree to ~0.35–0.5, so the first impression of a
  large roadmap is tiny. Enhancement, not a defect: add a readable-min fit clamp
  (~0.6) with root-anchoring, keeping the user zoom floor untouched.
- [2026-07-25 — Build-from-PRD runs the LLM before the conflict guard](2026-07-25-build-from-prd-runs-llm-before-conflict-guard.md)
  — `handleBuildFromPrd` calls `PlanMVP` (LLM) before `ReplaceRoadmapTree`
  returns the non-empty 409, so a first build on a populated roadmap pays the
  call then discards it. Correctness is safe (in-tx advisory-locked guard is
  authoritative); deferred as a wasted-cost/latency fix.
- [2026-07-25 — Canvas failure feedback is a banner, not the specced toast](2026-07-25-canvas-failure-feedback-is-a-banner-not-the-specced-toast.md)
  — collab A3's stacked `useToast` + delete descendant-count confirm shipped
  as a simpler in-canvas error banner + static confirm; corrects the collab
  plan's T4 line. No mutation fails silently, so the intent is met.
- [2026-07-23 — Backend pre-commit review findings](2026-07-23-backend-precommit-review-findings.md)
  — deferred hardening/correctness follow-ups from the adversarial review before
  the first public backend commit (2 medium concurrency/data-integrity bugs, 6
  low items). No push-blockers.
- [2026-07-26 — Backend optimization review, deferred findings](2026-07-26-backend-optimization-deferred.md)
  — the items NOT fixed in the 2026-07-26 optimization pass. Six were resolved
  in the [2026-07-27 hardening pass](../changelogs/2026-07-27-security-hardening-pass.md)
  (invite-code keyspace, ConvertIdea race, PRD CAS race, EnsureProject race,
  GitHub sync N+1, provider error leaks); auth rate-limiting resolved
  2026-10-04 (remediation wave 2); still open: argon2 timing, `loadHistory`
  over-fetch, the `ai_runs` reaper index, the `ListWorkspacesForUser`
  correlated count, and the join-endpoint rate limit.
- [2026-07-21 — Tracked coding-tool scratch directories](2026-07-21-tracked-tool-scratch-dirs.md)
  — **resolved 2026-07-27**: `.sisyphus/`, `.trae/`, `review/` (JuanLeMe-era
  tool scratch) untracked, gitignored, and purged from history. See the
  [cleanup changelog](../changelogs/2026-07-27-repo-structure-cleanup.md).
