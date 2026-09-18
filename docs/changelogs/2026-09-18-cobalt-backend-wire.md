# 2026-09-18 — Cobalt Flutter wired to a real `/api/v1`

Plan: [cobalt-backend](../plans/2026-09-18-cobalt-backend.md).
Restyle pixels were already in; this slice is connect, login, and the
loop on real rows. No theme change.

## What changed

- Web Coach SSE uses the existing `fetch` + `ReadableStream` transport
  with POST (`coachSseOpen` / `readCoachSse`) so deltas arrive while
  the stream is open. Closes the Coach half of
  [BUG-014](../issues/2026-08-23-bug-014-dio-web-sse-never-streams.md).
  Native still uses dio. `decodeSseBytes` copies `Uint8List` chunks to
  `List<int>` before `utf8.decoder` — the live web walk threw
  `Utf8Decoder is not a subtype of StreamTransformer<Uint8List, String>`.
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
  `message_start` / `delta` / `message_end` (`curl -N` at 9 / 11 / 15 ms).
- `flutter analyze --no-fatal-infos` (infos only) and `flutter test`
  (57 passed).
- Signed-in Flutter web walk on `API_BASE=http://127.0.0.1:8080/api/v1`:
  [design-qa](../design-qa/2026-09-18-cobalt-backend-walk.md).

## Operator

Client-only. Ship with `railway up --service app` after merge if you
want the hosted bundle to match. Do not seed a shared remote database
with the local demo identity.
