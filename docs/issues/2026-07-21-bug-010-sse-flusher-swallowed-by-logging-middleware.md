# BUG-010: logging middleware silently broke every SSE response

**Found**: 2026-07-21, during the InsideOut rewrite (P4), the same real-coaching-message debugging session as BUG-009 — this one surfaced first, before the langchaingo bug even got a chance to run.

**Symptom**: every call to `POST /conversations/{id}/messages` (the coach's SSE streaming endpoint) failed immediately with `sse: response writer does not support flushing`, even though the handler and its middleware chain compile and pass every other request type fine.

**Root cause**: `internal/api/middleware.go`'s `withLogging` wraps the `http.ResponseWriter` in a `*statusRecorder` to capture the status code for the request log line:

```go
type statusRecorder struct {
	http.ResponseWriter
	status int
}
```

Embedding the `http.ResponseWriter` *interface* only promotes the methods that interface declares (`Header`/`Write`/`WriteHeader`) onto `*statusRecorder` — it does **not** promote `Flush` from the separate `http.Flusher` interface, even though the real, concrete `http.ResponseWriter` underneath (from `net/http`'s server) almost always implements it. So `internal/agent/sse.go`'s `w.(http.Flusher)` type assertion — checking the *wrapped* `*statusRecorder`, not the original writer — failed every time, for every SSE-streaming endpoint, unconditionally. This is a well-known but easy-to-miss Go gotcha with response-writer-wrapping middleware, and it went unnoticed all session because nothing had exercised the coach's streaming endpoint against a live, non-unit-test server until this exact moment.

**Fix**: added an explicit `Flush()` method to `*statusRecorder` that forwards to the underlying writer if it implements `http.Flusher`:

```go
func (rec *statusRecorder) Flush() {
	if f, ok := rec.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
```

**Why it matters**: any `http.ResponseWriter`-wrapping middleware (logging, metrics, response-recording, gzip, etc.) must explicitly re-implement every "extra" interface (`http.Flusher`, `http.Hijacker`, `http.Pusher`) it wants to preserve — embedding the plain `http.ResponseWriter` interface is not enough, and the failure is silent at compile time (the type assertion just returns `ok = false` at runtime) rather than a build error. Worth grepping for every response-writer wrapper in this codebase if a new streaming/websocket endpoint is ever added.
