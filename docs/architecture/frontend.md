# Frontend

Railway production serves the **Flutter 3 Material 3** client under
`client/` (see
[`docs/plans/2026-08-17-flutter-client.md`](../plans/2026-08-17-flutter-client.md)).
The Nuxt 4 tree was deleted 2026-08-18
([changelog](../changelogs/2026-08-18-delete-nuxt-app.md)).

## Production: Flutter (`client/`)

Hand-written Dart HTTP client (Dio), Bearer access tokens,
`flutter_secure_storage`, `go_router`, Provider, zh-CN / en-US strings
(default zh-CN). Targets: web, iOS, Android.

### Transport

Hosted web is built with `--dart-define=API_BASE=/api/v1`. nginx on the
Railway `app` service serves the static Flutter bundle and proxies
`/api/` to `http://server.railway.internal:8080/api/` (buffering off for
Coach SSE). Native and `flutter run` use the public Go URL
(`https://server-production-9c338.up.railway.app/api/v1`). Login and
register still set httpOnly cookies (unused by Flutter web) and also
return top-level `accessToken` / `refreshToken`.

Web uses `usePathUrlStrategy()` so product paths (`/login`,
`/dashboard`, `/workspace/:id`, …) are real URLs; nginx `try_files`
falls back to `index.html`.

### Layout

```
client/lib/
  main.dart           hydrate session + appearance; path URL strategy
  app.dart            MaterialApp.router + signed-in AppScaffold
  router.dart         current product paths
  session/            Session, Appearance, auth redirect
  api/                Dio client, models, request builders, errors
  features/           landing, auth, dashboard, workspace, project,
                      roadmap, prd, profile
  l10n/               zh-CN + en-US
```

### Coach chat

The PRD page consumes the SSE contract in
[the agent doc](prd-coach-agent.md): deltas, stage changes, throttle
countdown (`APP_THROTTLE` / `CIRCUIT_OPEN`). History is
`GET /api/v1/conversations/{id}/messages`.
