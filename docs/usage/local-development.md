# Local Development

Developer setup for InsideOut: a Go 1.25 API server (`server/`) and the
Flutter client (`client/`, web + iOS + Android), backed by PostgreSQL.

This doc supersedes the old `docs/INSTALL.md` (removed 2026-07-21; see git history).

## Prerequisites

- **Go 1.25+** (see `server/go.mod`)
- **Flutter 3** (for `client/`; see [`client/README.md`](../../client/README.md))
- **PostgreSQL 14+** reachable via `DATABASE_URL` — either your own instance or the bundled one below
- **Docker** (optional) — only needed for the bundled Postgres / full-compose runs

## Environment

Interactive setup — prompts for the keys that need a choice, generates the
JWT secret on request, never prints values (manual alternative:
`cp .env.example .env` and edit):

```bash
./scripts/env.sh init
```

The two required values are `DATABASE_URL` and `INSIDEOUT_JWT_SECRET`
(min 32 chars) — they are the only two left uncommented in `.env.example`,
because they are the only two the server refuses to boot without.
`./scripts/env.sh check` validates them at any time — and shows the defaults
in effect for every unset optional, tagged by consumer — while the `dev.sh`
wrapper below runs that check automatically before launching.
`./scripts/env.sh edit` lists every variable with its live state (set,
default-from-the-skeleton, missing, placeholder, unset) and lets you set or
clear one; secrets stay masked both on screen and while typing.

Neither process loads the root `.env` itself — the Go server reads plain
environment variables; docker-compose interpolates the file directly.
Flutter uses `--dart-define=API_BASE`. For a bare `go run`/`go test`,
use the root wrapper that preflights `env.sh check <component>`, exports
`.env`, and then execs inside the directory you name (it never prints
values):

```bash
./scripts/dev.sh -C server go run ./cmd/insideout                              # dev server
./scripts/dev.sh -C server go test ./internal/store/... -run TestAuthz -v      # integration tests
./scripts/dev.sh -C client flutter run -d chrome --dart-define=API_BASE=http://127.0.0.1:8080/api/v1
```

The full variable reference — every variable grouped by consumer, with
required/default/meaning, the `.env`→process bridges, database setups,
recipes, and troubleshooting — lives in
[environment.md](environment.md). docker-compose topology vars are in
[deployment.md](deployment.md).

### Offline AI mode

If `INSIDEOUT_LLM_API_KEY` is unset, the server logs
`INSIDEOUT_LLM_API_KEY not set — using offline template-reply coach` and swaps
in a template-reply streamer (`server/internal/agent`). The PRD coach works
end-to-end (stages, tools, SSE) with canned replies — no network, no cost.
This is the recommended default for local dev (see the recipes in
[environment.md](environment.md)).

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

Then, from the repo root (the wrapper exports `.env` first):

```bash
./scripts/dev.sh -C server go run ./cmd/insideout migrate   # apply embedded SQL migrations (server/db/migrations/) and exit
./scripts/dev.sh -C server go run ./cmd/insideout seed      # optional: demo user, workspace, project, idea, PRD
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

Server (from the repo root):

```bash
./scripts/dev.sh -C server go run ./cmd/insideout   # listens on INSIDEOUT_ADDR, default :8080
```

Remember `INSIDEOUT_COOKIE_SECURE=0` when serving over plain http locally.

Frontend (from `client/`):

```bash
flutter run -d chrome --dart-define=API_BASE=http://127.0.0.1:8080/api/v1
```

Or `./scripts/dev.sh -C client flutter run -d chrome --dart-define=API_BASE=http://127.0.0.1:8080/api/v1`.
Hosted web uses same-origin `/api/v1` behind nginx
([deployment.md](deployment.md#railway-current-public-deploy)).

## Testing

Backend:

```bash
go test ./...                                          # pure unit tests, no DB needed (from server/)
./scripts/dev.sh -C server go test ./internal/store/... -run TestAuthz -v   # store/RLS integration tests (repo root)
```

The integration tests skip themselves when `DATABASE_URL` is unset. Run them
against a migrated database (they exercise the RLS policies).

Live end-to-end smoke test (all five surfaces over real HTTP, no mocks — needs
`curl` + `jq` and a reachable `DATABASE_URL` in `../.env`):

```bash
./server/scripts/smoke.sh                              # boots its own server on a random high port
SMOKE_BASE=http://127.0.0.1:54321 ./server/scripts/smoke.sh   # reuse an already-running server
```

It registers fresh uniquely-named users each run (rerun-safe) and exits non-zero
on any failed assertion. See
[docs/changelogs/2026-07-27-live-smoke-test.md](../changelogs/2026-07-27-live-smoke-test.md).

Frontend (from `client/`):

```bash
flutter analyze --no-fatal-infos
flutter test
```
