# 40 · Remediation plan

Waves ordered by impact over effort; start at Wave 0, row one. Register IDs
refer to [01-findings-register.md](01-findings-register.md). Every wave names
its runnable check. Nothing here requires new product judgment — each row is
a verified defect with a named change.

## Wave 0 — doors and records (one day, mostly docs; land as focused commits)

| # | IDs | File | Change | Diff size |
|---|---|---|---|---|
| 0.1 | TEST-01 | `.env.example:131-132` | Delete the two orphan `INSIDEOUT_GH_CLIENT_ID/SECRET` declarations (no consumer tree-wide). | 2 lines |
| 0.2 | ARCH-01, L2 | `docker-compose.yml:17-18`, `README.md:72-77` | Replace `:?` guards with `:-` dev defaults (the postgres-init script already hard-fails on reuse of defaults); correct "whole stack" to "backend stack (postgres + server)". | ~6 lines |
| 0.3 | USER-01 | `README.md:41-45`, `docs/SETENV.md`, `docs/usage/local-development.md` | Insert the signup-policy step: default is invitation-only; bootstrap = `promote-admin` then mint invites, or `signup-mode open` for local dev. Link `docs/usage/cli-mcp-and-guidance.md:64-69`. | ~15 lines |
| 0.4 | PM-02, ARCH-03 | `docs/HANDOFF.md`, `docs/plans/README.md` | Remove "merge PR #3" next actions; renumber HANDOFF; sync the MCP tool count to 50; record deploy state honestly. | ~20 lines |
| 0.5 | ARCH-02 | `docs/INIT.md:25-27` | Rewrite "Current technical foundation" for Go + Flutter/Cobalt, or add a staleness banner pointing at `docs/architecture/`. | ~8 lines |
| 0.6 | ARCH-07 | `docs/TODO.md` | Refresh the delta to the shipped state; mark Nuxt-era items historical or retire to `docs/history/`. | ~30 lines |
| 0.7 | PM-03 | `README.md:29` | One line under 核心功能 noting the section describes the current build while `PRODUCT.md` holds the target (which supersedes rigid stages and admin-approves-all). | 2 lines |
| 0.8 | PM-06 | `PRODUCT.md:436` | Brand commitments: Ink & Seal → Cobalt, with a pointer to `docs/design-system/cobalt/`. | ~5 lines |
| 0.9 | PM-08, SEC-08, CRIT-02 | `README.md:64`, `docs/HANDOFF.md`, `SKILL.md:19`, `client/README.md`, changelogs | Redaction sweep per AGENTS.md: replace the Railway app host and production custom domain with placeholders; keep CLI/MCP defaults off the hosted API (ARCH-06: default `127.0.0.1:8080`). | ~15 lines |
| 0.10 | CRIT-01 | `README.md:88-99`, `client/assets/fonts/` | State the repo ships OFL Noto variable fonts for native targets; add OFL.txt beside the binaries. | ~10 lines + license file |
| 0.11 | CRIT-05 | `CLAUDE.md:15` | Point "known limitations" at `docs/plans/README.md`; restore rule-for-rule parity with AGENTS.md. | ~5 lines |
| 0.12 | L10 | `app/.env` (untracked) | Delete the leftover Nuxt-era `app/` directory from disk; no manifest entry needed once gone. | local only |

**Check:** `docker compose config -q` exits 0; `python3 -m unittest
test_env_catalog` = 47 passed / 0 failed; `rg -n 'up\.railway\.app' --no-ignore
README.md SKILL.md docs/HANDOFF.md client/README.md` returns nothing, and a
manual sweep confirms the production custom domain appears nowhere in those
files (per the AGENTS.md redaction rule).

## Wave 1 — client unblocks (one day, one PR)

| # | IDs | File | Change | Diff size |
|---|---|---|---|---|
| 1.1 | FS-02 | `client/lib/router.dart:28`, `register_page.dart`, `auth_form.dart` | Thread `s.uri.queryParameters['invite']` into the register form and pre-fill the invite controller — mirror the adjacent magic-link/reset routes. | ~10 lines |
| 1.2 | FS-03 | `scripts/dev.sh` or `docs/usage/local-development.md` | Export `INSIDEOUT_CORS_ORIGINS=localhost` in the documented dev path (the `localhost` token already matches any port) or document the one-liner. | 1-3 lines |
| 1.3 | USER-02 | `client/lib/api/client.dart:35-43` | Add an `onError` interceptor: on 401, refresh once and retry the request; on refresh failure, sign out to the auth door. | ~40 lines + test |

**Check:** `cd client && flutter test` stays green; manual walk — fresh
invite link registers end-to-end; app survives >15 min idle without a reload.

## Wave 2 — server hardening (two days)

| # | IDs | File | Change | Diff size |
|---|---|---|---|---|
| 2.1 | SEC-01 | `server/internal/api/server.go`, new middleware | Per-IP + per-account throttle on login/register/refresh/magic-link/code endpoints (attempt counter with backoff; reuse the login-code 5-attempt budget pattern). Target design already in [`docs/issues/2026-08-24-auth-endpoints-no-rate-limiting.md`](../../issues/2026-08-24-auth-endpoints-no-rate-limiting.md) — close that issue when this lands. | ~80 lines + tests |
| 2.2 | SEC-02 | `server/internal/api/auth_email_flows.go:36` | Per-email + per-IP send cooldown (one per 60 s) before sending; keep the uniform 200. | ~20 lines |
| 2.3 | FS-07, CRIT-03 | `server/Dockerfile`, `client/Dockerfile`, `docker-compose.yml` | `USER nonroot:nonroot` (server), unprivileged nginx (client), healthchecks on `/healthz` both services. | ~10 lines |
| 2.4 | ARCH-05 | `docker-compose.yml:38-52` | Forward the full env catalog with `:-` defaults (mail, GitHub webhook/App, CORS, cookie, TTLs, PUBLIC_URL). | ~10 lines |
| 2.5 | AIDEV-01 | `server/internal/agent/taxonomy.go:38` | Guard the transient retry with `streamed.Len() == 0` (the pattern the context-length retry already uses) or reset the stream before re-issuing. | ~5 lines |
| 2.6 | REL-06 | `server/internal/agent/coach.go:43-46` | Read `AI_MAX_CONCURRENT` / `AI_QUEUE_WAIT` from env, current values as defaults. | ~15 lines |
| 2.7 | REL-08 | `server/internal/agent/sse.go:39` | Return and act on SSE write errors (abort the turn on a dead client). | ~15 lines |
| 2.8 | REL-02 | `server/internal/mail/mail.go:49` | Dev-mode log keeps to/subject, drops the body (the link); update the `.env.example` line that advertises "links included". | 2 lines |
| 2.9 | ARCH-04 | `server/internal/config/config.go:78` | `INSIDEOUT_PUBLIC_URL`: fail loudly when mail delivery is enabled with the localhost default still set. | ~5 lines |

**Check:** `go build ./... && go vet ./... && go test ./...` green; exercise
the coach SSE endpoint against a live server (repo practice) and confirm a
killed client aborts the turn; `docker compose up -d` boots with mail vars
set.

## Wave 3 — truth in advertising (two days; small product decisions)

| # | IDs | File | Change | Diff size |
|---|---|---|---|---|
| 3.1 | FEAT-01 | `server/internal/api/projects.go`, board UI | Compute staleness from `LatestUpdateAt` against a threshold, expose `flagged`, render a badge — or cut the README sentence until it ships. | ~40 lines or 1 line |
| 3.2 | FEAT-02 | `client/lib/features/workspace/workspace_page.dart:128` | Render `latestUpdateKind`/`latestUpdateContent` (already parsed in models.dart:116-126) plus a relative timestamp on each board card. | ~30 lines |
| 3.3 | FEAT-03 | `client/lib/features/prd/export_page.dart:71` | Trigger a real file download for markdown export on web. | ~15 lines |
| 3.4 | USER-05, FEAT-04 | `server/internal/store/ideas.go:65`, `PRODUCT.md` | Product decision: filter ideas to the author (PRODUCT.md private-by-default) or update the docs to state workspace-shared. Record the decision either way. | ~10 lines or docs |
| 3.5 | FEAT-05 | roadmap UI | Render `GET /roadmap/progress` as the default roadmap view (server-side pressure levels already exist). | ~60 lines |

**Check:** board walk shows flag/badge and latest update; export downloads a
file; the idea-privacy decision is recorded in `docs/plans/` + changelog.

## Wave 4 — verification spine (one day)

| # | IDs | File | Change | Diff size |
|---|---|---|---|---|
| 4.1 | PM-05, TEST-02 | `.github/workflows/ci.yml` (new) | Minimal CI: `go build/vet/test`, `flutter analyze + test`, env suites, on pull_request. First run must fail on TEST-01 unless Wave 0.1 landed. | ~40 lines |
| 4.2 | TEST-02 | CI job step | `services: postgres` + `DATABASE_URL` so the 43 skipped RLS tests execute on every PR. | ~15 lines |
| 4.3 | TEST-04 | `server/internal/api/conversations_test.go` (new) | httptest POST against a stubbed coach asserting incremental flushed SSE frames. | ~60 lines |
| 4.4 | TEST-05 | `server/internal/agentcontext/agentcontext_test.go:42` | Turn the log-only branch into `t.Errorf`. | 2 lines |
| 4.5 | CQ-02 | `client/test/feature_wiring_test.dart` | Demote to lint-tier backstop; add behavioral widget tests per the `_FakeApi` pattern. | ~40 lines |
| 4.6 | CQ-01 | `docs/issues/` | Four dated modularity issues for the untracked oversize files (coach.go, anthropic.go, client.dart, models.dart). | 4 small files |

**Check:** CI goes green on the PR that lands it; `go test ./...` in CI shows
0 SKIP for the store package.

## Not in scope for this release

New PRODUCT.md frontier features (Product Branches, Decision Log UI,
multi-provider Git); accessibility and performance/load programs; native
iOS/Android release (blocked on SDK, already on the board); purging the 582
legacy users (operator decision); the nginx resolver rework beyond filing the
dated issue (REL-07/FS-06); live-provider coach evals. Revisit after the top
ten close and the deploy gate (PM next_first) has run once.
