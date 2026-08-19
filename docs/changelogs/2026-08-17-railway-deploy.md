# 2026-08-17 — First Railway deploy + build-time API proxy

## What changed

InsideOut now has a public Railway instance so the product can be walked
through without a local stack. The hosted topology is the compose graph
on Railway private DNS: dedicated Postgres, a private Go `server`, and a
public Nuxt `app` at `https://app-production-591e.up.railway.app`.

Two small code/config changes were required to make that topology boot:

- `server/internal/config.Load` now listens on `PORT` when
  `INSIDEOUT_ADDR` is unset, so a platform-assigned port works. The
  Railway service still pins `INSIDEOUT_ADDR=:8080` so the app's private
  URL stays stable.
- `app/Dockerfile` takes `NUXT_API_INTERNAL_BASE` as a build-arg and
  `docker-compose.yml` passes `http://server:8080/api/v1`. Nitro
  `routeRules` bake the proxy destination at image build; a runtime-only
  value left the first Railway image proxying to `127.0.0.1` (HTTP 502
  on every `/api/v1/**` call).

Operator steps, variables, and redeploy commands:
[`docs/usage/deployment.md`](../usage/deployment.md#railway-current-public-deploy).

## Verification

- `go test ./internal/config/` — listen-address fallbacks.
- Railway `server` deploy SUCCESS; runtime log `listening addr=":8080"`
  and `GET /healthz` 200.
- Railway `app` deploy SUCCESS after the Dockerfile change; healthcheck
  `GET /` 200.
- Public checks: `/`, `/login`, `/register` 200; `GET /api/v1/me` 401
  logged out; `POST /api/v1/auth/register` then `GET /api/v1/me` 200
  with the session cookie.

The coach is the offline template reply (`ANTHROPIC_AUTH_TOKEN` unset).
Set that token on `server` via `--stdin` when a live coach is wanted.

## Operator notes

Existing compose deploys should rebuild the `app` image so the new
build-arg is baked (`NUXT_API_INTERNAL_BASE=http://server:8080/api/v1`).
A container restart without rebuild keeps the old localhost proxy.
If an existing Railway (or other) image was built before this change,
redeploy `app` after setting the literal private API URL.
