# Local Development

Developer setup for InsideOut: a Go 1.25 API server (`server/`) and a Nuxt 4
Universal SSR frontend (`app/`), backed by PostgreSQL.

This doc supersedes the old `docs/INSTALL.md` (removed 2026-07-21; see git history).

## Prerequisites

- **Go 1.25+** (see `server/go.mod`)
- **Node 22 + pnpm** (the app Dockerfile builds on `node:22-alpine`; pnpm is the package manager)
- **PostgreSQL 14+** reachable via `DATABASE_URL` — either your own instance or the bundled one below
- **Docker** (optional) — only needed for the bundled Postgres / full-compose runs

## Environment

Copy the example file and edit it:

```bash
cp .env.example .env
```

The Go server reads plain environment variables (it does not load `.env`
itself — docker-compose does; for a bare `go run`, export them or use a
wrapper like `env $(cat .env | xargs)`).

Authoritative list (`server/internal/config/config.go`):

| Variable | Required | Default | Meaning |
|----------|----------|---------|---------|
| `DATABASE_URL` | **yes** | — | Postgres connection string |
| `INSIDEOUT_JWT_SECRET` | **yes** | — | JWT signing secret, **min 32 chars** (fail-fast) |
| `INSIDEOUT_ADDR` | no | `:8080` | HTTP listen address |
| `INSIDEOUT_ACCESS_TTL` | no | `15m` | Access-token lifetime (Go duration) |
| `INSIDEOUT_REFRESH_TTL` | no | `720h` | Refresh-token lifetime (Go duration) |
| `ANTHROPIC_BASE_URL` | no | — | Anthropic-Messages-API-compatible endpoint |
| `ANTHROPIC_AUTH_TOKEN` | no | — | AI auth token; **unset = offline mode** (see below) |
| `AI_MODEL` | no | `claude-sonnet-4-20250514` | Model id sent to the AI endpoint |
| `INSIDEOUT_COOKIE_SECURE` | no | on | Set `0` for plain-http local dev, or the browser drops the auth cookies |
| `INSIDEOUT_DEV_CORS` | no | off | Set `1` to enable permissive CORS in dev |

docker-compose additionally reads `POSTGRES_PORT`, `POSTGRES_APP_PASSWORD`,
`POSTGRES_SUPERUSER_PASSWORD`, `SERVER_PORT`, `APP_PORT` — see
[deployment.md](deployment.md).

The frontend/Nitro reads one variable: `NUXT_API_INTERNAL_BASE` (default
`http://127.0.0.1:8080/api/v1`), the internal base URL of the Go API.

### Offline AI mode

If `ANTHROPIC_AUTH_TOKEN` is unset, the server logs
`ANTHROPIC_AUTH_TOKEN not set — using offline template-reply coach` and swaps
in a template-reply streamer (`server/internal/agent`). The PRD coach works
end-to-end (stages, tools, SSE) with canned replies — no network, no cost.
This is the recommended default for local dev.

## Database setup

Two provisioning models, same migrations:

**(a) Bundled docker-compose Postgres** — `docker compose up -d postgres`
brings up `postgres:17` on host port `${POSTGRES_PORT:-5442}`. On first boot,
`docker/postgres-init/001-create-app-role.sh` creates a non-superuser
`insideout_app` role and makes it OWNER of the `insideout` database. Point
`DATABASE_URL` at it:

```
DATABASE_URL=postgres://insideout_app:<POSTGRES_APP_PASSWORD>@localhost:5442/insideout?sslmode=disable
```

**(b) Existing / shared instance** — any Postgres 14+ host. On a dedicated
database, the `insideout_app` role should own the database; on a shared
multi-tenant instance, `insideout_app` owns only the `insideout` schema.
Migrations create and stay inside the `insideout` schema — they never touch
`public` objects belonging to other tenants.

Then, from `server/`:

```bash
go run ./cmd/insideout migrate   # apply embedded SQL migrations (server/db/migrations/) and exit
go run ./cmd/insideout seed      # optional: demo user, workspace, project, idea, PRD
```

`seed` logs the demo login on completion: `demo@insideout.local` /
`demo12345` (hashed through the same argon2id path the server uses).
Migrations also run automatically on server startup, so `migrate` is mainly
for CI or checking a fresh database.

**Transaction poolers:** if `DATABASE_URL` contains `pgbouncer=true`
(PgBouncer/Supavisor transaction mode), the server automatically switches pgx
to the simple query protocol (`server/internal/store/pool.go`) because
server-side prepared statements don't survive connection multiplexing. No
action needed — just don't strip that query parameter.

## Running

Server (from `server/`):

```bash
go run ./cmd/insideout            # listens on INSIDEOUT_ADDR, default :8080
```

Remember `INSIDEOUT_COOKIE_SECURE=0` when serving over plain http locally.

Frontend (from `app/`):

```bash
pnpm install
pnpm dev                          # Nuxt dev server on :3000
```

The browser only ever talks to the Nuxt origin; Nitro proxies `/api/v1/**` to
the Go server at `NUXT_API_INTERNAL_BASE`. If your Go server isn't at
`http://127.0.0.1:8080/api/v1`, set that variable before `pnpm dev`.

## Testing

Backend (from `server/`):

```bash
go test ./...                                          # pure unit tests, no DB needed
DATABASE_URL=... go test ./internal/store/... -run TestAuthz -v   # store/RLS integration tests
```

The integration tests skip themselves when `DATABASE_URL` is unset. Run them
against a migrated database (they exercise the RLS policies).

Frontend (from `app/`):

```bash
pnpm test              # vitest --run
npx nuxi typecheck     # type checking (dev server does not block on type errors)
```
