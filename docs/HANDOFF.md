# HANDOFF

Resume here. The authoritative multi-task board is
[`docs/plans/README.md`](plans/README.md). The target product experience is
[`PRODUCT.md`](../PRODUCT.md), and the running system is described by
[`docs/architecture/`](architecture/index.md). Ordinary changes are
recorded in [`docs/changelogs/`](changelogs/README.md) and open debt in
[`docs/issues/`](issues/README.md) — this file holds only the resume
path and current warnings, never history.

## What to do next

**The deploy is done; re-keying Resend is the one open step.** On 2026-10-11
`railway up` shipped `main` @ `b6c34c4` to both services: the seven pending
migrations applied (confirmed by running `insideout migrate` against the
hosted database — "already up to date"), `/api/v1/auth/config` reports
`mailEnabled`, the hosted read-only smoke probes pass, BUG-027's owner
checks passed both (a server-only redeploy answered `401` with no app
restart — the restart ritual below is dropped), and one real LLM coach turn
ran on the hosted box (model answered, 45 deltas, clean `message_end`).
The review-3 remediation and its records are
[on `main`](changelogs/2026-10-11-rv3-remediation/index.md).

1. **Re-key Resend — the mail funnel's only blocker.** The live admin mail
   probe (2026-10-11, transcript in the
   [changelog](changelogs/2026-10-11-owner-deploy-and-mail-truth.md))
   reports `resend returned 401 Unauthorized`: every stored copy of
   `INSIDEOUT_MAIL_API_KEY` is revoked — Railway's, the root `.env`'s, and
   both hosts' key stores were each tested and rejected. Sign in to Resend,
   mint a fresh key, set it in Railway (`railway variables --service server
   --set "INSIDEOUT_MAIL_API_KEY=…"` + redeploy) and in the three local
   homes, then re-run the probe (expect `status: "sent"`), send one real
   magic link and one password reset to the operator's address, and click
   both through. Until then invitations still work via the inline code the
   admin API returns; the emailed flows (magic link, login code, reset)
   log-only.
2. **Destructive or secret steps still the owner's:** purging the 582 legacy
   users; a store-signed Android bundle (needs the keystore); the Supabase
   automated-backup toggle; the GitHub App install. Each is on its
   [board](plans/README.md) row.
3. **MBP:** install Flutter 3.44.x
   ([issue](issues/2026-10-09-host-setup-drift-between-macs.md)) and check that
   the leftover `app/.env` from Wave 0.12 is gone (absent on MBP2, not
   verified on MBP).

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
- **GitHub Actions billing (2026-10-11):** CI and the release workflow
  cannot start — GitHub reports failed account payments / spending limit
  (runs 38104710052 and 38105183080). The `v0.1.0` assets were built and
  published locally instead. Re-run `gh run rerun 38105183080 --failed`
  once billing is settled; `2929074` is docs-only and locally verified.
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
