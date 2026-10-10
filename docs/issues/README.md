# Issues

Concrete, dated problem records: bugs with root causes and fixes, structural
debt, deferred splits, and open questions that need a decision. Compact
issues are single dated files (`YYYY-MM-DD-title.md`); issues that accumulate
evidence get a dated folder. English only.

Bug records keep their historical `BUG-NNN` identity in the filename and
prose (they are cross-referenced by number throughout `docs/`); new bug
records continue the numbering (next: BUG-031).

**State on 2026-10-10:** every record is fixed, superseded or closed with
evidence, except the two under "Waiting on the owner" and one deferred
engineering item under Client. The
[close-open-gaps plan](../plans/2026-10-10-close-open-gaps/2026-10-10-close-open-gaps.md)
worked through all of them. Owner steps that are not issue records (production
deploys and migrations, purging legacy users, mail and signing credentials,
backups) are kept in [HANDOFF](../HANDOFF.md) and the [board](../plans/README.md).
2026-10-11: the parked PRODUCT.md slices opened as records under
"Open product slices" below — open by design, they track promises the
running system does not implement yet.

## Waiting on the owner

- [BUG-027 — nginx resolved the API upstream once](2026-10-10-bug-027-nginx-resolved-the-api-upstream-once.md)
  — fixed and checked on a real nginx; it takes effect when `app` is next
  deployed (`railway up --service app`). The record lists the two requests that
  confirm it.
- [2026-10-09 — Host setup drift between the two Macs](2026-10-09-host-setup-drift-between-macs.md)
  — hooks on MBP2, MBP's upstream and its HTTP/2 route are **fixed
  2026-10-10**; the Flutter 3.47.0-vs-3.44.x lock drift is guarded by
  `pub get --enforce-lockfile` in CI and the Dockerfile. Open only for the
  owner: installing 3.44.x on MBP.

## Open product slices (2026-10-11)

PRODUCT.md promises with no end-to-end implementation, one dated record per
slice with a bounded fix prompt; the enumeration and recommended slice order
is in [2026-10-11 — PRODUCT.md promise gaps](2026-10-11-product-promise-gaps.md).

- [Product Branch, Merge and the Decision Log](2026-10-11-product-branch-merge-decision-log.md)
  — no branch machinery anywhere; the Decision Log is a free-text Commit
  note plus proposal decisions.
- [Scoped external feedback invitations](2026-10-11-scoped-feedback-invitations.md)
  — the only promise with a privacy boundary; no feedback route exists,
  the only invitations grant workspace membership.
- [Tokenized audience-share link](2026-10-11-tokenized-audience-share-link.md)
  — audience views are auth-only app routes; the only export is a manual
  markdown/print download.
- [Persona grounding labels](2026-10-11-persona-grounding-labels.md)
  — personas are a free-text section; no core/key/later priority and no
  grounded/synthesized/assumed label anywhere.
- [Facts trust layer](2026-10-11-facts-trust-layer.md) — facts are recorded
  agent-side only (ledger.go); no route or client surface to confirm,
  correct or remove them.

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
  `https://github.com/…` for display only. **Not applicable 2026-10-10**: the Vue component
  went with the Nuxt app, and the Flutter card renders no link.
- [BUG-014 — dio's web adapter never streams SSE](2026-08-23-bug-014-dio-web-sse-never-streams.md)
  — XHR + `arraybuffer` delivers a "stream" body only at connection
  close, so web SSE over dio is silently dead; presence fixed 2026-08-23
  with fetch + ReadableStream; Coach web POST joined that transport
  2026-09-18.
- [BUG-015 — SetSignupMode sent POST to a PUT-only route](2026-09-03-bug-015-signup-mode-wrong-method.md)
  — CLI `signup-mode <mode>` 405'd against the real API since the
  invitation slice; httptest pinned the client's consistency, not its
  correctness. Fixed (PUT) same day; lesson: projected verbs need one
  live call before closing.
- [BUG-016 — Web boot gate hangs when Google Fonts is unreachable](2026-10-06-bug-016-web-boot-gate-hangs-on-font-load.md)
  — `document.fonts.load()` never settles on a dead font host, so the
  boot gate's deadline (checked only between awaits) never fires and
  the app spins forever. Fixed with a per-await timeout.
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
- [BUG-021 — Go tests pin only `HOME`, so they share one config file on Linux](2026-10-09-bug-021-go-tests-share-one-config-file-on-linux.md)
  — Linux `os.UserConfigDir()` prefers `XDG_CONFIG_HOME`, which GitHub's
  runner sets, so two tests read config that earlier tests saved and the
  CI Go job was red. **Fixed 2026-10-10** (`8ee97fc`, CI green), together
  with the older, separate python key-store failure noted in the record.
- [BUG-022 — "Build MVP" drafted a roadmap and wrote nothing](2026-10-10-bug-022-build-mvp-wrote-nothing.md)
  — the PRD page treated the draft from `POST /prds/{id}/build` as the
  result and opened an empty roadmap; the write is `build/confirm`, which
  the client never called. **Fixed 2026-10-10**, checked in a browser
  against a real server.
- [BUG-023 — bundled Postgres setup never wrote the owner DSN](2026-10-10-bug-023-bundled-setup-never-wrote-the-owner-dsn.md)
  — `setup --local --postgres bundled` wrote only the app-role
  `DATABASE_URL`, so a fresh run ended at the migrator's "use
  DATABASE_OWNER_URL" refusal. **Fixed 2026-10-10**.
- [BUG-024 — the "env catalog suites" CI step ran one battery of two](2026-10-10-bug-024-env-suites-ci-step-ran-one-battery.md)
  — `unittest` imported the first script-style module, which ended the
  process, so 37 checks never ran while the step stayed green.
  **Fixed 2026-10-10**; the step runs both and prints both totals.
- [BUG-025 — phone layouts and copy defects found by the web walk](2026-10-10-bug-025-phone-layouts-and-copy-found-by-the-web-walk.md)
  — five defects at 390px and 1440px that no widget test had looked at
  (roadmap toggles clipped off the left edge, an empty Progress view for
  undated goals, a squeezed PRD header, …). **Fixed 2026-10-10**, re-walked
  in a browser.
- [BUG-026 — a committed version's card hid what the version recorded](2026-10-10-bug-026-committed-versions-hid-what-they-recorded.md)
  — the card showed three counts but not the carried open items, the
  changed sections or the decision note, and the commit dialog never asked
  for a note. **Fixed 2026-10-10**.
- [BUG-027 — the app's nginx resolved the API upstream once, at start](2026-10-10-bug-027-nginx-resolved-the-api-upstream-once.md)
  — after a `server` redeploy `/api/` returned 504 until `app` was
  restarted, and nginx refused to start for a name that did not resolve yet.
  **Fixed in the repository 2026-10-10**; live after an `app` deploy.
- [BUG-028 — Enter did not activate a focused block in a browser](2026-10-10-bug-028-enter-did-not-activate-focused-blocks-on-the-web.md)
  — on the web Enter is `ButtonActivateIntent`, the custom blocks bound only
  `ActivateIntent`, and every test ran on the VM. **Fixed 2026-10-10**,
  re-walked in a browser.
- [BUG-029 — the Versions eyebrow counted revisions, not commits](2026-10-10-bug-029-versions-eyebrow-counted-revisions.md)
  — "Commit 3 of 2": the line used the pinned PRD revision as the position.
  **Fixed 2026-10-10**.
- [BUG-030 — the audience view rendered a blank page against the real server](2026-10-10-bug-030-audience-view-rendered-a-blank-page.md)
  — the page cast the server's ordered list of section picks to a map and threw
  while building, and every section card refetched the whole PRD; the tests'
  fixtures had been written to match the page, not the wire. **Fixed
  2026-10-10**, with a server-side pin of the shape.

## Other records

Each opens with its resolution and keeps the original text below it. All were
resolved, superseded or closed on 2026-10-10 unless a date says otherwise.

### Line-budget splits (pure moves)

- [2026-10-06 — roadmap.go](2026-10-06-roadmap-go-line-budget.md) — `store/roadmap.go` is 309 lines.
- [2026-10-04 — coach.go](2026-10-04-coach-go-line-budget.md) — 349 lines, the provider/tool loop in its own file.
- [2026-10-04 — anthropic.go](2026-10-04-anthropic-go-line-budget.md) — 235 lines plus `anthropic_parse.go` (186).
- [2026-09-03 — apiclient.go](2026-09-03-apiclient-line-budget.md) — 169 lines, the verbs in their own files.
- [2026-10-04 — models.dart](2026-10-04-models-dart-line-budget.md) — a 4-line barrel over four domain files.
- [2026-09-18 — Flutter ApiClient](2026-09-18-flutter-apiclient-line-budget.md) — `client.dart` is a 27-line library over per-resource part files.
- [2026-10-08 — l10n tables](2026-10-08-l10n-tables-line-budget.md) — `en.dart` (34) and `zh.dart` (18) spread four namespace files each.
- [2026-10-08 — project_page.dart](2026-10-08-project-page-line-budget.md) — 297 lines; the GitHub card, updates feed and modals are their own files.
- [2026-10-08 — roadmap_canvas.dart](2026-10-08-roadmap-canvas-line-budget.md) — 200 lines; the minimap is `roadmap_minimap.dart` (184).
- [2026-07-29 — env.sh](2026-07-29-envsh-line-budget.md) — resolved 2026-07-30: `env.sh` / `env-lib.sh` / `env-write.sh`; the duplicated `KNOWN_NAMES` list was deleted, not moved.

### Backend, setup and environment

- [2026-07-23 — Backend pre-commit review findings](2026-07-23-backend-precommit-review-findings.md) — every confirmed item fixed, each test-first on real PostgreSQL.
- [2026-07-26 — Backend optimization, deferred findings](2026-07-26-backend-optimization-deferred.md) — every item still open is closed; the `ListWorkspacesForUser` count was measured and deliberately not shipped.
- [2026-07-25 — Build-from-PRD runs the LLM before the conflict guard](2026-07-25-build-from-prd-runs-llm-before-conflict-guard.md) — superseded by the draft/confirm split ([BUG-022](2026-10-10-bug-022-build-mvp-wrote-nothing.md) found the client still on the old contract).
- [2026-10-07 — Own-URL fresh-database bootstrap](2026-10-07-own-url-fresh-db-bootstrap.md) — `setup --local --postgres own` probes the two roles, offers to create them, or stops and prints the SQL.
- [2026-10-07 — Env-catalog truth duplicated (Go + Python)](2026-10-07-env-catalog-go-python-duplication.md) — both readers kept; a parity test fails CI on drift.
- [2026-10-06 — Install skill's Ubuntu path unverified](2026-10-06-install-skill-ubuntu-unverified.md) — `ubuntu-install.yml` runs the skill's own commands on `ubuntu-24.04`.
- [2026-08-24 — Auth endpoints have no rate limiting](2026-08-24-auth-endpoints-no-rate-limiting.md) — resolved 2026-10-04: per-IP and per-account throttle with lockout, plus a resend cooldown.
- [2026-07-21 — Tracked coding-tool scratch directories](2026-07-21-tracked-tool-scratch-dirs.md) — resolved 2026-07-27: untracked, gitignored and purged from history ([cleanup changelog](../changelogs/2026-07-27-repo-structure-cleanup.md)).

### Client

- [2026-10-10 — Throttled roadmap copy deferred](2026-10-10-throttled-roadmap-copy-deferred.md) — open, deferred by choice: a 429/503 from roadmap build/expand still shows the generic error snackbar; the fix prompt is in the record.
- [2026-10-08 — Block-craft fidelity follow-ups](2026-10-08-block-craft-fidelity-followups.md) — every item done or reviewed and kept; the 390px risk was real and is [BUG-025](2026-10-10-bug-025-phone-layouts-and-copy-found-by-the-web-walk.md).
- [2026-10-08 — Block primitives have no focus ring or keyboard activation](2026-10-08-block-focus-ring-keyboard.md) — resolved 2026-10-08 ([changelog](../changelogs/2026-10-08-block-focus-ring.md)).
- [2026-10-06 — W3-N2 minified console errors](2026-10-06-wave3-minified-console.md) — closed, not reproducible on a source-mapped release build at 390px.
- [2026-10-06 — App icon candidates are not installed](2026-10-06-app-icon-not-installed.md) — resolved 2026-10-06: `breakout` is the web, iOS and Android icon.
- [2026-07-25 — Canvas failure feedback is a banner, not the specced toast](2026-07-25-canvas-failure-feedback-is-a-banner-not-the-specced-toast.md) — superseded: the Flutter client shows localized errors in a SnackBar and states a delete's blast radius.
- [2026-07-25 — RoadmapCanvas.vue modularity](2026-07-25-roadmap-canvas-modularity.md) — superseded: the Vue file is gone; the Flutter canvas split is above.
- [2026-07-26 — Canvas fit-to-view zooms wide roadmaps out too small](2026-07-26-canvas-fit-too-small-on-wide-trees.md) — not applicable: the Flutter canvas has no fit-to-view zoom.
- [2026-07-26 — Roadmap canvas adversarial follow-ups](2026-07-26-roadmap-canvas-adversarial-followups.md) — superseded: keyboard pan and reduced motion are done in Flutter and pinned by tests.

### Reviews

- [2026-10-04 — Multirole review findings register](../review/2026-10-04-multirole/01-findings-register.md) — 75 verified findings (12 High confirmed) with a five-wave remediation plan; the review of record for prioritization. A second pass is in [multirole-2](../review/2026-10-04-multirole-2/01-findings-register.md).
