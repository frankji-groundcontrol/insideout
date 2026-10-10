# Deployment

## docker-compose topology

`docker-compose.yml` at the repo root defines three services:

- **postgres** — `postgres:18`, the *dedicated-instance* database option.
  Bootstraps as the image's `postgres` superuser, then
  `docker/postgres-init/001-create-app-role.sh` (run once by the image's
  initdb hook) creates the non-superuser `insideout_app` role and makes it
  the owner of the `insideout` database. The app never connects as the
  superuser.
- **server** — multi-stage build (`server/Dockerfile`):
  `golang:1.25-alpine` builder → `gcr.io/distroless/static-debian12` runtime
  (~18 MB image), runs as `nonroot` with a `HEALTHCHECK` driven by the
  binary's own `health` subcommand (distroless ships no shell). Requires
  `DATABASE_URL` and `INSIDEOUT_JWT_SECRET` explicitly (Compose cannot
  derive `DATABASE_URL` from the postgres vars — nested `${...}` defaults
  are unsupported, see
  [BUG-004](../issues/2026-07-20-bug-004-compose-nested-interpolation.md)).
- **client** — Flutter web build (`client/Dockerfile`:
  `ghcr.io/cirruslabs/flutter:3.44.0` → `nginxinc/nginx-unprivileged:1.27-alpine`)
  with [`client/nginx.conf`](../../client/nginx.conf) rendered from a
  template at container start, proxying `/api/` to `$NGINX_API_UPSTREAM`
  (`http://server:8080` in compose), resolved per request through the
  container's nameservers
  ([BUG-027](../issues/2026-10-10-bug-027-nginx-resolved-the-api-upstream-once.md)).
  The same Dockerfile is Railway's
  production `app` service (upstream default baked as
  `http://server.railway.internal:8080`).

Ports (host side, all overridable): postgres `POSTGRES_PORT` (default 5442),
server `SERVER_PORT` (default 8080), client `CLIENT_PORT` (default 5173,
matching the default `INSIDEOUT_PUBLIC_URL`).

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

Migrations can be run explicitly via `go run ./cmd/insideout migrate` (or the
container binary with the `migrate` argument), and the server **also applies
them on boot**: `runServe` calls `store.Migrate` before it starts listening and
exits non-zero if that fails
([`server/cmd/insideout/main.go`](../../server/cmd/insideout/main.go)). The
explicit subcommand exists so a deploy can migrate as a separate, observable
step rather than as a side effect of the first container start.

## Environment

`.env.example` at the repo root documents every variable bilingually with
both provisioning setups. The authoritative parser is
`server/internal/config/config.go`. AI credentials (`INSIDEOUT_LLM_BASE_URL`,
`INSIDEOUT_LLM_API_KEY`, `INSIDEOUT_LLM_MODEL`, `INSIDEOUT_LLM_SCHEMA`) are
optional — without a key the coach uses the offline template reply. The base
URL should already include `/v1`; the server appends `/messages` or
`/responses`. Verify the model id the endpoint actually serves (`GET {base}/models`)
before setting `INSIDEOUT_LLM_MODEL`.

Operator walkthroughs live in [`docs/usage/deployment.md`](../usage/deployment.md).
That guide also records the current Railway topology (public Flutter
`app` + `server` pointed at the shared-instance `insideout_app` session
pooler).
