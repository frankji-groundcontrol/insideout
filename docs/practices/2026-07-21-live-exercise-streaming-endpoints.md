# Practice: Live-exercise streaming (SSE) endpoints before completion

**Date**: 2026-07-21

## Trigger

Any work on an SSE or streaming endpoint (here: `POST /api/v1/conversations/{id}/messages`), on middleware that wraps `http.ResponseWriter`, or on the provider client that feeds the stream. Compiling and unit-testing the handler is not enough — streaming failures in this repo have been silent until a real client hit a real server.

## Sequence / guardrail

1. Start the real server (not `httptest`) with real config.
2. Drive the endpoint with a real streaming client, unbuffered:
   ```bash
   curl -N -X POST http://localhost:<port>/api/v1/conversations/<id>/messages \
     -H 'Content-Type: application/json' \
     -b '<auth cookies>' \
     -d '{"content":"hello"}'
   ```
3. Watch the raw event stream: events must arrive **incrementally** (the full sequence `message_start` / `delta` / ... / `message_end`, not one buffered blob at the end), and the payloads must contain the actual assistant text.
4. When adding any new `ResponseWriter`-wrapping middleware, re-run this check — embedding the `http.ResponseWriter` interface does not preserve `http.Flusher`, and the loss is invisible at compile time.

## Verification

A live `curl -N` transcript showing incremental, well-formed SSE events end to end through the full middleware chain and the real provider path.

Standing automation: `server/scripts/smoke.sh` performs exactly this check (plus the other four surfaces) on every run — it boots the server, drives `POST /conversations/{id}/messages` with `curl -N`, and fails unless `message_start`, `delta`, and `message_end` all arrive. See [docs/changelogs/2026-07-27-live-smoke-test.md](../changelogs/2026-07-27-live-smoke-test.md).

## Failure signals

- `sse: response writer does not support flushing` — a wrapper in the middleware chain dropped `http.Flusher` (BUG-010: the logging middleware's `statusRecorder` did exactly this, unconditionally, for every SSE request).
- Stream "works" but arrives as one buffered chunk at `message_end` — flushing silently lost somewhere.
- A generic empty-response error (`no response`) while a hand-replayed `curl` of the same provider request succeeds — the provider response is being mis-parsed (BUG-009: a `thinking` content block hid the real text from the old client). Both of this repo's streaming bugs were independent, silent, and only surfaced live.

## Related

- [BUG-010: logging middleware silently broke every SSE response](../issues/2026-07-21-bug-010-sse-flusher-swallowed-by-logging-middleware.md)
- [BUG-009: langchaingo dropped a real answer behind a thinking block](../issues/2026-07-21-bug-009-langchaingo-removed.md)
- [Learning records](../learning/README.md)
