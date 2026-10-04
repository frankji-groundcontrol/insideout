# 2026-10-04 — Remediation Waves 0–1 (multirole review follow-up)

**Status:** Finished (all rows landed and verified; 0.12 deferred to the user)
**Source:** [`docs/review/2026-10-04-multirole/40-remediation-plan.md`](../review/2026-10-04-multirole/40-remediation-plan.md)
Waves 0 and 1. Register IDs below refer to
[`01-findings-register.md`](../review/2026-10-04-multirole/01-findings-register.md).

## Wave 0 — doors and records

- [x] 0.1 TEST-01 — delete orphan `INSIDEOUT_GH_CLIENT_ID/SECRET` lines in
      `.env.example` (unblocks the red env-catalog suite).
- [x] 0.2 ARCH-01 — compose `:?` → `:-` dev defaults for the two bundled-pg
      passwords; README "whole stack" → "backend stack (postgres + server)".
- [x] 0.3 USER-01 — README/SETENV/local-development: signup-policy step
      (invitation-only default; `promote-admin` + invites, or `signup-mode open`
      for local dev; link cli-mcp-and-guidance.md).
- [x] 0.4 PM-02/ARCH-03 — HANDOFF: drop "merge PR #3", renumber, MCP tool
      count 50, honest deploy state; plans board Cobalt rows updated.
- [x] 0.5 ARCH-02 — `docs/INIT.md`: stop presenting Nuxt 4 + Ink & Seal as
      current; point at Go + Flutter/Cobalt architecture.
- [x] 0.6 ARCH-07/PM-04 — `docs/TODO.md`: slim rewrite to the shipped
      baseline; Nuxt-era delta marked historical (path kept, links intact).
- [x] 0.7 PM-03 — README: one line under 核心功能 noting current build vs
      PRODUCT.md target.
- [x] 0.8 PM-06 — PRODUCT.md brand commitments: Ink & Seal → Cobalt.
- [x] 0.9 PM-08/SEC-08/CRIT-02/ARCH-06 — redaction sweep (live docs only;
      dated changelogs stay as history): Railway host + prod domain →
      placeholders in README, HANDOFF, SKILL.md, client/README, usage docs;
      CLI/MCP default API base → `http://127.0.0.1:8080`.
- [x] 0.10 CRIT-01 — README fonts claim corrected (OFL Noto variable fonts
      ship for native targets); add `client/assets/fonts/OFL.txt`.
- [x] 0.11 CRIT-05 — CLAUDE.md: known-limitations pointer → plans board;
      restore parity with AGENTS.md.
- [x] 0.12 L10 — **deferred to user**: deleting the untracked local
      `app/.env` is irreversible and its contents are unreadable by design;
      flagged in the final report instead of deleted by the agent.

## Wave 1 — client unblocks (TDD)

- [x] 1.1 FS-02 — thread `?invite=` through router → RegisterPage → AuthForm
      (mirror magic-link pattern); failing test first in routes/auth tests.
- [x] 1.2 FS-03 — `scripts/dev.sh` exports
      `INSIDEOUT_CORS_ORIGINS=localhost` when unset (dev-only); doc line via
      0.3/0.9 edits in local-development.md.
- [x] 1.3 USER-02 — Dio `onError` interceptor: refresh-once-and-retry on 401,
      sign out on refresh failure; failing test first.

## Verification (coordinator, after both agents)

- [x] `docker compose config -q` exits 0
- [x] `python3 -m unittest test_env_catalog test_env_writes` in `scripts/` green (47 + 21)
- [x] `cd client && flutter test` green (63/63)
- [x] `cd server && go build ./... && go vet ./... && go test ./...` green
- [x] redaction greps return nothing on live docs
- [x] changelog `docs/changelogs/2026-10-04-remediation-wave-0-1.md` + index,
      board updated, this plan closed
