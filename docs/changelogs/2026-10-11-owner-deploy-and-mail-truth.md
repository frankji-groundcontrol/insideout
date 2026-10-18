# 2026-10-11 — The owner gap closure: deploy, live probes, first release, and the mail truth

The owner directed full closure of the review-3 owner-blocked rows
([plan](../plans/2026-10-11-owner-gap-closure.md)). Deploy-dependent rows
closed with live evidence; the mail rows ran live and found a revoked key —
every copy of it — so they close to a single, precisely-scoped owner
credential. Nothing here changed source; the one tree change is record truth
(the client README's deploy claim, the backend layout tree the truth check
caught, README's release section) plus this record set.

## Deploy (SEC-01, PM-02, PM-04, USER-02 deploy half, BUG-027)

`railway up` shipped `main` @ `b6c34c4` to both services on 2026-10-11
(server 09:36 +08, app 09:58 +08; build `69053a11`).

- **Migrations**: `insideout migrate` run against the hosted database
  through the service env (`railway run`, values never printed) reports
  *"nothing to do, already up to date"* — the seven pending migrations
  (four `20261010*` + three `20261011*`) applied at boot.
- **`/api/v1/auth/config`** now answers `{"mailEnabled":true,
  "signupMode":"invitation_only"}` — PM-04's first check passes against a
  real field.
- **BUG-027 owner checks both pass**: check 1 (`/api/v1/me` → 401), then a
  server-only redeploy answered 401 continuously for 30+ seconds with no
  app restart and no 504 — the deployed nginx re-resolves. The restart
  ritual is dropped from [HANDOFF](../HANDOFF.md).
- **Hosted read-only smoke**: `SMOKE_HOSTED=1 SMOKE_BASE=https://
  insideout.yalotein.net ./scripts/smoke.sh` — 6/6 PASS (SPA, HTML, healthz,
  auth config, 401).

## Live funnel rehearsal (USER-02, FEAT-03, REL-06, SEC-05, PM-01)

Every step ran against production through the real endpoints, with the
owner-pool bootstrap smoke.sh itself uses as the sanctioned pattern:

1. One invitation minted through the owner pool (the app's own
   delete-then-insert + sha256-hex code hash; one-off program under
   `railway run`, never committed).
2. **Register 201** with that invite code through the public API.
3. Account promoted via `insideout promote-admin` (the sanctioned path),
   exercised, then **demoted back to `member`** — the owner's real account
   is sole admin again (verified by a fresh login).
4. **Admin mail probe**: `status:"error", "mail: resend returned 401
   Unauthorized"` — the deployed key is dead. This probe is the delivery
   signal review-3 ARCH-03 built, and it worked.
5. **Every stored copy of `INSIDEOUT_MAIL_API_KEY` is revoked**: Railway's
   (401 via probe), the root `.env`'s, MBP2's key store's, and MBP's key
   store's — each tested directly against Resend's API, each rejected.
   Minting a replacement needs the owner's Resend login, which exists on
   no machine here (automation browser logged out of Resend and GitHub;
   the owner's Chrome profile is TCC-protected). **Re-key is the one open
   owner step** — procedure in [HANDOFF](../HANDOFF.md) item 1.
6. **One real LLM coach turn on the hosted box**: workspace → idea →
   convert → POST message; SSE `message_start` (model answered,
   `offline:false`) → 45 `delta` frames → `message_end`. The LLM quartet
   in Railway is live and valid.

PM-01 closes on this: the three disagreeing mail records (HANDOFF, the
board's email-auth row, the 2026-10-05 rotation row) now state one truth —
the 2026-08-24 "all five flows exercised live" predates any key (dev-mode
log), the 2026-08-25/2026-10-05 keys are revoked everywhere, and the
funnel's live state is what the probe says.

## First release (USER-05)

- `v0.1.0` tagged at `b6c34c4` — the exact deployed commit.
- The release workflow could not start: GitHub Actions reports the
  account's billing/spending limit. The assets were built locally with the
  workflow's own commands (four CGO-off targets, both binaries per tarball)
  and published via `gh release create` — release
  [v0.1.0](https://github.com/frankji-groundcontrol/insideout/releases/tag/v0.1.0),
  assets download- and sha256-verified.
- **The anonymous `curl | bash` one-liner still 404s — the repository is
  private**, and GitHub answers `releases/latest/download` with 404 for
  private repos. README now states exactly this; making the repo public is
  the owner's visibility call and the only thing between the one-liner and
  working.

## Records

- [HANDOFF](../HANDOFF.md) — resume path is now the Resend re-key; the
  BUG-027 restart ritual is retired.
- [Disposition ledger](../review/2026-10-10-disposition-ledger/04-multirole-3.md)
  — the nine owner-half rows re-dispositioned with today's evidence.
- [BUG-027](../issues/2026-10-10-bug-027-nginx-resolved-the-api-upstream-once.md)
  — live verification recorded.
- Board rows (plans README): the email-auth row reconciled; the
  close-open-gaps remainder updated (deploys done).
- `check_docs_truth.py` green; it caught one stale tree (backend.md missing
  `internal/linebudget/`) — fixed here.

### The deferred engineering item

[Throttled roadmap copy](../issues/2026-10-10-throttled-roadmap-copy-deferred.md)
— the board's one deliberately-deferred engineering item — is closed
red-first in the same pass: `guardWrite` now maps `APP_THROTTLE` /
`CIRCUIT_OPEN` to the coach's own countdown copy (both proved failing
before the fix), keeping the generic line for every other error; client
suite 458/458, analyze at the 14 baseline, format clean.

### The MBP host drift closed too

MBP came back online: `app/.env` confirmed gone, and Flutter **3.44.8
(arm64)** installed at `~/flutter` with the PATH wired —
`flutter pub get --enforce-lockfile` passes there and the lockfile stays
unwritten ([drift issue](../issues/2026-10-09-host-setup-drift-between-macs.md),
including the Intel-vs-arm64 archive gotcha). All three machines and
GitHub sit on one commit again.

## The mailed-flow pass (FEAT-03, REL-06 — closed)

The owner handed over a fresh **send-scoped** Resend key (its `/domains`
401 is the scope restriction, not a dead key) via the MBP2 key store. It
was synced to Railway (auto-redeploy `d6ccd21d`), the root `.env`, and
MBP's store, then every mailed flow ran live on production — each email's
token was read back from the provider and redeemed:

| Flow | Evidence |
| --- | --- |
| Admin mail probe | `{"status":"sent","mailEnabled":true}` |
| Invitation | minted via the admin API → email delivered (verified-domain sender) → **emailed code == inline code** → redeemed by `register` 201 |
| Magic link | requested → token extracted from the delivered email → `magic-link/consume` → 200, tokens issued |
| Login code | requested (after the resend cooldown — the first attempt was correctly swallowed) → 6-digit code read from the email → `login-code/verify` → 200, tokens issued |
| Forgot/reset | requested → token extracted → `reset-password` → 200 → **login with the new password** → 200 |

Delivery facts learned: the verified sender domain
(`INSIDEOUT_MAIL_FROM`) delivers to any address; the Resend account's own
address delivers; the first rehearsal address bounces (provider-side), and
a suppressed send is visible in the provider's events. The flow account is
`<owner-email>` (a member on the owner's domain; its password is
not recorded anywhere); the first rehearsal account was demoted back to
member after its admin leg — sole-admin posture holds.

## Minimal CI, public repo, legacy purge, and the App (owner-directed)

Per the owner's direction — *only run the necessary [CI], otherwise purge;
for the rest of the open ones, use the recommended choice*:

- **CI minimized**: the one necessary run (head `b2cdfca`) re-ran and is
  **green** once the repo went public (Actions are free for public repos —
  the payment failure no longer gates CI). Ten superseded billing-blocked
  runs were purged (8 stale CI + the release run whose assets I had built
  locally).
- **Repository is public** (the recommended choice — the product is built
  for it: installer-first README, releases, free CI). Guarded by a privacy
  sweep first: no `.env` ever committed, no tokens or private keys in the
  tree or the full 1,153-commit history (the only pattern hits are a PEM
  format mention in a code comment and the Supabase **anon** key, public
  by design). Anonymous install path verified end-to-end: bootstrap 200,
  tarball sha256 OK, anonymous clone OK.
- **Legacy purge executed** after a schema-scoped snapshot
  (`pg_dump --schema=insideout`, 46 tables — the shared instance's other
  schemas are unreadable to our role and untouched). The census first:
  864 users = 840 `@test.local` fixtures + 19 old-app/simulation accounts
  never seen since + **5 real accounts kept** (the owner-admin, the
  dogfood account that owns the GitHub binding, the active demo account,
  and the two funnel-rehearsal accounts — a date-only rule would have
  deleted the dogfood binding, so the keep-set was enumerated, not
  dated). One transaction deleted 859 users, 446 workspaces, 214
  projects and their rows; post-purge hosted probes 6/6 green.
- **GitHub App: already installed** — `insideout-roadmap` (the board's
  install item was stale), installation live on the owner's account,
  verified with the app's own JWT.
- Still owner-console only: the Supabase automated-backup **toggle**
  (dashboard; today's snapshot mitigates) and the Android **signing
  keystore** (secret material).

## Verification

Full gates (all CI lanes) ran green on the deployed `b6c34c4` (run
38077887705, 7/7). The records commit `2929074` is docs-only; GitHub
Actions could not start its CI run — the account's billing/spending limit
failed (same cause as the release workflow above; annotation: "recent
account payments have failed or your spending limit needs to be
increased"). Locally verified in its place: `check_docs_truth.py` green,
all new/changed relative links resolve. Re-run CI when the billing is
settled (`gh run rerun 38105183080 --failed`).

## What an operator of an existing deployment must do

*Corrected 2026-10-11: the re-key is done — this record's "The mailed-flow
pass" section (above) records the fresh send-scoped key synced everywhere
and every mailed flow run live; the sentence below predates that close.*

Nothing new to apply — the deploy is live. The one open step is the Resend
re-key in [HANDOFF](../HANDOFF.md); the rehearsal account
(`<rehearsal-account>`) stays as a plain member with its workspace, and its
password is not recorded anywhere.
