# Deployment

Operator guide for running InsideOut with the repo's docker-compose stack.
For developer setup, see [local-development.md](local-development.md).

## Topology

`docker-compose.yml` defines two services. Flutter is not in compose
(local `flutter run`, or Railway `app`):

```
browser ── flutter run / hosted nginx ──> server (Go API, :8080)
                                            │  DATABASE_URL
                                            ▼
                                          postgres (optional)
```

| Service | Image / build | Container port | Host port |
|---------|---------------|----------------|-----------|
| `postgres` | `postgres:17` | 5432 | `${POSTGRES_PORT:-5442}` |
| `server` | built from `server/` (golang:1.25 → distroless/static) | 8080 | `${SERVER_PORT:-8080}` |

The bundled `postgres` service is the self-hosted default. When
`DATABASE_URL` points at an external instance instead, simply don't start it
(`docker compose up -d server`).

## Environment (compose reads `.env`)

The full variable reference is [environment.md](environment.md); the tables
below cover only what the compose topology needs.

Required — compose fails fast (`:?`) without them:

| Variable | Used by | Notes |
|----------|---------|-------|
| `DATABASE_URL` | server | No derived default (compose can't nest `${...}` in `:-`). For the bundled DB: `postgres://insideout_app:<POSTGRES_APP_PASSWORD>@postgres:5432/insideout?sslmode=disable` |
| `INSIDEOUT_JWT_SECRET` | server | Min 32 chars; server refuses to start otherwise |
| `POSTGRES_APP_PASSWORD` | postgres init | Password for the `insideout_app` runtime role (only required if you start the bundled `postgres`) |

Optional, with defaults:

| Variable | Default | Used by |
|----------|---------|---------|
| `POSTGRES_SUPERUSER_PASSWORD` | `insideout_dev_password` | postgres bootstrap superuser — **change it** |
| `POSTGRES_PORT` | `5442` | host port for postgres |
| `SERVER_PORT` | `8080` | host port for the Go API |
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

Two cookie-related knobs on the server:

- `INSIDEOUT_COOKIE_SECURE` defaults to on (Secure cookies). Only set it to
  `0` for plain-http testing — behind a TLS-terminating proxy, leave it on.
- `INSIDEOUT_DEV_CORS=1` enables permissive CORS for development only; never
  set it in production.

## Railway (current public deploy)

The first hosted instance lives on Railway in project `insideout`
(workspace **Frank Ji's Projects**), region `asia-southeast1-eqsg3a`.
Public URL (app only):

`https://app-production-591e.up.railway.app`

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
  use the dashboard `postgres.` user.
- `INSIDEOUT_JWT_SECRET` — at least 32 characters; generate with
  `openssl rand -base64 48` and pipe into `railway variable set
  INSIDEOUT_JWT_SECRET --stdin --service server` (never print the value)
- `INSIDEOUT_ADDR=:8080` so the process port matches private DNS
- `INSIDEOUT_LLM_BASE_URL`, `INSIDEOUT_LLM_API_KEY`,
  `INSIDEOUT_LLM_MODEL`, `INSIDEOUT_LLM_SCHEMA` — also via `--stdin`

On `app`:

- `PORT=8080` so nginx matches the Railway domain target port
- The Flutter image bakes `API_BASE=/api/v1`. nginx
  [`client/nginx.conf`](../../client/nginx.conf) proxies `/api/` to
  `http://server.railway.internal:8080/api/` (literal hostname, buffering
  off).

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
