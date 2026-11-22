# HANDOFF

Resume here. The authoritative multi-task board is
[`docs/plans/README.md`](./plans/README.md). The target product experience is
[`PRODUCT.md`](../PRODUCT.md), and the running system is described by
[`docs/architecture/`](./architecture/index.md). Ordinary changes are
recorded in [`docs/changelogs/`](./changelogs/README.md) and open debt in
[`docs/issues/`](./issues/README.md) — this file holds only the resume
path and current warnings, never history.

## What to do next

**2026-11-22: the roadmap follows "people in charge" on every surface**
([changelog](changelogs/2026-11-22-roadmap-graph-and-dependencies.md)).
Nodes can depend on each other. The Canvas shows one level of steps in
dependency order, with pan and zoom, and the hierarchy is reached by opening
a step. `GET /roadmap/steps`, `roadmap steps` and MCP `roadmap_steps` give
the same order, pinned by one shared fixture. `mine` and the dashboard's "My
work" open on what each person is in charge of, and agents brief on their
human's work. Opening a node shows it as its person in charge sees it, with a banner of their first glance and a toggle for their nodes ([changelog](changelogs/2026-11-22-view-as-the-person-in-charge.md)). A click on a node opens it in a movable, resizable window; a double-click drills in ([changelog](changelogs/2026-11-22-node-windows.md)). The local dev database has the dependencies migration applied.

**2026-11-21: the review 5 remediation is merged to `main`** (#24,
`a945784`). Every gate was green on the integrated tree
([final state](changelogs/2026-11-21-rv5-remediation/index.md#final-state)):
the build path checks before it writes; the task tree for roles works on web,
REST, CLI and MCP; the four surfaces reach the same routes under one error
style, with a parity test; the code follows a responsibility rule with a
function-length guard; and the documentation graph is checked, with Marksman
at 0. Owner decision 7 is built: a node writer, and their agent, may file
proposals inside their subtree
([changelog](changelogs/2026-11-21-proposals-inside-a-delegates-subtree.md)). The gaps the remediation left open are closed too
([changelog](changelogs/2026-11-21-close-review-5-gaps.md)): no function is
over the function budget, every store error answers through one helper, the
key manifests are one shared set, and the parity walk reads nested fields. CI is disabled in the repository at the owner's
request: the `ci`, `security`, `smoke` and `ubuntu install skill` workflows
are off ([how to re-enable](usage/local-development.md)); verification is
local.

**2026-11-21: focused review 5 is delivered.**
[The report](./review/2026-11-21-focused-review-5/README.md) (127 findings,
21 High) answers five questions from the owner. The task tree for roles is
one quarter built: a node's assigned owner is refused on every write inside
it, no node opens as its own tree, and delegation is web-only
([target design](./review/2026-11-21-focused-review-5/41-task-tree-target-design.md)).
Three build-path defects lose or misplace data. The order is the
[remediation plan](./review/2026-11-21-focused-review-5/40-remediation-plan.md).
The owner's decisions are recorded in the
[open-findings record](./issues/2026-11-21-review-5-open-findings.md).

**2026-10-20: the install front door is the user guide, and `insideout setup` asks numbered questions.**
A person starts at [the user guide](./usage/user-guide.md). The wizard
paths are maintained in [setup paths](./usage/setup-paths.md).
`insideout forgot-password` waits on a watch id and notices when the
reset link is used somewhere else
([record](./changelogs/2026-10-21-reset-watch.md)). The developer
key files are not part of that guide. Agents use
[insideout-setup](../skills/insideout-setup/SKILL.md) for the hosted
service and [insideout-server](../skills/insideout-server/SKILL.md) only
when someone asked to run the server. Record:
[install front door](./changelogs/2026-10-20-install-front-door.md).

**2026-11-21: wave 4b and the adopted decisions are on `origin/main` at
`4d186da9`** — the owner directed the sync from MBP and the push
([sync record](./changelogs/2026-11-21-wave-4b-sync-and-push.md)).
[The report](./review/2026-10-18-multirole-4/README.md) (145 findings,
10 High) says the core held but the front door and the gates did not. The
fixes were built on `rv4/integration` from `main` @ `d53035a`. Waves 0 to 4
are on `origin/main` at `e6a3d14`. Their tested code head
is `97cf4c06`. The gate re-run is in
[wave-4.md](./changelogs/2026-10-18-rv4-remediation/wave-4.md). The first CI
run on `e6a3d14` is red on `env-suites` only. Wave 4b landed at `4e16d1b`
([wave 4b](./changelogs/2026-10-18-rv4-remediation/wave-4b.md)):
the four code clusters and the five living-doc files; it reached
`origin/main` 2026-11-21 within `4d186da9`. What
landed from the owner settings is in
[the decisions record](./changelogs/2026-10-20-rv4-decisions.md). The
[execution contract](./review/2026-10-18-multirole-4/41-remediation-contract.md)
and the [plan](./review/2026-10-18-multirole-4/40-remediation-plan.md) still
govern. `TestProjectQuietFlagged` was red on `main` from 2026-10-16 until
`e6a3d14`; the clock fix is on `main`. Do not tag a release until the
release-job test gate from plan row 1.5 has run in CI. Before the first
deploy, confirm the hosted `DATABASE_URL` is the app role (`serve` now
refuses a role that bypasses row-level security), set
`INSIDEOUT_TRUSTED_PROXY_HOPS=1`, inspect the hosted app role for migration
`20261018200000` (the users read grant) and deploy the server before the MCP
binary. The wave-4b `record_fact` database gate was run by the verifier
on a private Postgres at `127.0.0.1:55442`, not re-run in this checkout.
Board row:
[plan](./plans/2026-10-18-multirole-4.md); open records:
[issues](./issues/2026-10-18-review-4-open-findings.md).

Retention and erasure (rv4-DATA-06) are on `origin/main` (pushed 2026-11-21): a coach conversation lives 365 days after its newest message, erasure is a tombstone, and `DELETE /api/v1/me` requires the current password and is human-only ([record](./issues/2026-10-18-rv4-retention-erasure.md), [changelog](./changelogs/2026-10-20-retention-and-erasure.md)).

Session-swap tools are withheld while a login is held. Marking a roadmap node done needs evidence or a one-sentence reason ([changelog](./changelogs/2026-10-20-completion-evidence-or-reason.md), migration `20261020150000`). A PATCH that omits `ownerId` keeps the project owner ([changelog](./changelogs/2026-10-20-omitted-project-owner.md)). The keep-as-built decisions are in [the adopted record](./issues/2026-10-20-rv4-decisions-adopted.md). The verifier ran the new database tests on private Postgres. This checkout did not re-run them. Still the owner's console: the hosted database role, `INSIDEOUT_TRUSTED_PROXY_HOPS=1`, GitHub `deployment_status`, key rotation, and the Railway `VERSION` variable.

**Engineering is not closed: nine product-promise slices are open, and the
owner's account actions below remain.** The slices are what
[PRODUCT.md](../PRODUCT.md) promises and the running system does not
implement; take them in the
[recommended order](./issues/2026-10-11-product-promise-gaps.md), starting with
the keystone (Product Branch, Merge and the Decision Log), and open a dated
plan on the [board](./plans/README.md) before editing. The 2026-10-11 docs-order pass (survey, folder READMEs,
index consistency, Marksman cross-references — zero orphan docs) landed
with its [changelog](./changelogs/2026-10-11-docs-order-marksman.md) and
closed [plan](./plans/2026-10-11-docs-marksman-order.md). 2026-10-11: the deploy (server @ `b6c34c4`, app @
`6b1f24f`), the Resend re-key and the full mailed-flow pass are done — the seven pending
migrations applied (confirmed by running `insideout migrate` against the
hosted database — "already up to date"), `/api/v1/auth/config` reports
`mailEnabled`, the hosted read-only smoke probes pass, BUG-027's owner
checks passed both (a server-only redeploy answered `401` with no app
restart — the restart ritual below is dropped), and one real LLM coach turn
ran on the hosted box (model answered, 45 deltas, clean `message_end`).
The review-3 remediation and its records are
[on `main`](./changelogs/2026-10-11-rv3-remediation/index.md).
The 2026-10-12 guided key workflow is complete in the local worktree:
[plan](./plans/2026-10-12-guided-key-workflow.md), [setup tutorial](./usage/key-setup-tutorial.md),
[testing tutorial](./usage/key-testing-tutorial.md), and
[changelog](./changelogs/2026-10-12-guided-key-workflow.md).

1. ~~Re-key Resend~~ **Done 2026-10-11** — the owner's fresh send-scoped
   key is synced to Railway + `.env` + both hosts' stores; the admin mail
   probe answers `status:"sent"` and every mailed flow ran live through the
   provider (invitation, magic link, login code, forgot/reset — each token
   read from the delivered email and redeemed; transcript in the
   [changelog](./changelogs/2026-10-11-owner-deploy-and-mail-truth.md)).
   Note for the future: the key is send-scoped (`/domains` answers 401 —
   that is the restriction, not a dead key), and the verified sender
   domain delivers to any address; the Resend account's own address
   (`<admin-email>`) and `<owner-domain>` addresses deliver, the old
   rehearsal address `<rehearsal-account>` bounces.
2. **Still the owner's, console-only:** the Supabase automated-backup
   toggle (dashboard — a schema-scoped snapshot was taken 2026-10-11
   before the purge) and a store-signed Android bundle (needs the
   keystore). The legacy-user purge is **done** (859 junk accounts,
   keep-set enumerated — [changelog](./changelogs/2026-10-11-owner-deploy-and-mail-truth.md)),
   and the GitHub App is already installed (`insideout-roadmap`).
3. **MBP:** closed 2026-10-11 — Flutter 3.44.8 (arm64) installed and
   `pub get --enforce-lockfile` passes with the lockfile unwritten;
   the leftover `app/.env` is gone
   ([drift issue](./issues/2026-10-09-host-setup-drift-between-macs.md)).

Every other thread is closed with its exact status and links on its
[board](./plans/README.md) row. Local run commands:
[`docs/usage/local-development.md`](./usage/local-development.md).

## Warnings

- **`main` is meant to be one commit on GitHub, MBP, and MBP2**; confirm with
  the three-way check in the
  [sync practice](./practices/2026-10-09-sync-two-macs-and-github.md).
  2026-11-21: GitHub and MBP2 carry the sync record commits on top of
  `4d186da9`; MBP stays at `4d186da9` with a whole uncommitted feature
  slice (install front door, reset watch, reset code — 35 tracked files
  plus new files). Commit that slice on MBP before the next sync
  ([record](./changelogs/2026-11-21-wave-4b-sync-and-push.md)). Untracked
  `.grok/` is tool scratch, held out by rule and listed in the local manifest
  `.git/info/untracked-manifest.tsv`.
  **Do not `git add .`** — stage and checkpoint one task at a time, and do not
  revert unrelated edits.
- **Flutter SDK pin:** the repo pins Flutter 3.44.x (CI and the Dockerfile run
  3.44.0); MBP was brought onto 3.44.8 on 2026-10-11 (item 3 above). A host on
  another SDK rewrites `client/pubspec.lock` on `flutter pub get`; CI and the
  Dockerfile use `pub get --enforce-lockfile`, which fails on a drifted
  lockfile, so never commit a rewritten `client/pubspec.lock`.
- **GitHub Actions billing (2026-10-11):** the repository is now
  **public**, so Actions are free and CI runs normally. Last green `ci`
  run: `9f0b1c7` (2026-10-11). The run on `d53035a` (2026-10-18) was red on
  the server job (`TestProjectQuietFlagged`); that clock fix is on `main`
  as of `e6a3d14`. The first CI run on `e6a3d14` is red on `env-suites`
  only. Wave 4b's fetch-depth fix (`460c33cf`) is on `origin/main` since
  2026-11-21. On the pushed tip `4d186da9` the `security` workflow is red on
  [govulncheck toolchain findings](./issues/2026-11-21-govulncheck-go-toolchain.md)
  (gitleaks passed); the `ci` run was still in progress at recording time. The ten
  billing-blocked runs were purged as superseded. A failed payment may still be sitting on the account for other
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
  [`docs/usage/local-development.md`](./usage/local-development.md).
