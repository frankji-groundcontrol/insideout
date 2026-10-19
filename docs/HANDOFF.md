# HANDOFF

Resume here. The authoritative multi-task board is
[`docs/plans/README.md`](plans/README.md). The target product experience is
[`PRODUCT.md`](../PRODUCT.md), and the running system is described by
[`docs/architecture/`](architecture/index.md). Ordinary changes are
recorded in [`docs/changelogs/`](changelogs/README.md) and open debt in
[`docs/issues/`](issues/README.md) — this file holds only the resume
path and current warnings, never history.

## What to do next

**2026-10-19: multirole review 4 is delivered and its remediation is under
way.** [The report](review/2026-10-18-multirole-4/README.md) (145 findings,
10 High) says the core held but the front door and the gates did not. The
fixes are four waves on the local branch `rv4/integration` (base `main` @
`d53035a`, nothing pushed). Waves 0 to 4 are integrated. Wave 4 head is
`97cf4c06` (499 commits above `d53035a`); the gate re-run is in
[wave-4.md](changelogs/2026-10-18-rv4-remediation/wave-4.md). Resume at the
open prose to-dos and the wave-4b carry-forward in that file. Do not push
until the owner says so. The
[execution contract](review/2026-10-18-multirole-4/41-remediation-contract.md)
and the [plan](review/2026-10-18-multirole-4/40-remediation-plan.md) still
govern. `main`
still has the red test (`TestProjectQuietFlagged`, wall clock, red since
2026-10-16) until that branch lands; do not tag a release from `main` before
it does (the release test gate is plan row 1.5). The first CI run after a push
settles every workflow property of waves 1 and 3. Before the first deploy of
the branch, confirm the hosted `DATABASE_URL` is the app role (`serve` now
refuses a role that bypasses row-level security), set
`INSIDEOUT_TRUSTED_PROXY_HOPS=1`, inspect the hosted app role for migration
`20261018200000` (the users read grant) and deploy the server before the MCP
binary. Board row: [plan](plans/2026-10-18-multirole-4.md); open records:
[issues](issues/2026-10-18-review-4-open-findings.md).

**Engineering is not closed: nine product-promise slices are open, and the
owner's account actions below remain.** The slices are what
[PRODUCT.md](../PRODUCT.md) promises and the running system does not
implement; take them in the
[recommended order](issues/2026-10-11-product-promise-gaps.md), starting with
the keystone (Product Branch, Merge and the Decision Log), and open a dated
plan on the [board](plans/README.md) before editing. The 2026-10-11 docs-order pass (survey, folder READMEs,
index consistency, Marksman cross-references — zero orphan docs) landed
with its [changelog](changelogs/2026-10-11-docs-order-marksman.md) and
closed [plan](plans/2026-10-11-docs-marksman-order.md). 2026-10-11: the deploy (server @ `b6c34c4`, app @
`6b1f24f`), the Resend re-key and the full mailed-flow pass are done — the seven pending
migrations applied (confirmed by running `insideout migrate` against the
hosted database — "already up to date"), `/api/v1/auth/config` reports
`mailEnabled`, the hosted read-only smoke probes pass, BUG-027's owner
checks passed both (a server-only redeploy answered `401` with no app
restart — the restart ritual below is dropped), and one real LLM coach turn
ran on the hosted box (model answered, 45 deltas, clean `message_end`).
The review-3 remediation and its records are
[on `main`](changelogs/2026-10-11-rv3-remediation/index.md).
The 2026-10-12 guided key workflow is complete in the local worktree:
[plan](plans/2026-10-12-guided-key-workflow.md), [setup tutorial](usage/key-setup-tutorial.md),
[testing tutorial](usage/key-testing-tutorial.md), and
[changelog](changelogs/2026-10-12-guided-key-workflow.md).

1. ~~Re-key Resend~~ **Done 2026-10-11** — the owner's fresh send-scoped
   key is synced to Railway + `.env` + both hosts' stores; the admin mail
   probe answers `status:"sent"` and every mailed flow ran live through the
   provider (invitation, magic link, login code, forgot/reset — each token
   read from the delivered email and redeemed; transcript in the
   [changelog](changelogs/2026-10-11-owner-deploy-and-mail-truth.md)).
   Note for the future: the key is send-scoped (`/domains` answers 401 —
   that is the restriction, not a dead key), and the verified sender
   domain delivers to any address; the Resend account's own address
   (`<admin-email>`) and `<owner-domain>` addresses deliver, the old
   rehearsal address `<rehearsal-account>` bounces.
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

## Warnings

- **`main` is meant to be one commit on GitHub, MBP, and MBP2**; confirm with
  the three-way check in the
  [sync practice](practices/2026-10-09-sync-two-macs-and-github.md). Untracked
  `.grok/` and `.qoder/` are tool scratch and raw agent journals, held out by
  rule and listed in the local manifest `.git/info/untracked-manifest.tsv`.
  **Do not `git add .`** — stage and checkpoint one task at a time, and do not
  revert unrelated edits.
- **Flutter SDK pin:** the repo pins Flutter 3.44.x (CI and the Dockerfile run
  3.44.0); MBP was brought onto 3.44.8 on 2026-10-11 (item 3 above). A host on
  another SDK rewrites `client/pubspec.lock` on `flutter pub get`; CI and the
  Dockerfile use `pub get --enforce-lockfile`, which fails on a drifted
  lockfile, so never commit a rewritten `client/pubspec.lock`.
- **GitHub Actions billing (2026-10-11):** the repository is now
  **public**, so Actions are free and CI runs normally. Last green `ci`
  run: `9f0b1c7` (2026-10-11); the run on `d53035a` (2026-10-18) is red on
  the server job (`TestProjectQuietFlagged`), fixed on `rv4/integration` and
  not yet on `main`. The ten billing-blocked runs were purged as
  superseded. A failed payment may still be sitting on the account for other
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
