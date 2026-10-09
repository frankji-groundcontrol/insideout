# HANDOFF

Resume here. The authoritative multi-task board is
[`docs/plans/README.md`](plans/README.md). The target product experience is
[`PRODUCT.md`](../PRODUCT.md), and the running system is described by
[`docs/architecture/`](architecture/index.md). Ordinary changes are
recorded in [`docs/changelogs/`](changelogs/README.md) and open debt in
[`docs/issues/`](issues/README.md) — this file holds only the resume
path and current warnings, never history.

## What to do next

**Frontend block craft: code complete and committed on `main`, one
verification item open (2026-10-09).** The component layer now sits on
the frozen cobalt tokens — block primitives in
`client/lib/theme/block_*.dart`, every surface swept onto them, gates
green (analyze 0 errors / 0 warnings, **167/167** tests, format clean)
([plan](plans/2026-10-07-frontend-block-craft/2026-10-07-frontend-block-craft.md),
[changelog](changelogs/2026-10-07-frontend-block-craft.md)).
The fidelity review scored 7.5/10 and its one high finding — the custom
blocks had no focus ring or keyboard activation, an app-wide Tab
regression — is **fixed** by `theme/block_focus.dart`
([issue resolved](issues/2026-10-08-block-focus-ring-keyboard.md),
[changelog](changelogs/2026-10-08-block-focus-ring.md)). Medium/low
residue stays filed
([follow-ups](issues/2026-10-08-block-craft-fidelity-followups.md)).
**Resume path:** run the outstanding web walk at 1440×900 and 390×844 in
both themes — real Tab order across a signed-in page, the cobalt ring on
a live canvas, dock clearance, nothing hidden behind the dock — then flip
the plan and its board row to Finished.
`client/test/cobalt_test.dart` must stay unmodified; if a frozen token
assertion fails, the sweep was wrong.

Every other thread (setup TUI + release pipeline, install skill, app
icon, Cobalt-Neo token pass, simulation fixes, remediation operator
remainders) is closed or parked with its exact status, next action, and
links on its [board](plans/README.md) row. Local run commands:
[`docs/usage/local-development.md`](usage/local-development.md).

## Warnings

- **`main` is one commit on GitHub, MBP, and MBP2** (synced 2026-10-09,
  [plan](plans/2026-10-09-host-sync/2026-10-09-host-sync.md)); the
  block-craft and app-icon slices are committed. Untracked `.grok/` and
  `.qoder/` are tool scratch and raw agent journals, held out by rule and
  listed in the local manifest `.git/info/untracked-manifest.tsv`.
  **Do not `git add .`** — stage and checkpoint one task at a time, and
  do not revert unrelated edits.
- **Host drift:** MBP runs Flutter 3.47.0 but the repo pins 3.44.x, so
  `flutter pub get` there rewrites `client/pubspec.lock`; git hooks are
  not enabled on MBP2
  ([issue](issues/2026-10-09-host-setup-drift-between-macs.md)).
- **CI on `main` is red** in two jobs, neither from the newest slices: the
  python key-store job was red before the sync, and the Go job fails two
  config-isolation tests on Linux
  ([BUG-021](issues/2026-10-09-bug-021-go-tests-share-one-config-file-on-linux.md)).
- **Shipping:** Railway autodeploy is off for both services, and
  `railway redeploy` re-runs the last image instead of building. Ship
  landed changes explicitly: any `server/` change with
  `railway up --service server`, any `client/` change with
  `railway up --service app`. **After a `server` deploy, restart `app`**
  (`railway redeploy --service app`): its nginx resolves the server's
  internal address only at startup, so `/api` via the domain 504s until
  then (verify with `/api/v1/me` → 401, not `/healthz`, which the SPA
  answers). Nothing from either slice above is deployed; the hosted web
  bundle still predates both.

## Non-negotiables

- Never read or expose `.env` or `.env.local`.
- DB-dependent behavior requires a real PostgreSQL check and must stay
  inside the `insideout` schema.
- Commands and environment-safe launch instructions live in
  [`docs/usage/local-development.md`](usage/local-development.md).
