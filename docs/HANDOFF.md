# HANDOFF

Resume here. The authoritative multi-task board is
[`docs/plans/README.md`](plans/README.md). The target product experience is
[`PRODUCT.md`](../PRODUCT.md), and the running system is described by
[`docs/architecture/`](architecture/index.md).

## What to do next

1. Railway `app` serves Flutter web. The leftover Nuxt tree is gone
   ([changelog](changelogs/2026-08-18-delete-nuxt-app.md)). Remaining
   Flutter item: Android release when an SDK exists. Plan:
   [`docs/plans/2026-08-17-flutter-client.md`](plans/2026-08-17-flutter-client.md).

```bash
# Hosted web (same-origin /api/v1 via nginx)
open https://app-production-591e.up.railway.app

# Local Flutter against the hosted API
cd client && flutter run -d chrome --dart-define=API_BASE=https://server-production-9c338.up.railway.app/api/v1
```

The PRODUCT.md version-first slice stays on the board as P2.

## Worktree to preserve

Uncommitted work includes the Flutter Railway host, the Nuxt `app/`
deletion, env-catalog updates, and usage / architecture / changelog
records. Do not revert unrelated edits.

The public instance is documented in
[usage/deployment.md](usage/deployment.md#railway-current-public-deploy)
(`https://app-production-591e.up.railway.app`). The board's next product
slice is unchanged.

Their exact status, next action, blocker, and plan or record link are on the
[task board](plans/README.md). Review and checkpoint them one task at a time;
do not use `git add .`.

## Non-negotiables

- Never read or expose `.env` or `.env.local`.
- DB-dependent behavior requires a real PostgreSQL check and must stay inside
  the `insideout` schema.
- Commands and environment-safe launch instructions live in
  [`docs/usage/local-development.md`](usage/local-development.md).
