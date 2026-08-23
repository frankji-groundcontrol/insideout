# BUG-014 — dio's web adapter never streams SSE (body arrives at connection close)

**Found** 2026-08-23, during the canvas presence hardening live proof
([changelog](../changelogs/2026-08-23-canvas-presence-hardening.md)).

## Symptom

Presence chips and live cursors never rendered on the hosted Flutter
web app even though the server stream was verifiably healthy: `curl`
on the same stream showed the initial `event: presence` snapshot
immediately, and the browser's own tab was registered server-side.
The client simply never processed a frame while the stream was open.

## Root cause

`dio_web_adapter` implements requests with `XMLHttpRequest` and
`responseType = 'arraybuffer'`, reading `xhr.response` only when the
request completes. With `ResponseType.stream` the "stream" therefore
emits exactly once — when the connection **closes**. A long-lived SSE
connection never closes, so `await for` on the response stream never
yields. Any Flutter web feature built on dio SSE silently receives
nothing until the socket dies.

## Fix

`client/lib/api/sse_transport_web.dart` (conditional import, selected
by `kIsWeb` in `ApiClient.presenceStream`): `fetch()` +
`ReadableStream.getReader()` — the browser transport that actually
streams — yielding `Uint8List` chunks decoded through `utf8.decoder`.
Native targets keep the dio path, which streams correctly there.

## Still affected (follow-on)

The Coach SSE path (`sendCoach`) uses the same dio stream path, so on
web a coach reply arrives whole when the stream ends instead of typing
live. The transport above is the fix when live typing on web becomes
worth doing; native clients are unaffected.

## Detection hint

Server-side health plus client-side silence on web = suspect the
client transport, not the server. Grep the built `main.dart.js` for an
ASCII string from the suspected code path to prove which bundle is
served before debugging behavior (long CJK literals get escaped and
make unreliable markers).
