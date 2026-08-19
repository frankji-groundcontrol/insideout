# 2026-08-17 — Flutter client started (web + iOS + Android)

## What changed

Started the Nuxt → Flutter replacement locked in
[`docs/plans/2026-08-17-flutter-client.md`](../plans/2026-08-17-flutter-client.md).
Architecture confirmed; engineering review recorded in that plan.

- New `client/` Flutter 3 app (Material 3, `go_router`, Dio, secure
  storage) covering the current product routes: landing, auth, dashboard,
  profile, workspace board/ideas/settings, project + GitHub sync, roadmap
  tree, PRD editor + coach SSE, revisions, export.
- Go login/register JSON now includes `accessToken` and `refreshToken` as
  extra top-level fields (Nuxt still reads the user fields). Refresh and
  logout accept the refresh token from the cookie **or** a JSON body.
- `INSIDEOUT_CORS_ORIGINS` allow-list (token `localhost` = any localhost
  port). The Railway `server` service now has a public HTTPS domain so
  native Flutter can reach the API.

Nuxt stays on `https://app-production-591e.up.railway.app` until Flutter
parity is verified on web, iOS, and Android.

## Verification

- `go test ./internal/api/ ./internal/config/` — refresh-from-cookie,
  refresh-from-JSON, session JSON shape, CORS allow-list.
- `python3 scripts/test_env_catalog.py` — 47 passed (catalog still only
  requires the two fail-fast vars).
- `flutter analyze --no-fatal-infos` and `flutter test` (3 passed) in
  `client/`.

Railway `server` was redeployed with the token/CORS change. Flutter
against that host:

```bash
cd client && flutter run -d chrome --dart-define=API_BASE=https://server-production-9c338.up.railway.app/api/v1
```

## 2026-08-18 follow-up

Session restore, refresh rotation, auth redirects, and Coach SSE frames
are now small shipped units (`token_store.dart`, `auth_redirect.dart`,
`sse.dart`) with tests that parse the real login/refresh JSON fixtures
and the `event: delta` / `event: error` frames `sendCoach` uses.
`flutter build web` is repeatable; `flutter build ios --no-codesign`
succeeds. Android APK was not built here (no Android SDK). The export
page's second option now sends `format=print` (the Go export query),
not `html`.

Workspace/member/idea/project/update/roadmap/PRD-status/build mutations
now go through shipped `ApiCall` builders used by `ApiClient` and the
existing screens. Coach SSE also applies `prd_updated`, `stage_changed`,
and `fact_recorded`.

## 2026-08-18 remaining Nuxt surface

Flutter now carries the leftover current-surface behavior that was still
Nuxt-only:

- zh-CN default / en-US catalogs (same keys, `{placeholder}` interpolation)
  plus a Material theme toggle, both persisted.
- `GET /ideas/{id}` and `GET /conversations/{id}`.
- Revision snapshots accept an optional `note` and the history page shows
  section contents.
- Coach empty-state, stage suggestion chips, and
  `APP_THROTTLE` / `CIRCUIT_OPEN` countdown.
- Roadmap child nodes and move up/down; invite code on the workspace
  board; PRD title edit; update kinds; navigate after build-from-PRD.

Still not in this client (locked out of the plan or blocked): Ink & Seal
landing, the collaborative canvas, Railway `app` cutover, Android APK
(no SDK on this machine). `flutter test` 39 passed;
`flutter analyze --no-fatal-infos` is clean of errors.

## Operator notes

Set `INSIDEOUT_CORS_ORIGINS=localhost` (already set on Railway) for
Flutter web on a loopback origin. Do not enable `INSIDEOUT_DEV_CORS=1`
on the hosted server. Rebuild/redeploy `server` if you need the JSON
tokens locally via Docker.
