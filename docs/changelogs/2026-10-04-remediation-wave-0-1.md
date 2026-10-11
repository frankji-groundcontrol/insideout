# 2026-10-04 — Remediation Waves 0–1 (review follow-up)

Executes Waves 0 and 1 of the multirole review's
[`40-remediation-plan.md`](../review/2026-10-04-multirole/40-remediation-plan.md)
(register IDs below). Plan of record:
[`docs/plans/2026-10-04-remediation-wave-0-1.md`](../plans/2026-10-04-remediation-wave-0-1.md).

## Wave 1 — client unblocks (behavior changes, test-first)

- **FS-02 — invite deep link.** `/register?invite=` now reaches the register
  form: the router reads the query parameter (`client/lib/router.dart`),
  `RegisterPage`/`LandingPage` carry it, and `AuthForm` pre-fills its invite
  controller. Admin-copied invitation links work end-to-end under
  invitation-only signup. Pinned by a widget test in
  `client/test/routes_test.dart`.
- **FS-03 — local-dev CORS.** `scripts/dev.sh` exports
  `INSIDEOUT_CORS_ORIGINS=localhost` for the server component when unset
  (explicit setting wins), so the documented `flutter run -d chrome`
  quickstart reaches the local API from any loopback port. Documented in
  `docs/usage/local-development.md` and `docs/architecture/backend.md`.
- **USER-02 — expired-token recovery.** `client/lib/api/client.dart` gains a
  response-interceptor recovery: on a non-`/auth/` 401 it refreshes once
  (single-flight across concurrent 401s), retries with the rotated token, and
  on refresh failure clears the session so the app routes to the auth door
  (`Session.clear()` now nulls the user and notifies). Four new
  loopback-HTTP tests in `client/test/api_token_refresh_test.dart` cover
  retry-success, refresh-failure, no-loop, and single-flight — real wire I/O,
  no mocks.

## Wave 0 — doors and records

- **TEST-01** — deleted the orphan `INSIDEOUT_GH_CLIENT_ID/SECRET`
  declarations from `.env.example`; the env-catalog suite is green again
  (47/47, was red since the GitHub App slice).
- **ARCH-01** — compose carries dev-only defaults for the two bundled-pg
  passwords so `docker compose config` parses for remote-DSN setups; the
  bundled postgres now binds `127.0.0.1` only (SEC-07); README's "whole
  stack" claim corrected to "backend stack (postgres + server)".
- **USER-01** — README quick start, `docs/SETENV.md`, and
  `docs/usage/local-development.md` now state registration is
  invitation-only by default and how to bootstrap (`promote-admin` +
  invites, or `signup-mode open` for local dev).
- **PM-02/ARCH-03** — `docs/HANDOFF.md`: merged-PR item replaced by the
  pending release gate, list renumbered, MCP tool count corrected to 50;
  the two Cobalt board rows are Completed with the gate tracked separately.
- **ARCH-02** — `docs/INIT.md` reframed as a historical compatibility
  pointer (Nuxt 4 + Ink & Seal era), routing to `PRODUCT.md` and
  `docs/architecture/`.
- **ARCH-07/PM-04** — `docs/TODO.md` slimmed to the shipped baseline; the
  Nuxt-era delta tracker marked superseded (record survives in git history).
- **PM-03** — README core-features note: the section describes the current
  build; `PRODUCT.md` holds the target experience.
- **PM-06** — `PRODUCT.md` brand commitments updated: Cobalt is the shipped
  visual identity; Ink & Seal superseded 2026-09-18.
- **PM-08/SEC-08/CRIT-02/ARCH-06** — redaction sweep of live docs (dated
  changelogs kept as history): Railway hostnames and the production domain
  replaced with placeholders in README, SKILL.md, client/README, HANDOFF,
  and usage docs; the CLI and MCP binaries now default to
  `http://127.0.0.1:8080/api/v1` instead of the hosted API (**operator
  note**: scripts that relied on the old default must set `INSIDEOUT_API`).
- **CRIT-01** — README third-party assets corrected: the repo ships OFL
  Noto Sans SC / Noto Serif SC variable fonts for native builds (web loads
  Google Fonts); PuHuiTi notes reframed as historical attribution. Added
  `client/assets/fonts/OFL.txt` (SIL OFL 1.1 full text).
- **CRIT-05** — CLAUDE.md points known limitations at
  `docs/issues/README.md` + the plans board; `docs/index.md` TODO.md
  description updated to match.
- **L6 residual** — stale Nuxt-as-current references fixed in
  `.env.example` (CORS comments), `docs/architecture/backend.md` (proxy
  paragraph), and two misleading test names.
- **0.12 deferred** — the untracked local `app/.env` leftover was **not**
  deleted by the agent: unreadable by design and irreversible. Delete it
  locally when convenient (`rm -rf app/`).

## Verification

- `docker compose config -q` exits 0 (was the reproduced Wave-0 failure).
- `python3 -m unittest test_env_catalog test_env_writes`: **47 + 21 passed,
  0 failed** (catalog was 46 passed / 1 failed).
- `cd client && flutter test`: **63/63 pass** (54 before + 6 new + 3
  pre-existing recount); `flutter analyze`: only pre-existing info-level
  lints, none in the touched files.
- `cd server && go build ./... && go vet ./... && go test ./...`: green.
- Redaction greps over live docs and `server/cmd/`: clean.
- `bash -n scripts/dev.sh`; `./scripts/env.sh check server`: 5 ok.

## Operator action

- Rebuild local CLI/MCP binaries if you use them against the hosted API —
  set `INSIDEOUT_API` explicitly from now on.
- The release gate stays open: deploy `railway up --service app` and verify
  a signed-in page load, then refresh HANDOFF.

  *Superseded 2026-10-11: closed — the 2026-10-11 owner deploy shipped both
  services and the hosted probes pass, HANDOFF refreshed
  ([owner-deploy record](2026-10-11-owner-deploy-and-mail-truth.md)).*
- Remaining waves (server hardening, truth-in-advertising, CI) are untouched
  — see the remediation plan, Waves 2–4.
