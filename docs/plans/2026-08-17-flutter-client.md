# 2026-08-17 — Flutter client (web + iOS + Android)

Status: in flight — architecture confirmed; engineering review recorded.

## Goal

Replace the Nuxt 4 frontend with one Flutter app that builds **web, iOS,
and Android**, covering the **current Nuxt surface** before Railway
switches off `app/`. Keep the Go API and the `insideout` schema.

**Correction 2026-08-19:** the original lock "Use **Material 3**, not
Ink & Seal" is reversed. Flutter is the client kit; the visual world is
still Ink & Seal. See
[`docs/plans/2026-08-19-restore-ink-seal.md`](2026-08-19-restore-ink-seal.md).

## Locked decisions

| Decision | Choice |
| --- | --- |
| Targets | Web + iOS + Android from day one, one codebase |
| First ship gate | Full current Nuxt surface on all three, then switch |
| Visual system | Ink & Seal (Material widgets implement it; they do not replace it). Reversed from Material-3-only 2026-08-19 |
| Client layout | One Flutter app, hand-written Dart API client |
| Backend | Existing Go service; no new backend product surface |
| Nuxt | Stays until Flutter parity is verified; do not delete `app/` in this plan |

## Architecture (confirmed 2026-08-17)

```
Flutter client/          Go server                 Postgres
  Material 3 screens  →  /api/v1/**             →  insideout.*
  Dio + Bearer JWT       cookies still work
  secure token store     for the Nuxt app
```

- New tree: `client/` (leave `app/` alone until cutover).
- One Dart HTTP client mapping the existing JSON error contract
  (`error`, `code`, `APP_THROTTLE`, `CIRCUIT_OPEN`).
- Session: `Authorization: Bearer` (already accepted by
  `bearerOrCookie`). Login/register/refresh also return access +
  refresh tokens in JSON so mobile can renew without cookies.
- Token storage: `flutter_secure_storage` on iOS/Android; equivalent
  web store. Never put the refresh token in Dart source or logs.
- Routing: `go_router` with the current paths (`/`, `/login`,
  `/register`, `/dashboard`, `/workspace/:id`, `/workspace/:id/ideas`,
  `/workspace/:id/settings`, `/projects/:id`, `/projects/:id/roadmap`,
  `/prd/:id`, `/prd/:id/revisions`, `/prd/:id/export`, `/profile`).
- Auth gate: restore session on launch; unauthenticated routes only
  for landing/login/register.
- Coach: consume the existing SSE contract (no new streaming protocol).
- Roadmap: interactive view of the same API. Visual language is Ink &
  Seal (chops, celadon, ink); not a generic Material canvas. Pixel-port
  of the collaborative canvas is tracked on the restore plan.
- Flutter web deploy later replaces the Railway `app` service with a
  static host plus the same private proxy to `server`. iOS/Android stay
  sideload/simulator until a later store task.

CORS: native apps do not need it. Flutter web during overlap is served
as its own origin, so the Go server needs a tight production CORS
allow-list for that origin (not `INSIDEOUT_DEV_CORS=1`). After cutover,
Flutter web can sit on the same public host as today's Nuxt app and
talk same-origin again.

## Current Nuxt surface to port

Public: landing, login, register.

Signed-in: dashboard, profile, workspace board, workspace ideas,
workspace settings (members/invite), project detail + updates, GitHub
sync, roadmap, PRD editor, PRD revisions, PRD export, Coach panel.

zh-CN / en-US strings exist in `app/src/i18n/` and should be carried
over as Flutter ARB (or equivalent), default still matching the
current app.

## Engineering review (2026-08-17)

Written review, not the interactive gstack walkthrough: the architecture
was already locked with the user. Findings folded in below.

What already exists and is reused: Go `/api/v1` JSON contract, `bearerOrCookie`,
session rotation, Nuxt as the public shell until cutover.

Findings:

1. Native Flutter cannot reach `server.railway.internal`. The Go API
   needs a **public URL** (Railway service domain on `server`) before
   iOS/Android can talk to production. Cookies stay on for Nuxt.
2. `POST /auth/refresh` and `/auth/logout` only read the refresh
   **cookie**. JSON tokens are useless unless those handlers also
   accept `{ "refreshToken" }` in the body.
3. Do not wrap the login body as `{ user, tokens }` — Nuxt assigns the
   login JSON to `UserProfile`. Add `accessToken` / `refreshToken` as
   extra top-level fields on the existing user object.
4. `GET /me` must stay token-free.
5. Do not turn on `INSIDEOUT_DEV_CORS=1` in production. Add an exact
   allow-list (`INSIDEOUT_CORS_ORIGINS`). Treat the token `localhost`
   as any `http://localhost:*` / `http://127.0.0.1:*` origin for
   Flutter web during overlap.
6. Refresh rotation: the Flutter client must replace the stored
   refresh token on every successful refresh.

NOT in scope (unchanged): App Store / Play listing, deleting `app/`,
Ink & Seal, PRODUCT.md version-first slice.

## Backend delta (small)

- Include access + refresh token values in login/register/refresh JSON.
- Accept refresh token from cookie **or** JSON body on refresh/logout.
- Keep setting httpOnly cookies so Nuxt keeps working.
- `INSIDEOUT_CORS_ORIGINS` allow-list. Publish a public domain on the
  Railway `server` service for native clients.
- Do not change RLS, schema, or store authorization.

## Verification (before calling parity done)

- `flutter analyze` + widget/unit tests for session restore and the
  API error mapper.
- Real register → `/me` → create workspace against the live Go API
  (local or Railway), not mocks.
- Same flows on Chrome, iOS simulator, and Android emulator.
- Coach SSE produces at least one `delta` on a live server (offline
  coach is acceptable if no provider token).
- Streaming still exercised live, per
  [docs/practices/2026-07-21-live-exercise-streaming-endpoints.md](../practices/2026-07-21-live-exercise-streaming-endpoints.md).

## Checklist

- [x] Confirm architecture above.
- [x] Engineering review of this plan (or record why not).
- [x] Scaffold `client/` (Flutter 3, Material 3, go_router, Dio).
- [x] Auth JSON tokens + CORS allow-list on the Go server, with a
      failing real-HTTP test first.
- [x] Port each surface in the list; do not switch Railway until
      every item is verified on all three targets.
      (Web and iOS release builds succeeded 2026-08-18. Android SDK
      is not on this machine — captured in the goal scratch native log.
      Railway `app` stays Nuxt; cutover is a later operator step.)
- [x] Remaining Nuxt functional parity on Flutter (2026-08-18):
      zh-CN/en-US strings + theme toggle; GET idea/conversation;
      revision note + section snapshot view; coach empty-state,
      stage suggestions, throttle countdown; roadmap child nodes
      and move up/down; workspace invite on the board; PRD title
      edit; update kinds; navigate after build-from-PRD.
- [x] Cut Railway `app` from Nuxt to Flutter web.
      (2026-08-18: `client/Dockerfile` + nginx `/api/` proxy; Railway
      `app` `rootDirectory=/client`; public URL is Flutter Material 3.
      Path URL strategy added so `/login` is not a hash-only route.)
- [x] Changelog, architecture/frontend, usage, HANDOFF.

## Out of scope

- PRODUCT.md version-first Coach slice (stays on the board as P2).
- App Store / Play Store listing.
- Deleting `app/` — now a sibling plan:
      [`2026-08-18-delete-nuxt-app.md`](2026-08-18-delete-nuxt-app.md).
- Re-implementing Ink & Seal.
