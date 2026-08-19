# 2026-08-18 — Railway `app` serves Flutter web, not Nuxt

Production public frontend is now the Flutter Material 3 client. The Nuxt
tree in `app/` is still in the repo (local compose / leftover source) but
is no longer what Railway serves.

## What changed

- [`client/Dockerfile`](../../client/Dockerfile) — multi-stage
  `ghcr.io/cirruslabs/flutter:3.44.0` `flutter build web --release
  --dart-define=API_BASE=/api/v1`, then `nginx:1.27-alpine`.
- [`client/nginx.conf`](../../client/nginx.conf) — listen 8080 (IPv4 +
  IPv6), `/healthz` 200, `/api/` private proxy to
  `http://server.railway.internal:8080/api/` with buffering off and a
  3600s read timeout (Coach SSE), SPA `try_files` to `index.html`.
- Railway `app` `source.rootDirectory` is `/client` (Dockerfile builder);
  healthcheck `/healthz`; `PORT=8080`.
- [`client/lib/main.dart`](../../client/lib/main.dart) calls
  `usePathUrlStrategy()` on web so `/login` and `/dashboard` are real
  paths, not hash routes. The first Flutter image used hash URLs, so a
  refresh of `/login` painted the landing page.

`app/` was **not** deleted.

## Verification

- Railway deploy `34932e26-d975-4a6e-928d-dc8b745ae9a4` (path strategy)
  follows a healthy `a8d4f5cc-b647-4857-8b8c-cae7f24d3658` Flutter image
  (`/healthz` 200).
- `GET https://app-production-591e.up.railway.app/` is Flutter
  (`flutter_bootstrap.js`), not Nuxt.
- `GET /api/v1/me` through the public app host returns 401
  `authentication required` (nginx → Go).
- `POST /api/v1/auth/login` through that same host returns tokens for
  `test084614`.
- Browser login at `/#/login` (first Flutter image, hash routes)
  reached the Material dashboard and showed workspace
  `Retest WS 20260818084614`.
- After the path-strategy deploy, unsigned `/login` shows the Material
  login form (URL stays `/login`, not `/#/login`). Signed-in `/login`
  redirects to `/dashboard`.
- 2026-08-18 live walk as `test084614` on the public host: dashboard,
  workspace (invite + project list), ideas, settings, project, empty
  roadmap, PRD editor + coach panel, profile, revisions. Coach SSE
  through nginx (`curl -N` `POST /api/v1/conversations/{id}/messages`)
  produced incremental `message_start` / 28 `delta` / `message_end`
  (200 `text/event-stream`). Invite copy said “6-digit” while codes
  are 32-char hex — strings corrected in
  [`client/lib/l10n/`](../../client/lib/l10n/).

## Operator notes

Redeploy `app` from the repo root after the `/client` root directory is
set (`railway up --service app`). Do not `railway up ./client
--path-as-root` while `rootDirectory` is `/client`. Pin the Flutter
image; GHCR `:stable` lagged local Dart 3.12.2.

`NUXT_API_INTERNAL_BASE` on `app` is unused after this cutover.
