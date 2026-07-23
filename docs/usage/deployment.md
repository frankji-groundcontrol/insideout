# Deployment

Operator guide for running InsideOut with the repo's docker-compose stack.
For developer setup, see [local-development.md](local-development.md).

## Topology

`docker-compose.yml` defines three services:

```
browser ── :${APP_PORT:-3000} ──> app (Nuxt/Nitro SSR, node:22)
                                    │  proxies /api/v1/** same-origin
                                    ▼
                                  server (Go API, distroless, :8080)
                                    │  DATABASE_URL
                                    ▼
                                  postgres (postgres:17, optional)
```

| Service | Image / build | Container port | Host port |
|---------|---------------|----------------|-----------|
| `postgres` | `postgres:17` | 5432 | `${POSTGRES_PORT:-5442}` |
| `server` | built from `server/` (golang:1.25 → distroless/static) | 8080 | `${SERVER_PORT:-8080}` |
| `app` | built from `app/` (node:22-alpine, `pnpm build` → `.output`) | 3000 | `${APP_PORT:-3000}` |

The bundled `postgres` service is the self-hosted default. When
`DATABASE_URL` points at an external instance instead, simply don't start it
(`docker compose up -d server app`).

## Environment (compose reads `.env`)

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
| `APP_PORT` | `3000` | host port for the app |
| `ANTHROPIC_BASE_URL` / `ANTHROPIC_AUTH_TOKEN` | empty | AI provider; empty token = offline template-reply coach |
| `AI_MODEL` | `claude-sonnet-4-20250514` | model id |

Compose hard-codes `NUXT_API_INTERNAL_BASE=http://server:8080/api/v1` on the
`app` service so Nitro reaches the server over the compose network.

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

Auth uses httpOnly cookies, and the app keeps them first-party by proxying
`/api/v1/**` through Nitro to the server container — the browser never talks
to the Go API directly, so there is no CORS to configure. Your public reverse
proxy (nginx, Caddy, ...) should therefore expose **only the `app` port** and
forward everything to it; do not publish the server port to the internet.

Two cookie-related knobs on the server:

- `INSIDEOUT_COOKIE_SECURE` defaults to on (Secure cookies). Only set it to
  `0` for plain-http testing — behind a TLS-terminating proxy, leave it on.
- `INSIDEOUT_DEV_CORS=1` enables permissive CORS for development only; never
  set it in production.
