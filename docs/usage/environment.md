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
   name with `-C` (`-C server` or `-C client`). This is how a bare
   `go run`/`go test` sees your values.
2. **`scripts/env.sh propagate`** — would generate a component `.env` from
   the root if that component owned a `.env.example`. Server and client
   do not, so the verb is a documented no-op.
3. **`docker-compose.yml`** — auto-reads the root `.env` beside it purely for
   `${VAR}` interpolation into container environment. Two required vars use
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
two agree, and `server/internal/setup/catalog/parity_test.go` feeds the same
skeleton and a live `.env` through both the Go catalog and `scripts/env_catalog.py`
and fails CI if they differ. `./scripts/env.sh edit` shows every variable's live state; the
hands-on walkthrough is [SETENV.md](../SETENV.md).

**Naming convention:** ecosystem variables keep their conventional names
(`DATABASE_URL`, `GITHUB_TOKEN`, `POSTGRES_*`); app-owned variables are
prefixed `INSIDEOUT_*`. Flutter web does not read this file.

**Fail-fast validation:** `config.Load` returns an error and the server
refuses to start if `DATABASE_URL` is empty, `INSIDEOUT_JWT_SECRET` is empty
or under 32 characters, or either TTL is not a valid Go duration. Everything
else is optional and never blocks startup.

## Quickstart

Interactive setup (prompts for the keys that need a choice, generates the JWT
secret on request, never prints values):

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
(`GITHUB_TOKEN` is read ad hoc in `server/internal/github/github.go` and by
`insideout setup --check` in
[`server/cmd/insideout/cli_setup_check.go`](../../server/cmd/insideout/cli_setup_check.go)).
That check also resolves the saved login through `clientconfig.Resolve`
(`INSIDEOUT_TOKEN`, else the 0600 config, for the base it was saved for) and
asks the API as that user: a rejected login fails the check, an unreachable
server is a skip.
The real-DB integration tests also read `DATABASE_URL` directly and skip
themselves when it is unset.

| Variable | Required | Default | Meaning |
|----------|----------|---------|---------|
| `DATABASE_URL` | **yes** | — | PostgreSQL DSN for the **runtime** role `insideout_app` (DML, subject to RLS). `serve` exits 1 when this role has `BYPASSRLS` or is a superuser (see [Troubleshooting](#troubleshooting)). May carry `?pgbouncer=true` (see [Database setups](#database-setups)). |
| `DATABASE_OWNER_URL` | no | empty | DSN for `insideout_owner` (NOSUPERUSER) used only to apply migrations. Empty = migrate uses `DATABASE_URL`, which must already be that owner. Required on the bundled postgres, whose `DATABASE_URL` is the app role: `env.sh check` fails when it is empty there. |
| `INSIDEOUT_JWT_SECRET` | **yes** | — | HMAC signing secret for access/refresh JWTs. **Min 32 characters** (fail-fast). |
| `INSIDEOUT_ADDR` | no | `:8080` | HTTP listen address (host:port). |
| `INSIDEOUT_ACCESS_TTL` | no | `15m` | Access-token lifetime (Go `time.Duration`). |
| `INSIDEOUT_REFRESH_TTL` | no | `720h` | Refresh-token lifetime (Go `time.Duration`). |
| `INSIDEOUT_DEV_CORS` | no | off | Permissive-CORS toggle for dev; on only when the value is exactly `1`. |
| `INSIDEOUT_CORS_ORIGINS` | no | empty | Comma-separated Origin allow-list for browser clients that are not same-origin (Flutter web). The token `localhost` matches any `localhost` / `127.0.0.1` port. Native apps do not need CORS. |
| `INSIDEOUT_COOKIE_SECURE` | no | on | Sets the `Secure` attribute on auth cookies; secure unless the value is exactly `0` (plain-http local dev). |
| `INSIDEOUT_MAIL_API_KEY` | no | — | Resend API key for transactional email (magic-link login, password reset). **Empty = mail dev mode**: recipient and subject are logged, not sent — message bodies (magic links) appear only with `INSIDEOUT_MAIL_DEV_BODY=1`. Setting a key requires `INSIDEOUT_PUBLIC_URL` (fail-fast otherwise, so emailed links never point at the localhost default). |
| `INSIDEOUT_MAIL_DEV_BODY` | no | off | Dev mode only: exactly `1` includes full message bodies in the not-sent mail log (`scripts/dev.sh` sets it locally). Never set on a deployed host. |
| `INSIDEOUT_MAIL_FROM` | no | `InsideOut <onboarding@resend.dev>` | From address for auth email; must be a Resend-verified sender. The default is Resend's sandbox address, which only delivers to the Resend account's own email. |
| `INSIDEOUT_PUBLIC_URL` | no | `http://localhost:5173` | Browser origin of the Flutter app — the base of every emailed invite/magic/reset link. Set to the hosted domain in production. Locally, run the web dev server on this port (`flutter run --web-port=5173`) or set this to the port in use. |
| `GITHUB_TOKEN` | no | — | Optional GitHub PAT; when non-empty, added as a Bearer `Authorization` header on commit-sync requests (raises rate limits). Also read by `insideout setup --check`, which validates it against the GitHub API. |
| `INSIDEOUT_GH_APP_ID` | no | — | Numeric id of the GitHub App; with the App's private key it mints installation access tokens (server-to-server evidence sync). Authenticates nothing alone. |
| `INSIDEOUT_GH_PRIVATE_KEY` | no | — | The App's `.pem` with newlines escaped as literal `\n` on a single line. Ignored when `INSIDEOUT_GH_PRIVATE_KEY_FILE` is set. |
| `INSIDEOUT_GH_PRIVATE_KEY_FILE` | no | — | Path to the App's `.pem` on disk; wins over the escaped inline var when set — the file is read raw, no `\n` unescaping. |
| `INSIDEOUT_GH_WEBHOOK_SECRET` | no | — | HMAC-SHA256 shared secret (`X-Hub-Signature-256`) verifying GitHub App webhook deliveries; the same string is configured on the App at github.com. Empty disables `POST /api/v1/hooks/github`. |

### AI provider

Also read by the Go backend. All optional — an empty token selects the
offline coach, not an error.

| Variable | Required | Default | Meaning |
|----------|----------|---------|---------|
| `INSIDEOUT_LLM_BASE_URL` | no | `https://api.anthropic.com/v1` | Provider base URL. Include `/v1` yourself; the server only appends `/messages` or `/responses`. |
| `INSIDEOUT_LLM_API_KEY` | no | — | Provider API key. **Empty = offline template-reply coach** (see [Recipes](#recipes)). |
| `INSIDEOUT_LLM_MODEL` | no | `claude-sonnet-4-20250514` | Model id passed to the streamer. |
| `INSIDEOUT_LLM_SCHEMA` | no | `messages` | Wire format: `messages` (Anthropic Messages at `{base}/messages`) or `responses` (OpenAI Responses at `{base}/responses`). |
| `INSIDEOUT_AI_MAX_CONCURRENT` | no | `4` | Concurrency ceiling for AI turns; excess turns queue briefly, then are rejected (503 with a retry hint). |
| `INSIDEOUT_AI_QUEUE_WAIT` | no | `15s` | How long a queued AI turn waits for a slot before rejection (Go `time.Duration`). |

### docker-compose only

Read by [`docker-compose.yml`](../../docker-compose.yml) for interpolation;
no application process ever reads these. `SERVER_PORT` / `CLIENT_PORT` /
`POSTGRES_PORT` are **host-side port mappings** — inside the
containers the server and client listen on `:8080` and postgres on `5432`
regardless.

| Variable | Required | Default | Meaning |
|----------|----------|---------|---------|
| `POSTGRES_APP_PASSWORD` | no | `insideout_app_dev_password` (dev-only) | Init password for `insideout_app`. Dev-only default so the file also parses for remote-DSN setups; override in `.env` for any real deployment. Must match the password in `DATABASE_URL` when the bundled `postgres` is started. |
| `POSTGRES_OWNER_PASSWORD` | no | `insideout_owner_dev_password` (dev-only) | Init password for `insideout_owner`. Same dev-only default posture; must match `DATABASE_OWNER_URL` when that URL is set. |
| `POSTGRES_SUPERUSER_PASSWORD` | no | `insideout_dev_password` | Bootstrap superuser password for the postgres image. The app never connects with it. |
| `POSTGRES_PORT` | no | `5442` | Host side of the postgres mapping (`5432` inside). |
| `SERVER_PORT` | no | `8080` | Host side of the server mapping (`:8080` inside, hardcoded). |
| `CLIENT_PORT` | no | `5173` | Host side of the client mapping (nginx on `:8080` inside); default matches `INSIDEOUT_PUBLIC_URL`. |

Compose also interpolates the operator-facing vars that flow into the Go
backend — `DATABASE_URL`, `INSIDEOUT_JWT_SECRET`, the `INSIDEOUT_LLM_*`
provider set, `INSIDEOUT_AI_MAX_CONCURRENT` / `INSIDEOUT_AI_QUEUE_WAIT`,
the `INSIDEOUT_MAIL_*` set with `INSIDEOUT_PUBLIC_URL`, the browser knobs
(`INSIDEOUT_CORS_ORIGINS`, `INSIDEOUT_DEV_CORS`, `INSIDEOUT_COOKIE_SECURE`,
`INSIDEOUT_ACCESS_TTL`, `INSIDEOUT_REFRESH_TTL`), and the GitHub
integration (`INSIDEOUT_GH_*`, `GITHUB_TOKEN`).
The first two carry `:?` guards (see above); the LLM key defaults to empty,
while model/schema/base take the same fallbacks `config.Load` applies.
`DATABASE_URL` deliberately has no
derived default because Compose cannot nest `${...}` inside a `:-` default
([why](../issues/2026-07-20-bug-004-compose-nested-interpolation.md)).

### Frontend

Flutter does not read the root `.env`. Hosted web bakes
`--dart-define=API_BASE=/api/v1` in [`client/Dockerfile`](../../client/Dockerfile)
and nginx proxies `/api/` to the Go server. Locally:

```bash
cd client && flutter run -d chrome --web-port=5173 --dart-define=API_BASE=http://127.0.0.1:8080/api/v1
```

## Database setups

Both use the same migrations; the runtime app always connects as the
non-superuser `insideout_app` role.

**(a) Remote instance** (dev default) — any Postgres 14+ host. The role named
in the URL must own that database:

```
DATABASE_URL=postgres://insideout_app:change_me@your-remote-host:5432/insideout?sslmode=require
```

**(b) Bundled compose postgres** (self-hosted default) —
`docker compose up -d postgres` brings up `postgres:18`. On a fresh data
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

**Offline AI mode.** Leave `INSIDEOUT_LLM_API_KEY` empty. The server logs
`INSIDEOUT_LLM_API_KEY not set — using offline template-reply coach` and runs
the deterministic template streamer — the PRD coach works end-to-end with no
network calls. Recommended default for local dev.

**Plain-http local dev cookies.** Set `INSIDEOUT_COOKIE_SECURE=0` so the
browser stores auth cookies without TLS. Secure by default; only disable for
local http.

**Permissive dev CORS.** Set `INSIDEOUT_DEV_CORS=1`. Off unless the value is
exactly `1`; never enable it in production.

**Changing ports.** Under compose, set `SERVER_PORT` / `POSTGRES_PORT` to
remap the host side; container ports are fixed (postgres binds to
`127.0.0.1` on the host). Under a bare `go run`, set `INSIDEOUT_ADDR` instead.

**Production.** Get env from the platform, never a file. With this repo's
compose stack, values arrive via root `.env` `${VAR}` interpolation, and the
two `:?`-guarded vars (`DATABASE_URL`, `INSIDEOUT_JWT_SECRET`) abort startup
if empty.

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
| `config: INSIDEOUT_LLM_SCHEMA must be messages or responses` | Set `INSIDEOUT_LLM_SCHEMA` to `messages` or `responses`, or unset it to take `messages`. |
| `config: INSIDEOUT_DEV_CORS=1 reflects any Origin with credentials, so it is only allowed with a localhost INSIDEOUT_PUBLIC_URL` | Unset `INSIDEOUT_DEV_CORS` (never set it on a hosted deployment). Needs a LAN or hosted browser origin? List it in `INSIDEOUT_CORS_ORIGINS` instead. |

**`serve` exits 1: "the role in DATABASE_URL has BYPASSRLS ... would skip
every row-level security policy".** This refusal is raised after the pool
opens (before the startup migrations), not by `config.Load`, so it is not in
the table above. A role with `BYPASSRLS` or `SUPERUSER` ignores every policy,
`FORCE` included, and the check will not let the server run tenant isolation
silently off. Point `DATABASE_URL` at the `insideout_app` role and keep the
privileged role in `DATABASE_OWNER_URL`; `migrate`, `seed` and
`promote-admin` do not go through `serve` and are not affected. The message
names the attribute and the variable, never the role or the DSN.

**The browser drops auth cookies over http.** That is the `Secure` attribute
doing its job. Set `INSIDEOUT_COOKIE_SECURE=0` for plain-http local dev.

**"INSIDEOUT_LLM_API_KEY not set — using offline template-reply coach"** in the
logs is informational, not an error — you are in offline AI mode. Set
`INSIDEOUT_LLM_API_KEY` (and `INSIDEOUT_LLM_BASE_URL`) to use a real provider.

**`docker compose up` aborts before starting.** A `:?` guard fired — set
`DATABASE_URL` and `INSIDEOUT_JWT_SECRET` in `.env`. (`POSTGRES_APP_PASSWORD`
/ `POSTGRES_OWNER_PASSWORD` no longer guard startup; they fall back to
dev-only defaults — see the compose table above.)
