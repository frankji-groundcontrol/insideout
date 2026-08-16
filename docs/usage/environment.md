# Environment Configuration

The single source of truth for every environment variable InsideOut reads —
what each one means, which process consumes it, whether it is required, and
its default. For the developer workflow around these variables (prereqs,
running, testing) see [local-development.md](local-development.md); for the
compose topology see [deployment.md](deployment.md). The committed template is
[`.env.example`](../../.env.example).

## How it works

The root `.env` is a **dead text file**. Nothing in the application reads it
directly. The Go server has no dotenv library — it reads the process
environment only via `os.Getenv`
([`server/internal/config/config.go`](../../server/internal/config/config.go)).
Three bridges make `.env` live:

1. **`scripts/dev.sh`** (repo root) — preflights
   `scripts/env.sh check <component>` (validates the required keys without
   printing values; on failure it names exactly what to fix and aborts before
   anything launches; and either way lists the defaults in effect for unset
   optionals, tagged by consumer), then exports `.env`
   (`set -a; source .env; set +a`) and execs a command inside the directory you
   name with `-C` (`-C server` or `-C app`). This is how a bare
   `go run`/`go test`/`pnpm dev` sees your values.
2. **`scripts/env.sh propagate`** — generates `app/.env` from the root file,
   scoped to what `app/.env.example` declares and stamped with the root file's
   checksum, so the Nuxt component also works when launched straight from its
   own directory. `check app` reports the copy as stale when the root moves on,
   and `dev.sh` refuses to launch `app` until it is regenerated.
3. **`docker-compose.yml`** — auto-reads the root `.env` beside it purely for
   `${VAR}` interpolation into container environment. Three required vars use
   `:?` guards, so `docker compose up` aborts with a message before starting
   if they are empty.

Under plain `go run` without `dev.sh`, `.env` is inert unless you export it
yourself.

**The skeleton is the schema.** In `.env.example` an *uncommented* assignment
means required and a *commented* one means optional, with the value shown being
the default already in effect. Only `DATABASE_URL` and `INSIDEOUT_JWT_SECRET`
are uncommented, because only those two make the server refuse to boot. The
"Required" column below restates that, and
[`scripts/test_env_catalog.py`](../../scripts/test_env_catalog.py) asserts the
two agree. `./scripts/env.sh edit` shows every variable's live state; the
hands-on walkthrough is [SETENV.md](../SETENV.md).

**Naming convention:** ecosystem variables keep their conventional names
(`DATABASE_URL`, `ANTHROPIC_*`, `GITHUB_TOKEN`, `POSTGRES_*`); app-owned
variables are prefixed `INSIDEOUT_*`; the frontend's one variable follows the
Nuxt `NUXT_*` convention.

**Fail-fast validation:** `config.Load` returns an error and the server
refuses to start if `DATABASE_URL` is empty, `INSIDEOUT_JWT_SECRET` is empty
or under 32 characters, or either TTL is not a valid Go duration. Everything
else is optional and never blocks startup.

## Quickstart

Interactive setup (prompts for the keys that need a choice, generates the JWT
secret on request, regenerates `app/.env`, never prints values):

```bash
./scripts/env.sh init
./scripts/env.sh edit    # or: pick any single variable from a live list
```

Or by hand — `cp .env.example .env`, then fill in the two required values:

- `DATABASE_URL` — a reachable Postgres connection string (see
  [Database setups](#database-setups)).
- `INSIDEOUT_JWT_SECRET` — any random string, **at least 32 characters**.

Then run the server (from the repo root), exporting `.env` via the wrapper:

```bash
./scripts/dev.sh -C server go run ./cmd/insideout
```

Serving over plain http locally? Prefix `INSIDEOUT_COOKIE_SECURE=0` or the
browser drops the auth cookies (see [Recipes](#recipes)).

## Variable reference

### Go backend

Read by [`server/internal/config/config.go`](../../server/internal/config/config.go)
(`GITHUB_TOKEN` is read ad hoc in `server/internal/github/github.go`). The
real-DB integration tests also read `DATABASE_URL` directly and skip
themselves when it is unset.

| Variable | Required | Default | Meaning |
|----------|----------|---------|---------|
| `DATABASE_URL` | **yes** | — | PostgreSQL connection string for the pgx pool. May carry a `?pgbouncer=true` query param (see [Database setups](#database-setups)). |
| `INSIDEOUT_JWT_SECRET` | **yes** | — | HMAC signing secret for access/refresh JWTs. **Min 32 characters** (fail-fast). |
| `INSIDEOUT_ADDR` | no | `:8080` | HTTP listen address (host:port). |
| `INSIDEOUT_ACCESS_TTL` | no | `15m` | Access-token lifetime (Go `time.Duration`). |
| `INSIDEOUT_REFRESH_TTL` | no | `720h` | Refresh-token lifetime (Go `time.Duration`). |
| `INSIDEOUT_DEV_CORS` | no | off | Permissive-CORS toggle for dev; on only when the value is exactly `1`. |
| `INSIDEOUT_COOKIE_SECURE` | no | on | Sets the `Secure` attribute on auth cookies; secure unless the value is exactly `0` (plain-http local dev). |
| `GITHUB_TOKEN` | no | — | Optional GitHub PAT; when non-empty, added as a Bearer `Authorization` header on commit-sync requests (raises rate limits). |

### AI provider

Also read by the Go backend. All optional — an empty token selects the
offline coach, not an error.

| Variable | Required | Default | Meaning |
|----------|----------|---------|---------|
| `ANTHROPIC_BASE_URL` | no | `https://api.anthropic.com` | Base URL of the Anthropic-Messages-API-compatible provider. |
| `ANTHROPIC_AUTH_TOKEN` | no | — | Provider auth token. **Empty = offline template-reply coach** (see [Recipes](#recipes)). |
| `AI_MODEL` | no | `claude-sonnet-4-20250514` | Model id passed to the streamer. |

### docker-compose only

Read by [`docker-compose.yml`](../../docker-compose.yml) for interpolation;
no application process (Go or Nuxt) ever reads these. `SERVER_PORT` /
`APP_PORT` / `POSTGRES_PORT` are **host-side port mappings** — inside the
containers the server listens on `:8080` and Nuxt on `:3000` regardless.

| Variable | Required | Default | Meaning |
|----------|----------|---------|---------|
| `POSTGRES_APP_PASSWORD` | **conditional** (`:?`) | — | Injected into the postgres container as `INSIDEOUT_APP_PASSWORD`; the one-shot init script uses it to `CREATE ROLE insideout_app` and make it owner of the `insideout` DB. Required only when you start the bundled `postgres` — which is why the skeleton leaves it commented, and why `env.sh check` requires it exactly when `DATABASE_URL` points there (and warns if the two passwords disagree). |
| `POSTGRES_SUPERUSER_PASSWORD` | no | `insideout_dev_password` | Bootstrap superuser password for the postgres image. The app never connects with it. |
| `POSTGRES_PORT` | no | `5442` | Host side of the postgres mapping (`5432` inside). |
| `SERVER_PORT` | no | `8080` | Host side of the server mapping (`:8080` inside, hardcoded). |
| `APP_PORT` | no | `3000` | Host side of the app mapping (`:3000` inside, hardcoded). |

Compose also interpolates five vars that flow into the Go backend —
`DATABASE_URL`, `INSIDEOUT_JWT_SECRET`, `ANTHROPIC_BASE_URL`,
`ANTHROPIC_AUTH_TOKEN`, and `AI_MODEL`. The first two carry `:?` guards (see
above); the two `ANTHROPIC_*` vars default to empty, while `AI_MODEL`
defaults to `claude-sonnet-4-20250514` (the same fallback the Go config
applies). `DATABASE_URL` deliberately has no
derived default because Compose cannot nest `${...}` inside a `:-` default
([why](../issues/2026-07-20-bug-004-compose-nested-interpolation.md)).

### Frontend

The entire Nuxt frontend reads exactly **one** environment variable. The
browser sees none of it — `runtimeConfig.public` is empty.

| Variable | Required | Default | Meaning |
|----------|----------|---------|---------|
| `NUXT_API_INTERNAL_BASE` | no | `http://127.0.0.1:8080/api/v1` | Internal base URL of the Go API, including the `/api/v1` suffix. Server-only: it populates `runtimeConfig.apiInternalBase`, which the SSR process uses to call the backend directly and which the `/api/v1/**` Nitro proxy forwards the browser's same-origin requests to. |

The default is correct for local dev, so **the frontend needs nothing in
`.env`**. The browser always calls same-origin `/api/v1`; the proxy follows
this variable. If the Go server lives elsewhere, set it to that origin plus
`/api/v1` — one value moves both consumers. Under compose it is hardcoded to
`http://server:8080/api/v1` on the `app` service.

Set it in the **root** `.env`, then run `./scripts/env.sh propagate`. Nuxt does
not auto-load the root file, so the generated `app/.env` is what makes
`cd app && pnpm dev` agree with `./scripts/dev.sh -C app pnpm dev`.

Do not hand-write `app/.env`. The two layers do not compose the way the file
suggests: an exported value **wins** over it (Nuxt's c12 assigns a dotenv key
only when `process.env` has none), so a hand-written line is live under one
launch path and inert under the other depending on whether the root also
declares the key — with no symptom either way. The generated copy carries the
root file's checksum, `env.sh check app` fails when it drifts, and `dev.sh`
refuses to launch on a stale copy. `app/.env.example` is the component's
contract and bounds what `propagate` may copy into it.

## Database setups

Both use the same migrations; the runtime app always connects as the
non-superuser `insideout_app` role.

**(a) Remote instance** (dev default) — any Postgres 14+ host. The role named
in the URL must own that database:

```
DATABASE_URL=postgres://insideout_app:change_me@your-remote-host:5432/insideout?sslmode=require
```

**(b) Bundled compose postgres** (self-hosted default) —
`docker compose up -d postgres` brings up `postgres:17`. On a fresh data
volume, [`docker/postgres-init/`](../../docker/postgres-init) creates the
non-superuser `insideout_app` role (password wired from
`POSTGRES_APP_PASSWORD`) and makes it owner of the `insideout` database.
Point `DATABASE_URL` at it using the same password:

```
DATABASE_URL=postgres://insideout_app:change_me_app@postgres:5432/insideout?sslmode=disable
```

**Transaction poolers:** if `DATABASE_URL` contains the substring
`pgbouncer=true` (PgBouncer/Supavisor transaction mode, e.g. Supabase port
6543), the server switches pgx to the simple query protocol because
server-side prepared statements don't survive transaction pooling. This is a
substring inside `DATABASE_URL`, not a separate variable — just don't strip
it. Session-mode / dedicated Postgres is unaffected.

## Recipes

**Offline AI mode.** Leave `ANTHROPIC_AUTH_TOKEN` empty. The server logs
`ANTHROPIC_AUTH_TOKEN not set — using offline template-reply coach` and runs
the deterministic template streamer — the PRD coach works end-to-end with no
network calls. Recommended default for local dev.

**Plain-http local dev cookies.** Set `INSIDEOUT_COOKIE_SECURE=0` so the
browser stores auth cookies without TLS. Secure by default; only disable for
local http.

**Permissive dev CORS.** Set `INSIDEOUT_DEV_CORS=1`. Off unless the value is
exactly `1`; never enable it in production.

**Changing ports.** Under compose, set `SERVER_PORT` / `APP_PORT` /
`POSTGRES_PORT` to remap the host side; container ports are fixed. Under a
bare `go run`, set `INSIDEOUT_ADDR` instead.

**Production.** Get env from the platform, never a file. With this repo's
compose stack, values arrive via root `.env` `${VAR}` interpolation, and the
three `:?`-guarded vars (`DATABASE_URL`, `INSIDEOUT_JWT_SECRET`,
`POSTGRES_APP_PASSWORD`) abort startup if empty.

## Security

- `.env` is gitignored and never committed; no script prints its values. Only
  [`.env.example`](../../.env.example) (all placeholders) is tracked.
- `INSIDEOUT_JWT_SECRET` must be at least 32 characters; generate a fresh
  random value and never reuse a dev secret in production.
- `DATABASE_URL` may point at a shared multi-tenant instance — migrations and
  the app stay inside the `insideout` schema and never touch other tenants'
  `public` objects ([why](../architecture/database-and-rls.md)).
- Keep `POSTGRES_SUPERUSER_PASSWORD` away from the app; it connects only as
  `insideout_app`.

## Troubleshooting

Fail-fast errors from `config.Load` (the server refuses to start):

| Error | Fix |
|-------|-----|
| `config: DATABASE_URL is required` | Set `DATABASE_URL` (and export it via `dev.sh`, or let compose interpolate it). |
| `config: INSIDEOUT_JWT_SECRET is required` | Set a non-empty `INSIDEOUT_JWT_SECRET`. |
| `config: INSIDEOUT_JWT_SECRET must be at least 32 characters` | Use a longer secret (≥32 chars). |
| `config: invalid INSIDEOUT_ACCESS_TTL "xyz": time: invalid duration "xyz"` | Use a valid Go duration (e.g. `15m`, `720h`) or unset it to take the default. Same shape for `INSIDEOUT_REFRESH_TTL`. |

**The browser drops auth cookies over http.** That is the `Secure` attribute
doing its job. Set `INSIDEOUT_COOKIE_SECURE=0` for plain-http local dev.

**"ANTHROPIC_AUTH_TOKEN not set — using offline template-reply coach"** in the
logs is informational, not an error — you are in offline AI mode. Set
`ANTHROPIC_AUTH_TOKEN` (and `ANTHROPIC_BASE_URL`) to use a real provider.

**`docker compose up` aborts before starting.** A `:?` guard fired — set
`DATABASE_URL`, `INSIDEOUT_JWT_SECRET`, and `POSTGRES_APP_PASSWORD` in `.env`.
