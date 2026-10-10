# HANDOFF

Resume here. The authoritative multi-task board is
[`docs/plans/README.md`](plans/README.md). The target product experience is
[`PRODUCT.md`](../PRODUCT.md), and the running system is described by
[`docs/architecture/`](architecture/index.md). Ordinary changes are
recorded in [`docs/changelogs/`](changelogs/README.md) and open debt in
[`docs/issues/`](issues/README.md) — this file holds only the resume
path and current warnings, never history.

## What to do next

**Nothing is in flight in the repository (2026-10-10).** The
[close-the-open-gaps plan](plans/2026-10-10-close-open-gaps/2026-10-10-close-open-gaps.md)
worked through every issue record and board item, and the
[issues index](issues/README.md) lists what is left. What remains is a choice
or a secret, not unfinished code, and it is the owner's:

1. **Ship.** Nothing from 2026-10-08 on is deployed: the hosted web bundle and
   server predate the block-craft slices and this wave. `railway up --service
   server` applies two migrations on start (`20261010100000_ai_runs_reaper_index`
   and `20261010110000_rotate_short_invite_codes`; the second replaces
   workspace invite codes shorter than 32 characters, so an outstanding short
   code stops working and admins read the new one in the workspace). Then
   `railway up --service app`, the two requests in
   [BUG-027](issues/2026-10-10-bug-027-nginx-resolved-the-api-upstream-once.md#owner-check-after-the-deploy),
   and a signed-in page load.
2. **Destructive or secret steps:** purging the 582 legacy users;
   `INSIDEOUT_MAIL_API_KEY` and `INSIDEOUT_MAIL_FROM` in Railway plus a
   redeploy; a store-signed Android bundle (needs the keystore); the Supabase
   automated-backup toggle; the GitHub App install; a live critic exercise
   (needs an LLM key). Each is on its [board](plans/README.md) row.
3. **MBP:** install Flutter 3.44.x
   ([issue](issues/2026-10-09-host-setup-drift-between-macs.md)) and check that
   the leftover `app/.env` from Wave 0.12 is gone (absent on MBP2, not
   verified on MBP).

Every other thread (frontend block craft, setup TUI and release pipeline,
install skill, app icon, Cobalt-Neo token pass, simulation fixes) is closed
with its exact status and links on its [board](plans/README.md) row. Local run
commands: [`docs/usage/local-development.md`](usage/local-development.md).

## Warnings

- **`main` is meant to be one commit on GitHub, MBP, and MBP2**; confirm with
  the three-way check in the
  [sync practice](practices/2026-10-09-sync-two-macs-and-github.md). Untracked
  `.grok/` and `.qoder/` are tool scratch and raw agent journals, held out by
  rule and listed in the local manifest `.git/info/untracked-manifest.tsv`.
  **Do not `git add .`** — stage and checkpoint one task at a time, and do not
  revert unrelated edits.
- **Flutter SDK drift:** MBP runs 3.47.0 and the repo pins 3.44.x, so
  `flutter pub get` there rewrites `client/pubspec.lock`. CI and the Dockerfile
  use `pub get --enforce-lockfile`, which fails on a drifted lockfile; on MBP
  do not commit a rewritten `client/pubspec.lock`.
- **Shipping:** Railway autodeploy is off for both services, and
  `railway redeploy` re-runs the last image instead of building. Ship landed
  changes explicitly: any `server/` change with `railway up --service server`,
  any `client/` change with `railway up --service app`. **Until the BUG-027
  nginx fix is deployed and checked, restart `app` after a `server` deploy**
  (`railway redeploy --service app`): the image running now resolves the
  server's internal address only at startup, so `/api` via the domain 504s
  (verify with `/api/v1/me` → 401, not `/healthz`, which the SPA answers).
  Once the second request in the BUG-027 record passes without a restart, drop
  this ritual.

## Non-negotiables

- Never read or expose `.env` or `.env.local`.
- DB-dependent behavior requires a real PostgreSQL check and must stay
  inside the `insideout` schema.
- Commands and environment-safe launch instructions live in
  [`docs/usage/local-development.md`](usage/local-development.md).
