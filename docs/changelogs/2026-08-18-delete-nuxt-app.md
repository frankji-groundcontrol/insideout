# 2026-08-18 — Delete leftover Nuxt `app/`

The Nuxt 4 tree is gone. Flutter `client/` is the only frontend.

## What changed

- Deleted `app/` (Nuxt 4 SSR, pnpm, Ink & Seal).
- `docker-compose.yml` is postgres + server only. Flutter is
  `flutter run` locally or Railway nginx, not a compose service.
- Env catalog: no component `.env.example`. `COMPONENTS` is
  `server client`, both dotenv-less. Dropped `NUXT_API_INTERNAL_BASE`
  and `APP_PORT` from `.env.example`.
- `./scripts/dev.sh -C client …` is the frontend launch path.
- Live docs and agent routers no longer describe a Nuxt tree.
  Historical changelogs / issues / learning still cite `app/` as of
  the date they were written.

## Verification

- `python3 scripts/test_env_catalog.py`
- `python3 scripts/test_env_writes.py`
- `./scripts/env.sh check server` (does not require a missing `app/`)
- `test -d app` is false

## Operator notes

If a leftover `app/.env` exists on disk, delete it; nothing reads it.
Railway `app` was already Flutter and does not change with this delete.
