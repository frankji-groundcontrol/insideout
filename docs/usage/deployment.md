# Deployment

Operator guide for running InsideOut with the repo's docker-compose stack.
For developer setup, see [local-development.md](local-development.md).

## Topology

`docker-compose.yml` defines three services — `postgres`, `server`, and
`client` (Flutter web + nginx). Local `flutter run` and the hosted Railway
`app` remain alternatives to the compose `client`:

```
browser ──> client (Flutter web + nginx, :8080) ── /api/ ──> server (Go API, :8080)
                                                               │  DATABASE_URL
                                                               ▼
                                                             postgres (optional)
```

| Service | Image / build | Container port | Host port |
|---------|---------------|----------------|-----------|
| `postgres` | `postgres:18` | 5432 | `127.0.0.1:${POSTGRES_PORT:-5442}` (localhost only) |
| `server` | built from `server/` (golang:1.25 → distroless/static, runs as nonroot) | 8080 | `${SERVER_PORT:-8080}` |
| `client` | built from `client/` (Flutter build → nginx-unprivileged) | 8080 | `${CLIENT_PORT:-5173}` |

The bundled `postgres` service is the self-hosted default. When
`DATABASE_URL` points at an external instance instead, simply don't start it
(`docker compose up -d server client`). Both app containers run
unprivileged and self-report health (`/insideout health`; nginx `/healthz`).
Memory is capped with `mem_limit` at **512m** for `server` and **128m** for
`client` (`postgres` is uncapped). These are starting values that have not been
measured: after a local `docker compose up -d`, compare `docker stats` against
them and adjust.

## Environment (compose reads `.env`)

The full variable reference is [environment.md](environment.md); the tables
below cover only what the compose topology needs.

Required — compose fails fast (`:?`) without them:

| Variable | Used by | Notes |
|----------|---------|-------|
| `DATABASE_URL` | server | No derived default (compose can't nest `${...}` in `:-`). For the bundled DB: `postgres://insideout_app:<POSTGRES_APP_PASSWORD>@postgres:5432/insideout?sslmode=disable` |
| `INSIDEOUT_JWT_SECRET` | server | Min 32 chars; server refuses to start otherwise |

Optional, with defaults:

| Variable | Default | Used by |
|----------|---------|---------|
| `POSTGRES_APP_PASSWORD` | `insideout_app_dev_password` (dev-only) | postgres init — password for the `insideout_app` runtime role; override in `.env` for any real deployment, and it must match the password in `DATABASE_URL` when the bundled DB is used |
| `POSTGRES_OWNER_PASSWORD` | `insideout_owner_dev_password` (dev-only) | postgres init — password for `insideout_owner`; must match `DATABASE_OWNER_URL` when set |
| `POSTGRES_SUPERUSER_PASSWORD` | `insideout_dev_password` | postgres bootstrap superuser — **change it** |
| `POSTGRES_PORT` | `5442` | host port for postgres |
| `SERVER_PORT` | `8080` | host port for the Go API |
| `CLIENT_PORT` | `5173` | host port for the client (Flutter web + nginx); matches the default `INSIDEOUT_PUBLIC_URL` |
| `INSIDEOUT_LLM_BASE_URL` / `INSIDEOUT_LLM_API_KEY` | empty key | AI provider; empty key = offline template-reply coach. Include `/v1` on the base URL. |
| `INSIDEOUT_LLM_MODEL` | `claude-sonnet-4-20250514` | model id |
| `INSIDEOUT_LLM_SCHEMA` | `messages` | `messages` or `responses` |

## Bundled database bootstrap

On first boot against a fresh data volume, the official postgres image runs
`docker/postgres-init/001-create-app-role.sh`, which creates a
**non-superuser** `insideout_app` role (password `INSIDEOUT_APP_PASSWORD`,
wired from `POSTGRES_APP_PASSWORD`) and makes it OWNER of the `insideout`
database. The app never connects as the superuser — running migrations as a
superuser would bypass the `REVOKE CREATE ON SCHEMA public` lockdown that
migration #1 relies on. The init script only runs on an empty
`insideout_pgdata` volume; changing passwords later means `ALTER ROLE` by
hand or wiping the volume.

Migrations are embedded in the server binary and applied automatically on
startup; no separate migration step is needed in the compose flow.

Concurrent boots serialize: `Migrate` first takes a transaction-scoped Postgres
advisory lock, so a second replica (or a manual `insideout migrate` during a
deploy) waits for the first to finish and then finds nothing left to apply,
instead of racing on the same DDL and exiting. The lock holder keeps one
connection while the migrations run on another, so the migrate connection needs
`pool_max_conns` of at least 2 (the default); a lower value fails at boot with a
message.

## Build and run

```bash
cp .env.example .env    # then set the required vars above
docker compose build
docker compose up -d
```

The server waits for postgres's healthcheck (`pg_isready`) before starting.

## Reverse proxy / same-origin expectation

Hosted Flutter web keeps `/api/v1` same-origin through nginx on the
Railway `app` service. Local Flutter talks to the Go URL with Bearer
tokens (`INSIDEOUT_CORS_ORIGINS=localhost` matches loopback). Do not
publish the server port to the public internet if the nginx proxy is
the public front door.

Client-IP trust chain (REL-13): on Railway the request passes
browser → Railway edge → nginx → server. The edge appends the real
client IP to `X-Forwarded-For` (client-supplied spoofed values stay to
its left), and nginx passes that chain through untouched instead of
appending its own hop — so the server's rightmost XFF entry is the
edge-appended client IP and the per-IP auth throttle stays per-client.
Local compose has no edge in front of nginx, so a locally-spoofed XFF
is trusted there; that is dev-only and acceptable.

Two cookie-related knobs on the server:

- `INSIDEOUT_COOKIE_SECURE` defaults to on (Secure cookies). Only set it to
  `0` for plain-http testing — behind a TLS-terminating proxy, leave it on.
- `INSIDEOUT_DEV_CORS=1` enables permissive CORS for development only; never
  set it in production.

## Run exactly one server replica

The Go `server` keeps some state in its own process memory, so it is built
for one replica. Two replicas behind one URL quietly loosen what that state
enforces — the second one now says so at startup, but neither stops serving
(see the singleton guard below):

- **Auth throttle** ([`api/auth_throttle.go`](../../server/internal/api/auth_throttle.go)):
  the failed-login lockout and the send and volume caps are counted per
  replica, so every limit multiplies by the replica count.
- **Email resend cooldown** ([`api/auth_cooldown.go`](../../server/internal/api/auth_cooldown.go)):
  the 60-second window restarts on each replica, so a retry that lands on the
  other replica goes straight through.
- **Conversation turn lock and provider permits**
  ([`agent/dispatch.go`](../../server/internal/agent/dispatch.go)): one turn per
  conversation is enforced per replica, so two replicas can run two turns on
  the same conversation, and the cap on concurrent provider calls multiplies.
- **Presence and roadmap progress**
  ([`presence/presence.go`](../../server/internal/presence/presence.go),
  [`progresshub/hub.go`](../../server/internal/progresshub/hub.go)): a viewer
  only sees collaborators and live progress published on the replica their
  connection reached.

Already safe across replicas, because they live in Postgres: the per-user AI
rate limit (`ai_check_rate_limit`, serialised with `pg_advisory_xact_lock`)
and the AI circuit breaker (the `ai_circuit_breaker` row).

So leave the `server` service at one replica: do not raise the replica count
on Railway or run `docker compose up --scale server=N`. The `app` (nginx)
service holds no such state and can scale. The way up, from the code notes,
is `pg_advisory_lock` on the conversation id, a database-backed permit
counter, and a shared store for the throttle, cooldown and presence state.

A runtime guard makes a stray second replica loud instead of silent
(`api/single_replica.go`, review-3 SEC-03/REL-05): at startup the server
logs its single-replica contract once, then holds the
`insideout:singleton` Postgres advisory lock on a hijacked connection for
the life of the process, and any process that finds the lock already held
logs `singleton guard: another api server process already holds the
singleton lock …` at ERROR on its first moments of life. The guard is
advisory — it never blocks boot, serving, or `/healthz` — so alert on that
ERROR line (or grep the logs when scaling) rather than waiting for a health
signal, and note the held connection sits outside the request pool: each
replica opens one extra database connection on top of the budget above.
Redis- or table-backed registries remain out of scope; the counters move to
Postgres only if a second replica ever becomes a real requirement.

## Railway (current public deploy)

The first hosted instance lives on Railway in project `insideout`
(workspace **Frank Ji's Projects**), region `asia-southeast1-eqsg3a`.
Public URL (app only — use your own deployment's URL):

`https://<your-app>.up.railway.app`

```
browser ── HTTPS ──> app (Flutter web + nginx, :8080 public)
                       │  /api/ → server.railway.internal:8080
                       ▼
                     server (Go, :8080 public API + private)
                       │  DATABASE_URL (insideout_app, session pooler :5432)
                       ▼
                     shared Postgres (Supabase; insideout schema only)
```

| Service | Role | Public? |
|---------|------|---------|
| `server` | Dockerfile in `server/`, root directory `/server`; public domain for Flutter native | yes (API; browsers should still use the app host) |
| `app` | Dockerfile in `client/`, root directory `/client`; nginx static Flutter + `/api/` proxy | yes |

The dedicated Railway Postgres plugin was removed 2026-08-18. This is
provisioning model 2 from
[architecture/database-and-rls.md](../architecture/database-and-rls.md):
`insideout_app` is scoped to the `insideout` schema on a shared instance.

### Required service variables

On `server`:

- `DATABASE_URL` — session-mode pooler DSN as `insideout_app.<project-ref>`
  on port **5432** (`sslmode=require`, no `pgbouncer=true`). Set with
  `railway variable set DATABASE_URL --stdin --service server`. Do not
  use the dashboard `postgres.` user. Pool budget: each replica opens at most
  **10** connections for requests, plus **3** for the migrate step when
  `DATABASE_OWNER_URL` is set (13 in all), so replicas x 13 must fit the
  pooler's per-role connection limit. Override either number with pgx's own
  `pool_max_conns` query parameter in the DSN concerned (no environment
  variable; keep the migrate one at 2 or more). Connection lifetime (1h), idle
  time (30m) and health-check period (1m) stay at pgx's defaults.
- `INSIDEOUT_JWT_SECRET` — at least 32 characters; generate with
  `openssl rand -base64 48` and pipe into `railway variable set
  INSIDEOUT_JWT_SECRET --stdin --service server` (never print the value)
- `INSIDEOUT_ADDR=:8080` so the process port matches private DNS
- `INSIDEOUT_LLM_BASE_URL`, `INSIDEOUT_LLM_API_KEY`,
  `INSIDEOUT_LLM_MODEL`, `INSIDEOUT_LLM_SCHEMA` — also via `--stdin`

On `app`:

- `PORT=8080` so nginx matches the Railway domain target port
- The Flutter image bakes `API_BASE=/api/v1`. nginx
  [`client/nginx.conf`](../../client/nginx.conf) is rendered from a
  template at container start and proxies `/api/` to
  `$NGINX_API_UPSTREAM` (Dockerfile-baked default
  `http://server.railway.internal:8080`, buffering off; compose overrides
  it to `http://server:8080`). The value is `scheme://host:port` with no
  path. nginx looks it up on every request through the container's own
  nameservers (answers cached 10 s), so a redeployed `server` is picked up
  within seconds and `app` does not need a restart
  ([BUG-027](../issues/2026-10-10-bug-027-nginx-resolved-the-api-upstream-once.md);
  an `app` image built before that fix resolves only at start and does).
- The same nginx sends the app's security headers on every response, error
  pages included: `nosniff`, `Referrer-Policy`, `X-Frame-Options: DENY`, a
  Content-Security-Policy, and HSTS when the edge marks the request https
  (`X-Forwarded-Proto`; nginx itself only sees http behind the edge). The
  CSP allows exactly what the Flutter web build loads from elsewhere:
  CanvasKit's script and wasm from the gstatic CDN, the engine's fallback
  fonts, and the Google Fonts stylesheet `index.html` links. When a Flutter
  upgrade or a new `index.html` link makes the browser console report a CSP
  violation, loosen only that directive in
  [`client/nginx.conf`](../../client/nginx.conf) and run
  `python3 scripts/check_nginx_headers.py`
  ([local-development.md](local-development.md#testing)). The headers reach
  the hosted app with the next `railway up --service app`.

LLM vars are set on `server`. Empty `INSIDEOUT_LLM_API_KEY` still selects
the offline template coach.

### Redeploy

The directory is already linked (`railway status`). Upload the repo root
so each service's `rootDirectory` can find its Dockerfile:

```bash
railway up --service server --detach -m "describe the server change"
railway up --service app --detach -m "describe the app change"
```

Do not `railway up ./server --path-as-root`: that keeps the Docker
context at the repo root while `Dockerfile` expects `./cmd/insideout`.

### Health

- `server`: `GET /healthz` (Railway healthcheck)
- `app`: `GET /healthz` (nginx 200 `ok`)
- Same-origin API: `GET /api/v1/me` through the public app URL returns
  401 when logged out, 200 with a Bearer token (or leftover session
  cookie) when logged in
