# 2026-07-27 — Environment hygiene pass

The single root `.env` stays — flat, compose-native, 12-factor (prod gets env
from the platform, never a file). What was ugly was hygiene, not structure:
a fossil example file, an example template grouped by the wrong consumer, and
a shell incantation copy-pasted across docs. No behavior change.

## What changed

- **Deleted `app/.env.example`.** A Supabase-era fossil documenting
  `NUXT_PUBLIC_API_MODE` / `NUXT_PUBLIC_BUNDLE_*` / `NUXT_PUBLIC_SUPABASE_*`
  vars that do nothing — `nuxt.config.ts` has `runtimeConfig.public: {}`. The
  frontend's one real variable (`NUXT_API_INTERNAL_BASE`, server-only) is
  documented in prose in [local-development.md](../usage/local-development.md).
  The removal itself was already recorded in
  [2026-07-20-go-rewrite-and-rls-cutover/migration-notes.md](2026-07-20-go-rewrite-and-rls-cutover/migration-notes.md);
  this deletes the now-dead template. (The frozen
  [docs/history/nuxt-migration.md](../history/nuxt-migration.md) still
  mentions it as a record of the old project state — left as-is.)
- **Restructured the root [`.env.example`](../../.env.example) by consumer** —
  three sections: Go backend (`server/internal/config`), AI provider (also
  read by Go), docker-compose only (`SERVER_PORT`, `APP_PORT`, `POSTGRES_*`).
  This fixes two mislabels: `SERVER_PORT` and `APP_PORT` were filed under
  "Go server"/"Frontend" headings though neither Go nor Nuxt reads them —
  they are host-side compose port mappings (containers listen on `:8080`/`:3000`
  regardless). Added commented-out lines for the optional `INSIDEOUT_ADDR`,
  `INSIDEOUT_COOKIE_SECURE`, `INSIDEOUT_DEV_CORS`; kept the bilingual
  `DATABASE_URL` setup-(a)/(b) explanation and a note that the frontend needs
  nothing from this file.
- **Added `server/scripts/dev.sh`** (moved to the repo root the next day —
  see [2026-07-28 — dev.sh moves to the repo root](2026-07-28-devsh-root.md)) — exports
  the root `.env` (the sanctioned `set -a; source; set +a` pattern) and execs
  a command from `server/`, so `./scripts/dev.sh go run ./cmd/insideout`
  replaces the hand-typed incantation. It never prints env values and fails
  loudly if `.env` is missing. `smoke.sh` keeps its own inline sourcing — it
  is self-contained and untouched.
- **Docs updated to use it:** [CLAUDE.md](../../CLAUDE.md) key commands,
  [local-development.md](../usage/local-development.md) (Environment + Testing
  sections; the previous `env $(cat .env | xargs)` tip also broke on values
  containing spaces), and the [HANDOFF.md](../HANDOFF.md) verify block.

## Verification

- `bash -n` clean; `./scripts/dev.sh go version` execs with the real `.env`
  exported and prints nothing sensitive; no-args prints usage.
- `git status` confirms only the intended paths changed; `.env` itself
  untouched (never read).

## Operator notes

None — the `.env` format is unchanged; existing `.env` files keep working.
