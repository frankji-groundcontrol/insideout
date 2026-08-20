# HANDOFF

Resume here. The authoritative multi-task board is
[`docs/plans/README.md`](plans/README.md). The target product experience is
[`PRODUCT.md`](../PRODUCT.md), and the running system is described by
[`docs/architecture/`](architecture/index.md).

## What to do next

1. Owner/app DB roles. Migrate with `DATABASE_OWNER_URL` (`insideout_owner`,
   NOSUPERUSER). Runtime stays `DATABASE_URL` (`insideout_app`).
   Plan: [`docs/plans/2026-08-19-owner-app-roles.md`](plans/2026-08-19-owner-app-roles.md).
2. Restore Ink & Seal on Flutter (native fonts, collaborative canvas).
   Plan: [`docs/plans/2026-08-19-restore-ink-seal.md`](plans/2026-08-19-restore-ink-seal.md).
2. Flutter Android release when an SDK exists
   ([`docs/plans/2026-08-17-flutter-client.md`](plans/2026-08-17-flutter-client.md)).

```bash
# Local Flutter against the hosted API (real data; CORS already allows localhost)
cd client && flutter run -d chrome --web-port=5173 --web-hostname=localhost \
  --dart-define=API_BASE=https://server-production-9c338.up.railway.app/api/v1
```

Local `.env` `DATABASE_URL` currently fails SASL against the shared
instance, so do not boot the Go server from this machine until that
password is refreshed. Use the hosted API.

The PRODUCT.md version-first slice stays on the board as P2.

## Worktree to preserve

Uncommitted work includes Ink & Seal Flutter tokens/auth chrome, the
Flutter Railway host, the Nuxt `app/` deletion, and env/docs records.
Do not revert unrelated edits.

Their exact status, next action, blocker, and plan or record link are on the
[task board](plans/README.md). Review and checkpoint them one task at a time;
do not use `git add .`.

## Non-negotiables

- Never read or expose `.env` or `.env.local`.
- DB-dependent behavior requires a real PostgreSQL check and must stay inside
  the `insideout` schema.
- Commands and environment-safe launch instructions live in
  [`docs/usage/local-development.md`](usage/local-development.md).
