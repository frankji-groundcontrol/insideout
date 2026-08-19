# 2026-08-18 — Delete leftover Nuxt `app/`

Status: finished.

## Goal

Remove the unused Nuxt 4 tree from the repository now that Railway
`app` serves Flutter web. Keep the Go API, Flutter `client/`, and
historical records that cite `app/` as of the date they were written.

## Locked decisions

| Decision | Choice |
| --- | --- |
| Delete | The entire `app/` directory |
| Local compose | postgres + server only; no Nuxt image |
| Local frontend | `client/` via `flutter run` (or Railway) |
| Env catalog | Drop the `app` dotenv component and `NUXT_*` |
| Historical docs | Leave changelog/issue/learning `app/` citations in place |

## Engineering review

Written, not the interactive gstack walkthrough. The user asked to
delete `app/` after the Railway cutover; this is the remaining Flutter
plan item.

1. Do not put Flutter nginx in compose. `client/nginx.conf` proxies to
   `server.railway.internal`, which does not exist on the compose
   network. Local web stays `flutter run`.
2. `propagate` becomes a no-op when no component owns a `.env.example`.
   Keep the verb so existing docs and TUI save-path do not grow a
   special case.
3. Register `client` as a dotenv-less component (same as `server`) so
   `./scripts/dev.sh -C client flutter run` is a valid launch.
4. Do not rewrite historical changelogs. Current-state surfaces
   (architecture, usage, SETENV, routers, PLAN.md) must stop pointing
   at a live `app/` tree.

## Checklist

- [x] Open this plan and update the board.
- [x] Drop Nuxt from env catalog, compose, `.env.example`, scripts.
- [x] Update env unit tests (no `app/.env.example` contract).
- [x] Delete `app/`.
- [x] Update live docs + agent routers + Flutter plan + changelog.
- [x] `python3 scripts/test_env_catalog.py` (47) and
      `python3 scripts/test_env_writes.py` (37) green.

## Out of scope

- Android SDK / store listing.
- PRODUCT.md version-first slice.
- Rewriting historical Ink & Seal / Nuxt changelogs.
