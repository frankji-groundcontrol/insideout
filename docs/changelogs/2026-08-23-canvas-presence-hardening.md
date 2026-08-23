# 2026-08-23 — Canvas presence hardening: reconnect + a web SSE transport that streams

Plan: [canvas-presence-hardening](../plans/2026-08-23-canvas-presence-hardening.md).
The board's "real-time multi-user presence" follow-on turned out to be
already shipped (frontier 2026-08-22); what was open was robustness and
proof — plus one real bug the live proof exposed.

## Reconnect on stream drop

`_RoadmapPageState` now runs the presence subscribe in a loop: any
stream end or error re-opens after a capped backoff (0.5 s doubling to
5 s) while the page is mounted; 401/403/404 stop the loop instead of
hammering the API. The backoff is a cancellable `Timer`, cancelled in
`dispose` (a raw `Future.delayed` left a live timer at teardown — the
canvas widget test caught it). Widget test: fake stream dies once,
chips come back on the reopened stream.

## Web SSE transport (the real bug)

Live two-tab verification showed presence chips never rendered on the
hosted web build even though the server stream was healthy: dio's web
adapter is XHR + `arraybuffer`, so `ResponseType.stream` delivers the
body only when the connection **closes** — a long-lived SSE stream
never yields a frame in the browser. New `sse_transport_web.dart`
reads the stream with `fetch` + `ReadableStream` (package:web),
selected by conditional import; native targets keep the dio path.
Chunks decode through `utf8.decoder` so multibyte CJK split across
chunk boundaries stays safe.

## Verification

- `flutter analyze` no new issues; `flutter test` 46/46 (new
  reconnect test included).
- Live, local build against the hosted API, two browser tabs on the
  same project's canvas: each tab registers (two who-is-here chips),
  and tab B's pointer moves render as a labeled vermilion cursor in
  tab A (screenshot evidence in the plan thread).

## Known limitation (recorded, not fixed here)

Coach SSE on web uses the same dio stream path, so coach replies on
web arrive when the stream ends rather than typing live. The new
transport is the fix when that becomes worth doing — tracked as
[BUG-014](../issues/2026-08-23-bug-014-dio-web-sse-never-streams.md),
with the general lesson in
[learning](../learning/2026-08-23-dio-web-does-not-stream-sse.md).

## Operator notes

Client-only; ships with the same `railway up --service app` as the
font-flash fix. No server or database changes.
