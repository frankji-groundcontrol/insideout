# 2026-10-05 — Roadmap progress SSE

**Scope:** the Progress view updates while work runs, instead of only
on load. No migration. One server instance, same bound as canvas
presence.

## What changed

- `GET /api/v1/projects/{pid}/roadmap/progress/stream` (SSE) writes the
  same payload as `GET /roadmap/progress` as `event: progress` on
  connect, and again when a roadmap node is created, updated, moved,
  deleted, replaced, or expanded. `internal/progresshub` fans the
  project id out in memory; the handler re-reads the tree.
- The store notifies that hub after a committed node write
  (`Store.OnRoadmapChange`, wired in `NewServer`).
- The Flutter progress view subscribes (`ApiClient.progressStream`) and
  replaces the snapshot it is showing. A drop reconnects with the same
  backoff as presence.

## Verification

- `cd server && go test -count=1 -timeout 180s ./internal/api/ -run TestRoadmapProgressStream_StatusChangeEmitsEvent` — pass (1.362s). A pending → in_progress PATCH produced an `event: progress` frame carrying the node id and `in_progress`.
- `cd server && go test -count=1 -timeout 90s -c -o /dev/null ./internal/api/` — compile pass.
- `cd client && flutter test test/sse_test.dart test/roadmap_page_test.dart --name 'readProgressSse|a progress event replaces'` — 2 passed. The page test replaces the rendered snapshot from a progress frame.
- `cd client && flutter test test/roadmap_page_test.dart test/roadmap_canvas_test.dart test/roadmap_presence_test.dart test/sse_test.dart` — 18 passed. The full suite was not run.

## Operator

Nothing to configure. Ship with `railway up --service server` and
`railway up --service app`, then restart `app` so nginx re-resolves the
server.
