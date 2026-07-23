# BUG-004: Docker Compose rejects `${VAR:-default}` when `default` itself contains `${...}`

**Found**: 2026-07-20, during the InsideOut rewrite (P7), running `docker compose build` for the first time against the newly-written `docker-compose.yml`.

**Symptom**: `invalid interpolation format for services.server.environment.DATABASE_URL. You may need to escape any $ with another $.`

**Root cause**: `docker-compose.yml` tried to give `DATABASE_URL` a derived default that referenced another variable's own default: `${DATABASE_URL:-postgres://insideout_app:${POSTGRES_APP_PASSWORD:-insideout_app_dev_password}@postgres:5432/insideout?sslmode=disable}`. Compose's variable interpolation does not support nesting `${...}` inside the default-value branch of another `${VAR:-default}` — the default is treated as a literal string, not further interpolated, at any level of nesting.

**Fix**: stopped trying to derive `DATABASE_URL` from other compose variables. It's now a required variable (`${DATABASE_URL:?set DATABASE_URL in .env}`) that must be set explicitly in `.env` — either pointing at a remote instance, or at the bundled `postgres` service using the same password as `POSTGRES_APP_PASSWORD`. `.env.example` documents both cases with concrete example values. See `docker-compose.yml` and `.env.example`.

**Why it matters**: Compose Spec variable interpolation is single-level only — any future compose file changes that try to build one variable's default out of another variable's `${...}` reference will hit this same error. Prefer requiring the variable explicitly (`:?`) over trying to derive a clever default when more than one variable is involved.
