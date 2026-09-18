# 2026-09-18 — Cobalt Flutter wired to a real `/api/v1`

Plan: [cobalt-backend](../plans/2026-09-18-cobalt-backend.md).
Restyle pixels were already in; this slice is connect, login, and the
loop on real rows. No theme change.

## What changed

- Web Coach SSE uses the existing `fetch` + `ReadableStream` transport
  with POST (`coachSseOpen` / `readCoachSse`) so deltas arrive while
  the stream is open. Closes the Coach half of
  [BUG-014](../issues/2026-08-23-bug-014-dio-web-sse-never-streams.md).
  Native still uses dio.
- `LoopFocus` seeds from real loads: Ideas and workspace remember a
  project ID when the workspace has one; Evidence remembers the route
  id in `initState` (not during `build`). Dock hops keep those IDs.

## What did not change

No new routes, schema, or product verbs. Tokens stay
`accessToken` / `refreshToken` in secure storage. Path URL strategy
unchanged. Hosted production still serves the pre-restyle bundle until
someone runs `railway up --service app` after merge.

## Verification

- Local Go API (offline coach) + seed database: login, workspaces,
  idea persist, convert, commit, roadmap node, project updates, Coach
  `message_start` / `delta` / `message_end`.
- `flutter analyze --no-fatal-infos` and `flutter test` (this commit
  adds Coach POST contract tests; full suite after push).
- Signed-in Flutter web walk against that real `API_BASE` follows in
  the same plan (not a fake `ApiClient`).

## Operator

Client-only. Ship with `railway up --service app` after merge if you
want the hosted bundle to match. Do not seed a shared remote database
with the local demo identity.
