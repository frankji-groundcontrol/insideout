# Wrapping http.ResponseWriter silently drops http.Flusher (and Hijacker, Pusher)

**Date**: 2026-07-21

## What was learned

Embedding the `http.ResponseWriter` **interface** in a middleware wrapper
struct only promotes the three methods that interface declares (`Header`,
`Write`, `WriteHeader`). It does **not** promote `Flush` from the separate
`http.Flusher` interface — even though the concrete writer underneath (from
`net/http`'s server) implements it. Downstream code doing
`w.(http.Flusher)` asserts against the *wrapper*, gets `ok == false`, and
streaming breaks unconditionally.

Two properties make this dangerous:

1. **The failure is silent at compile time.** Everything builds; the type
   assertion simply fails at runtime, for every streaming request, only on
   the streaming code path.
2. **It hides until a streaming endpoint is actually exercised against a
   real server.** In InsideOut, a logging middleware's `*statusRecorder`
   wrapper broke every SSE response, and nothing noticed until the first
   live coaching message was sent — unit tests and non-streaming requests
   all passed.

The fix: every `http.ResponseWriter`-wrapping middleware (logging, metrics,
gzip, response-recording, ...) must explicitly forward each "extra"
interface it wants to preserve — `http.Flusher`, `http.Hijacker`,
`http.Pusher` — e.g. a `Flush()` method that type-asserts the underlying
writer and delegates. See `server/internal/api/middleware.go`.

## Evidence

[BUG-010](../issues/2026-07-21-bug-010-sse-flusher-swallowed-by-logging-middleware.md)
— symptom was `sse: response writer does not support flushing` on every call
to the coach's SSE endpoint.

## Scope

All Go HTTP middleware that wraps `http.ResponseWriter`, in this codebase
and any other; any endpoint relying on `http.Flusher` (SSE), `http.Hijacker`
(websockets), or `http.Pusher`.

## When to apply again

- Writing or reviewing any new response-writer wrapper: add explicit
  forwarding for Flusher/Hijacker/Pusher (or at minimum the ones the app's
  endpoints need) at the moment the wrapper is created, not when streaming
  breaks.
- Adding any new streaming or websocket endpoint: grep for every existing
  response-writer wrapper in the middleware chain and verify forwarding.
- Defining "done" for a streaming feature: it is not done until the endpoint
  has been exercised against a real, live server — compile success and unit
  tests cannot catch this class of bug.
