# 2026-10-04 — Multirole review (whole tree)

**Status:** Finished (report + records closed; remediation not started —
that is new work, one plan per wave when picked up).
**Scope:** `main` @ `877ad03` (604 files). Five roles + five dimensions +
completeness critic, one finding schema, skeptic verification of every
High/Critical, reproduction of the top findings.
**Report folder:** [`docs/review/2026-10-04-multirole/`](../review/2026-10-04-multirole/README.md)
(first review under `docs/review/`).

## Checklist

- [x] Branch enumeration + product identification (`main` @ `877ad03`;
      `cursor/vibrant-ink-seal-198b` fully merged, identical file count).
- [x] Scout routers, packaging, entry points, tests, deploy files.
- [x] Leads written and cheap leads run (results below).
- [x] Fan out 10 reviewers (each writes its role file + returns JSON findings).
- [x] Dedup; skeptic verdicts for every High/Critical (12 confirmed, 2 downgraded, 0 refuted).
- [x] Reproduce the top findings by running code (compose parse, red env suite, flag grep, skip census, invitation grep).
- [x] Completeness critic (5 verified adds: CRIT-01…05; score calibration ai-developer 7.2→5.8; census corrected to 188 top-level / 43 SKIP).
- [x] Redact; write folder (README, executive summary, register, remediation
      plan); link from `docs/review/README.md` and `docs/index.md`.
- [x] Close records: dated changelog, issues index line, plans board, learning note.

**Outcome:** 75 unique findings (0 Critical · 12 High · 25 Medium · 38 Low).
Report: [`docs/review/2026-10-04-multirole/`](../review/2026-10-04-multirole/README.md).
Next: remediation is executed wave by wave from
[`40-remediation-plan.md`](../review/2026-10-04-multirole/40-remediation-plan.md)
— open a new plan when a wave is picked up.

## Leads with run results

- L1 **confirmed**: no CI — `.github/workflows/` absent; nothing gates merges.
- L2 **confirmed**: README overpromises `docker compose up -d` ("whole stack"):
  compose has only postgres+server (Flutter is not a compose service), and
  `docker compose config` currently **fails** — the `:?` guards on
  `POSTGRES_OWNER_PASSWORD` / `POSTGRES_APP_PASSWORD` fire even when
  `DATABASE_URL` targets a remote instance, so the file cannot be parsed.
- L3 **confirmed**: README core features describe the superseded product model
  (four rigid Coach stages; workspace-admin approves/rejects PRDs) — both are
  listed under "Superseded assumptions" in `PRODUCT.md`.
- L4 **confirmed**: `PRODUCT.md` brand commitments still mandate Ink & Seal;
  shipped Flutter theme is Cobalt 1.0.0 (`client/lib/theme/cobalt*.dart`,
  `docs/design-system/cobalt/`); `docs/index.md` confirms Cobalt live.
- L5 **confirmed**: `docs/HANDOFF.md` stale ("Next: merge PR #3" though merged
  2026-09-18; list numbering 1,2,2,4,5; cites 27 MCP tools vs 50 on the board);
  this board's two Cobalt rows still say next action "Merge PR #3".
- L6 **confirmed**: stale Nuxt references in live surfaces: `.env.example`
  (DEV_CORS "Nuxt proxy" comment), `docs/architecture/{index,frontend,backend}.md`,
  `docs/INIT.md`, `README.md`, `scripts/env_catalog.py`,
  `client/test/{api_errors_test,appearance_test}.dart`.
- L7 **confirmed**: size-rule breaches (>350 lines): `apiclient.go` 478,
  `agent/coach.go` 410, `agent/anthropic.go` 406; `client/lib/api/client.dart`
  529, `api/models.dart` 472. Issues track only the two apiclient files
  (`docs/issues/2026-09-03-apiclient-line-budget.md`,
  `2026-09-18-flutter-apiclient-line-budget.md`).
- L8 **confirmed**: `server/Dockerfile` has no `USER` (root on distroless) and
  no `HEALTHCHECK`; the compose server service has no healthcheck either.
- L9 **confirmed**: compose server service does not forward the mail, GitHub
  App/webhook, CORS, cookie, or TTL variables from the `.env.example` catalog —
  mail and GitHub hooks are unconfigurable under compose.
- L10 **confirmed**: untracked leftover `app/.env` on disk from the deleted
  Nuxt app; AGENTS.md requires untracked secret-bearing files to be recorded
  in a local untracked-file manifest.
- L11 **confirmed**: deployment identifiers in tracked docs (Railway app host
  `README.md:64`; production custom domain in `docs/HANDOFF.md` and
  changelogs) versus the AGENTS.md redaction rule. The review report refers to
  them only as `<railway-host>` / `<prod-domain>`.
- L12 **confirmed clean**: `go build ./...`, `go vet ./...`, `go test ./...`
  all pass; store DB-gated integration tests skip without `DATABASE_URL`.
- L13 **not-run**: `flutter analyze` + `flutter test` (board claims green
  2026-09-18) — owned by the testing dimension.
- L14 **not-run**: hosted Railway deploy likely behind `main` (PR #3 merged
  2026-09-18; autodeploy off; no deploy record since) — records check only.
- L15 **confirmed clean**: key-shaped secret scan over tracked files finds only
  placeholders and test fixtures.
