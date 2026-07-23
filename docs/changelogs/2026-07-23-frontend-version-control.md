# 2026-07-23 — Bring the frontend rewrite + infra under version control

The Nuxt 4 SSR frontend rewrite and the deployment infra had been built and
verified across the 2026-07-20 → 2026-07-23 changelogs but were **never tracked
by git**. Following the [backend commit](2026-07-23-backend-version-control.md),
this change tracks the frontend and infra — completing the cutover so
`origin/main` now holds the whole new stack (frontend + Go backend + infra).

## What changed

- **Staged all of [`app/`](../../app/)**: the workshop → prd/projects/workspace
  feature migration, the 「Assembly」 landing (Ink & Seal), the detachable PRD
  coach sidebar with markdown rendering, and the idea-shaping copy reframe.
- **Removed [`supabase/`](../../supabase/)**: the old Supabase edge functions,
  migrations, and tests — superseded by the Go backend. This completes the
  2026-07-20 RLS cutover; the new frontend talks to the Go API via
  `services/api/`, and no `supabase`/`workshop` imports remain in `src/`.
- **Added deployment infra**: [`docker-compose.yml`](../../docker-compose.yml)
  (postgres + server + app), [`docker/postgres-init/`](../../docker/postgres-init/)
  (non-superuser `insideout_app` role provisioning), and
  [`app/Dockerfile`](../../app/Dockerfile) + `.dockerignore`.
- **Self-hosted fonts**: `app/public/fonts/` — 4 weights of Alibaba PuHuiTi
  (Latin + full CJK), all referenced by `@font-face` in `src/style.css`.

## Review & verification

- **Security sweep (inline)**: no secrets, hosts, or PII anywhere in `app/`
  source or infra. `docker-compose.yml` uses `${VAR:?}`/`${VAR:-dev-default}`
  placeholders throughout — `DATABASE_URL`, `INSIDEOUT_JWT_SECRET`,
  `POSTGRES_APP_PASSWORD` are all required-from-`.env` with no committed value;
  the only default is the clearly-marked local bootstrap `insideout_dev_password`.
- **No dangling references**: grep confirms zero remaining imports of the
  deleted `lib/supabase`, `services/supabase`, `stores/workshop`,
  `components/workshop`, or `features/workshop` paths in `src/`.
- **Build already verified**: typecheck, tests, production build, and browser
  (light + dark) all passed in the prior session for this exact tree.

## Operator notes

This completes the cutover: `origin/main` now has the new frontend, the new Go
backend, and the infra to run the whole stack (`docker compose up`). The old
Supabase backend is gone from the repo. Known follow-up: the 4 self-hosted TTF
fonts total ~32 MB — a WOFF2 conversion (+ CJK subsetting) would shrink that
considerably and is recorded as a future optimization, not done here to avoid
re-verifying rendering right before the push.
