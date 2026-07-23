# Deployment

## docker-compose topology

`docker-compose.yml` at the repo root defines three services:

- **postgres** — `postgres:17`, the *dedicated-instance* database option.
  Bootstraps as the image's `postgres` superuser, then
  `docker/postgres-init/001-create-app-role.sh` (run once by the image's
  initdb hook) creates the non-superuser `insideout_app` role and makes it
  the owner of the `insideout` database. The app never connects as the
  superuser.
- **server** — multi-stage build (`server/Dockerfile`):
  `golang:1.25-alpine` builder → `gcr.io/distroless/static-debian12` runtime
  (~18 MB image). Requires `DATABASE_URL` and `INSIDEOUT_JWT_SECRET`
  explicitly (Compose cannot derive `DATABASE_URL` from the postgres vars —
  nested `${...}` defaults are unsupported, see
  [BUG-004](../issues/2026-07-20-bug-004-compose-nested-interpolation.md)).
- **app** — multi-stage build (`app/Dockerfile`): `node:22-alpine` + pnpm
  builder → `node:22-alpine` runtime executing `node .output/server/index.mjs`.
  The first `COPY` includes `pnpm-workspace.yaml`, which pnpm reads during
  install — omitting it breaks fresh-container builds
  ([BUG-006](../issues/2026-07-20-bug-006-pnpm-ignored-build-scripts.md)). Receives
  `NUXT_API_INTERNAL_BASE=http://server:8080/api/v1` so the Nitro proxy
  targets the server container.

Ports (host side, all overridable): postgres `POSTGRES_PORT` (default 5442),
server `SERVER_PORT` (default 8080), app `APP_PORT` (default 3000). Users
reach only the app; it proxies API traffic to the server internally.

## Database provisioning models

The same migration files support two targets without modification (see
[database and RLS](database-and-rls.md) for the ownership details):

1. **Bundled dedicated instance** — the compose `postgres` service above.
   `DATABASE_URL` points at it with `sslmode=disable`.
2. **Existing/shared instance** — any PostgreSQL 14+ the operator provisions,
   including a shared multi-tenant managed project where `insideout_app`
   owns only the `insideout` schema. When connecting through a
   transaction-mode pooler (a `pgbouncer=true` connection string), the
   server automatically switches pgx to the simple query protocol
   (`server/internal/store/pool.go`) — prepared statements don't survive
   transaction pooling. Session-mode poolers and direct connections use the
   faster default mode.

Migrations run via `go run ./cmd/insideout migrate` (or the container binary
with the `migrate` argument) — the server does not auto-migrate on boot.

## Environment

`.env.example` at the repo root documents every variable bilingually with
both provisioning setups. The authoritative parser is
`server/internal/config/config.go`. AI credentials (`ANTHROPIC_BASE_URL`,
`ANTHROPIC_AUTH_TOKEN`, `AI_MODEL`) are optional — without a token the coach
uses the offline template reply. When pointing at an Anthropic-compatible
gateway, verify the model id it actually serves (`GET /v1/models`) before
setting `AI_MODEL`.

Operator walkthroughs live in [`docs/usage/deployment.md`](../usage/deployment.md).
