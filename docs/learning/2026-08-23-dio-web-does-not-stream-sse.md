# dio on Flutter web does not stream SSE — fetch + ReadableStream does

**Learned** 2026-08-23 from
[BUG-014](../issues/2026-08-23-bug-014-dio-web-sse-never-streams.md)
while live-proving canvas presence
([changelog](../changelogs/2026-08-23-canvas-presence-hardening.md)).

## The lesson

On Flutter web, dio's adapter is XHR with `arraybuffer`; a
`ResponseType.stream` body is delivered once, at connection close.
Long-lived SSE over dio on web is therefore silently dead — no error,
no frames. `fetch()` + `ReadableStream.getReader()` (package:web,
`dart:js_interop`) is the browser transport that yields chunks while
the connection is open. Gate it behind a conditional import so native
targets keep dio.

## When to apply

Any Flutter web code that consumes SSE, chunked uploads of progress
events, or any "stream that must emit before the socket closes".
Verify streaming on the **web** transport specifically: a healthy
`curl` of the same endpoint proves the server, not the client.

## Evidence

Presence chips rendered on native-style dio streaming assumptions:
never on web, always via curl. After swapping the web path to fetch +
ReadableStream, two browser tabs showed each other's chips and live
cursors against the same server. Coach POST on web joined that
transport on 2026-09-18 (`method` + `body` on `fetchSseStream`).
