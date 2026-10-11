# HANDOFF

Resume here. The authoritative multi-task board is
[`docs/plans/README.md`](plans/README.md). The target product experience is
[`PRODUCT.md`](../PRODUCT.md), and the running system is described by
[`docs/architecture/`](architecture/index.md). Ordinary changes are
recorded in [`docs/changelogs/`](changelogs/README.md) and open debt in
[`docs/issues/`](issues/README.md) — this file holds only the resume
path and current warnings, never history.

## What to do next

**Everything engineering is closed; the owner's account actions below are
all that remain.** 2026-10-11: the deploy (server @ `b6c34c4`, app @
`6b1f24f`), the Resend re-key and the full mailed-flow pass are done — the seven pending
migrations applied (confirmed by running `insideout migrate` against the
hosted database — "already up to date"), `/api/v1/auth/config` reports
`mailEnabled`, the hosted read-only smoke probes pass, BUG-027's owner
checks passed both (a server-only redeploy answered `401` with no app
restart — the restart ritual below is dropped), and one real LLM coach turn
ran on the hosted box (model answered, 45 deltas, clean `message_end`).
The review-3 remediation and its records are
[on `main`](changelogs/2026-10-11-rv3-remediation/index.md).

1. ~~Re-key Resend~~ **Done 2026-10-11** — the owner's fresh send-scoped
   key is synced to Railway + `.env` + both hosts' stores; the admin mail
   probe answers `status:"sent"` and every mailed flow ran live through the
   provider (invitation, magic link, login code, forgot/reset — each token
   read from the delivered email and redeemed; transcript in the
   [changelog](changelogs/2026-10-11-owner-deploy-and-mail-truth.md)).
   Note for the future: the key is send-scoped (`/domains` answers 401 —
   that is the restriction, not a dead key), and the verified sender
   domain delivers to any address; the Resend account's own address
   (`frank.ji@…`) and `scihill.com` addresses deliver, the old rehearsal
   address `dingjue.ji@…` bounces.
2. **Still the owner's, console-only:** the Supabase automated-backup
   toggle (dashboard — a schema-scoped snapshot was taken 2026-10-11
   before the purge) and a store-signed Android bundle (needs the
   keystore). The legacy-user purge is **done** (859 junk accounts,
   keep-set enumerated — [changelog](changelogs/2026-10-11-owner-deploy-and-mail-truth.md)),
   and the GitHub App is already installed (`insideout-roadmap`).
3. **MBP:** closed 2026-10-11 — Flutter 3.44.8 (arm64) installed and
   `pub get --enforce-lockfile` passes with the lockfile unwritten;
   the leftover `app/.env` is gone
   ([drift issue](issues/2026-10-09-host-setup-drift-between-macs.md)).

Every other thread is closed with its exact status and links on its
[board](plans/README.md) row. Local run commands:
[`docs/usage/local-development.md`](usage/local-development.md).

Every other thread is closed with its exact status and links on its
[board](plans/README.md) row. Local run commands:
[`docs/usage/local-development.md`](usage/local-development.md).

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
- **GitHub Actions billing (2026-10-11):** the repository is now
  **public**, so Actions are free and CI runs normally (the head run is
  green); the ten billing-blocked runs were purged as superseded. A
  failed payment may still be sitting on the account for other
  GitHub services — worth a look in Settings → Billing.
- **Shipping:** Railway autodeploy is off for both services, and
  `railway redeploy` re-runs the last image instead of building. Ship landed
  changes explicitly: any `server/` change with `railway up --service server`,
  any `client/` change with `railway up --service app`. The BUG-027 restart
  ritual is **retired 2026-10-11**: the deployed nginx re-resolves the API
  upstream, so a server-only deploy answers without touching `app`
  (owner check 2 passed — a server redeploy kept `/api/v1/me` at 401
  throughout).

## Non-negotiables

- Never read or expose `.env` or `.env.local`.
- DB-dependent behavior requires a real PostgreSQL check and must stay
  inside the `insideout` schema.
- Commands and environment-safe launch instructions live in
  [`docs/usage/local-development.md`](usage/local-development.md).
