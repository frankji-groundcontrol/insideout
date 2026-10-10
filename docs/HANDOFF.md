# HANDOFF

Resume here. The authoritative multi-task board is
[`docs/plans/README.md`](plans/README.md). The target product experience is
[`PRODUCT.md`](../PRODUCT.md), and the running system is described by
[`docs/architecture/`](architecture/index.md). Ordinary changes are
recorded in [`docs/changelogs/`](changelogs/README.md) and open debt in
[`docs/issues/`](issues/README.md) — this file holds only the resume
path and current warnings, never history.

## What to do next

**The review-3 remediation is finished; the owner steps below are all that
remain.** Review 3 (83 findings, 17 High) was executed as four verified waves
through `4464330` (150 commits): every cluster worked red-test-first in its
own worktree and passed an independent adversarial verifier; the
[disposition ledger](review/2026-10-10-disposition-ledger/04-multirole-3.md)
closes 73 of 83 rows by code, 3 by the report itself, parks 1 (CQ-04), and
leaves 6 owner-blocked. The [changelog](changelogs/2026-10-11-rv3-remediation/index.md)
has the wave detail and migration notes. Engineering work is done; resume
only for a new finding or an owner request.

The close-the-open-gaps plan finished 2026-10-10:
the review-register sweep landed as `a84feb8` (CI green, run 38047747110,
5/5 jobs; GitHub, MBP2 and MBP verified on one commit), its
[disposition ledger](review/2026-10-10-disposition-ledger/README.md) closed
at 187 findings and 0 unresolved
([server](changelogs/2026-10-10-close-open-gaps/review-sweep-server.md),
[client](changelogs/2026-10-10-close-open-gaps/review-sweep-client.md)
halves). What the [issues index](issues/README.md) lists as left is a choice
or a secret, not unfinished code, and it is the owner's:

1. **Ship.** Nothing from 2026-10-08 on is deployed: the hosted web bundle and
   server predate the block-craft slices and the remediation waves. `railway
   up --service server` applies **seven** migrations on start — the four
   `20261010*` files (`20261010100000`–`20261010130000`; the invite-code
   rotation replaces workspace invite codes shorter than 32 characters, so an
   outstanding short code stops working and admins read the new one in the
   workspace) plus `20261011100000_circuit_breaker_row_id` (drops and
   recreates the two circuit functions with a row-id parameter),
   `20261011110000_conversation_locale` and
   `20261011140000_roadmap_validation` (both additive). All seven apply
   together on a fresh clone of the migrated base, and CI's upgrade-replay
   lane rehearses them on production-shaped old data. Then `railway up
   --service app`, the two requests in
   [BUG-027](issues/2026-10-10-bug-027-nginx-resolved-the-api-upstream-once.md#owner-check-after-the-deploy),
   and a signed-in page load.
2. **Destructive or secret steps:** purging the 582 legacy users;
   `INSIDEOUT_MAIL_API_KEY` and `INSIDEOUT_MAIL_FROM` in Railway plus a
   redeploy; a store-signed Android bundle (needs the keystore); the Supabase
   automated-backup toggle; the GitHub App install; a live critic exercise
   (needs an LLM key). Each is on its [board](plans/README.md) row.
   Mail/lead-15 state (review-3 probes, 2026-10-10): invitation-only signup
   verified live and the hosted binary verified pre-wave; the mail-key
   setting is owner-blocked (Railway-only), and the board's email-auth row
   still records "all five flows exercised live" (2026-08-24) — the owner
   must reconcile the two when setting the keys.
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
