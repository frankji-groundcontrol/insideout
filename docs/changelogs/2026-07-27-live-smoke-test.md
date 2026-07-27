# 2026-07-27 — Live end-to-end smoke test for all five surfaces

## What changed

Added `server/scripts/smoke.sh`, a single reusable, rerun-safe live functional
test that drives all five product surfaces over real HTTP against the real
PostgreSQL — no mocks. It boots the server on a random high port (or reuses a
running one via `SMOKE_BASE`), registers fresh uniquely-named users through curl
cookie jars, and asserts, exiting non-zero on any failure:

- **S1 PRD coach SSE** — `POST /conversations/{id}/messages` must stream
  `message_start` → `delta` → `message_end` (exercised live, not traced).
- **S2 AI roadmap** — build-from-PRD (`nodeCount > 0`), list, node expand, and
  manual create/PATCH-status/move/delete.
- **S3 GitHub sync** — repo-URL validation (bad URL → 400), sync-with-no-repo
  → 400, and a real linked-repo sync (accepts 200/429, tolerates upstream
  404/502, fails on 400-with-repo/401/403/500).
- **S4 project-updates timeline** — progress/blocker/note add, invalid-kind
  rejection, edit, delete, and embedding in the project view.
- **S5 authz + negatives** — a second tenant gets 403/404 on every foreign
  resource, and an empty cookie jar gets 401.

It closes out task #75. The prior blocker premise — "authenticated curl cannot
use the httpOnly session cookie" — was false: `httpOnly` only hides the cookie
from JavaScript; curl stores and replays it via its cookie jar (`-c`/`-b`).

## Verification

Run twice against the real DB, both green: `48 passed, 0 failed`. The linked
GitHub sync pulled live commits (`added=3`) and the coach SSE stream completed
end-to-end. Rerun-safety comes from the per-run `$RUN` suffix on every entity.

```bash
cd server && ./scripts/smoke.sh                     # boots its own server
SMOKE_BASE=http://127.0.0.1:54321 ./scripts/smoke.sh  # against a running one
```

When the script boots the server itself it forces `ANTHROPIC_AUTH_TOKEN=` so the
offline template coach/planner make S1/S2 deterministic; pass `SMOKE_BASE` to
exercise a real-key server instead.

## Operator notes

None — this is a test harness only. It requires `jq` and `curl` on `PATH` and a
reachable `DATABASE_URL` in `../.env`.
