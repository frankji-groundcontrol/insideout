# HANDOFF

Resume here. The authoritative multi-task board is
[`docs/plans/README.md`](plans/README.md). The target product experience is
[`PRODUCT.md`](../PRODUCT.md), and the running system is described by
[`docs/architecture/`](architecture/index.md).

## What to do next

1. Roadmap GitHub loop: complete and live — webhook, guide scaffolding
   (API/CLI/MCP), installation tokens, and the guide-matched evidence
   loop, with the app installed and real deliveries flowing
   ([webhook](changelogs/2026-08-21-github-webhook.md),
   [guide+MCP](changelogs/2026-08-21-guide-and-mcp.md),
   [evidence](changelogs/2026-08-21-evidence-loop.md),
   [dogfood+cleanup](changelogs/2026-08-21-dogfood-and-cleanup.md)).
   Dogfood credentials: `~/.zcode-tracks/insideout-dogfood-credentials.txt`
   (local, never committed).
   Plan: [`docs/plans/2026-08-21-roadmap-parity-and-github.md`](plans/2026-08-21-roadmap-parity-and-github.md).
2. CLI / MCP parity: Stage 3 — write verbs + agent vocabulary
   (`context`, `focus`, `checkpoint/report`, `propose`, `version`) as
   API routes first, then projected. `insideout-mcp` (14 tools) and the
   CLI are live.
   Plan: [`docs/plans/2026-08-21-cli-mcp-parity.md`](plans/2026-08-21-cli-mcp-parity.md).
3. Restore Ink & Seal on Flutter: the collaborative canvas (sibling bands,
   minimap) is the only open slice item. Native fonts are bundled and
   visually signed off (2026-08-20,
   [changelog](changelogs/2026-08-20-native-fonts-bundling.md)).
   Plan: [`docs/plans/2026-08-19-restore-ink-seal.md`](plans/2026-08-19-restore-ink-seal.md).
4. Flutter Android release when an SDK exists
   ([`docs/plans/2026-08-17-flutter-client.md`](plans/2026-08-17-flutter-client.md)).

```bash
# Local Flutter against the hosted API (real data; CORS already allows localhost)
cd client && flutter run -d chrome --web-port=5173 --web-hostname=localhost \
  --dart-define=API_BASE=https://insideout.yalotein.net/api/v1
```

Local `.env` `DATABASE_URL` works again (repaired 2026-08-20 with the
known-good app DSN); the Go server can boot from this machine.

The two-role DB model is live end-to-end: shared instance, Railway
(`DATABASE_URL` app + `DATABASE_OWNER_URL` owner), and both `.env`
files ([changelog](changelogs/2026-08-20-owner-app-roles-shared-instance.md)).
Railway autodeploy is off for both services, and `railway redeploy`
re-runs the last image instead of building — after changes land on
`main`, ship them explicitly: any `server/` change with
`railway up --service server` (Go API), any `client/` change with
`railway up --service app` (Flutter web host). **After a `server`
deploy, restart `app`** (`railway redeploy --service app`): its nginx
resolves the server's internal address only at startup, so `/api`
via the domain 504s until then (verify with `/api/v1/me` → 401, not
`/healthz`, which the SPA answers).

The PRODUCT.md version-first slice stays on the board as P2.

## Worktree to preserve

Both machines and origin were synced at the last `[checkpoint]` commit; the
worktree is clean. Do not revert unrelated edits.

Their exact status, next action, blocker, and plan or record link are on the
[task board](plans/README.md). Review and checkpoint them one task at a time;
do not use `git add .`.

## Non-negotiables

- Never read or expose `.env` or `.env.local`.
- DB-dependent behavior requires a real PostgreSQL check and must stay inside
  the `insideout` schema.
- Commands and environment-safe launch instructions live in
  [`docs/usage/local-development.md`](usage/local-development.md).
