# Local Development

Developer setup for InsideOut: a Go 1.25 API server (`server/`) and the
Flutter client (`client/`, web + iOS + Android), backed by PostgreSQL.

This doc supersedes the old `docs/INSTALL.md` (removed 2026-07-21; see git history).

## Prerequisites

- **Go 1.25+** (see `server/go.mod`)
- **Flutter 3.44.x** (for `client/`; CI and `client/Dockerfile` pin 3.44.0; see [`client/README.md`](../../client/README.md))
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
./scripts/dev.sh -C client flutter run -d chrome --web-port=5173 --dart-define=API_BASE=http://127.0.0.1:8080/api/v1
```

The server builds invite, magic-link and reset links from
`INSIDEOUT_PUBLIC_URL` (default `http://localhost:5173`; `scripts/dev.sh`
logs them in dev), so the web dev server must run on that port (`flutter run`
picks a random one unless `--web-port` pins it) or `INSIDEOUT_PUBLIC_URL`
must be set to the port in use.

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

### Mail dev mode

If `INSIDEOUT_MAIL_API_KEY` is unset, the server delivers no email: sign-in
links, sign-in codes and password-reset mails are only written to the server
log (`mail dev mode — not sent`). The sign-in screens still offer them, with a
notice that mail is not configured on this server. To follow a link or code
locally, read it from that log: `scripts/dev.sh` turns on
`INSIDEOUT_MAIL_DEV_BODY` for the server, which adds the message body to the
log line. Both variables are in the Go backend table of
[environment.md](environment.md#go-backend) (`INSIDEOUT_MAIL_API_KEY` and
`INSIDEOUT_MAIL_DEV_BODY` rows).

## Database setup

Two provisioning models, same migrations:

**(a) Bundled docker-compose Postgres** — `docker compose up -d postgres`
brings up `postgres:18` on host port `${POSTGRES_PORT:-5442}`. On first boot,
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
./scripts/dev.sh -C server go run ./cmd/insideout seed      # optional: demo user, workspace, project, idea, PRD (safe to rerun)
```

`seed` logs the demo login on completion: `demo@insideout.local` /
`demo12345` (hashed through the same argon2id path the server uses).
It is safe to rerun: when the demo user already has the "InsideOut Demo
Team" workspace it logs `seed: already seeded` and creates nothing. It is
dev-only, so never point it at a shared or production database.
Migrations also run automatically on server startup, so `migrate` is mainly
for CI or checking a fresh database.

**Registration is invitation-only on a fresh database** (migrations seed
`signup_mode = invitation_only`; `register` returns 403 without an invite
code). Bootstrap the first admin with `insideout promote-admin <email>`
(server subcommand, owner DSN) and mint per-email invites — or, for local
dev only, `insideout signup-mode open`. See
[cli-mcp-and-guidance.md](cli-mcp-and-guidance.md).

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
flutter run -d chrome --web-port=5173 --dart-define=API_BASE=http://127.0.0.1:8080/api/v1
```

Or `./scripts/dev.sh -C client flutter run -d chrome --web-port=5173 --dart-define=API_BASE=http://127.0.0.1:8080/api/v1`.
For browser dev against the local server, `scripts/dev.sh` exports
`INSIDEOUT_CORS_ORIGINS=localhost` when unset, so Flutter web on any
localhost port passes CORS without extra config.
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
flutter pub get --enforce-lockfile   # fails on a non-3.44 SDK instead of rewriting pubspec.lock
flutter analyze --no-fatal-infos
flutter test
```

The app's nginx proxy (Python 3 and any nginx binary; nothing is installed):

```bash
python3 scripts/check_nginx_reresolve.py [--nginx /path/to/nginx]   # follows the API upstream when DNS changes
```

It renders `client/nginx.conf` as the image entrypoint does and runs it beside a
stub DNS server and two upstreams, so it takes about 30 seconds. Re-run it when
the nginx base image is bumped
([BUG-027](../issues/2026-10-10-bug-027-nginx-resolved-the-api-upstream-once.md)).
On a Mac whose CommandLineTools SDK the linker rejects, a source build of nginx
needs `SDKROOT` pointed at the Xcode SDK; the Homebrew `nginx` formula avoids it.

Credential + env script suites (Python; the same two batteries CI runs
in its env-suites job — from `scripts/`):

```bash
python3 test_env_catalog.py && python3 test_env_writes.py   # env catalog + write batteries (plain-assert scripts)
python3 -m unittest discover -s tests/keys -t .           # key store suites
```

The key suites mirror the key store itself
([`scripts/tests/keys/`](../../scripts/tests/keys/README.md): one file
per credential, filename = key name) and cover two layers: per-key
**shape and placement** (mode 0600, non-empty unless operator-filled,
the right env var mapped, kind-specific shape such as PEM framing or
DSN role) and store-wide **invariants** (perms, git-blindness, host
naming), plus the `keys.py` tool behaviors including multi-host
scoping. They never print values and skip cleanly where a host's
`.secrets/` is absent (CI). Run them after any key lands, copies
between hosts, or rotates; the per-key **live probes** (part 4 of each
doc under [keys/](keys/README.md)) remain the second layer — the suite
says the key is well-formed and well-placed, the probe says it works.
