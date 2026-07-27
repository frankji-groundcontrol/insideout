package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
)

type requestIDKey struct{}

// withRequestID is the outermost middleware. Besides stamping a request
// id, it installs the mutable user-id holder (see context.go) that
// requireAuth populates further down the chain and withLogging reads
// after the chain unwinds.
func (s *Server) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 8)
		_, _ = rand.Read(buf)
		id := hex.EncodeToString(buf)
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		ctx = newUserHolderContext(ctx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// statusRecorder captures the status code written so logging middleware
// can report it after the handler completes. wroteHeader records whether
// the response has been committed (headers sent) — a bare Write commits
// them implicitly with a 200, so it must set the flag too, or withRecover
// can't tell a not-yet-started response from a mid-stream one (F21).
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.wroteHeader = true
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *statusRecorder) Write(b []byte) (int, error) {
	rec.wroteHeader = true
	return rec.ResponseWriter.Write(b)
}

// Flush forwards to the underlying ResponseWriter's Flush, if it has
// one. Embedding the http.ResponseWriter *interface* only promotes the
// methods that interface declares (Header/Write/WriteHeader) — it does
// NOT promote Flush from http.Flusher, even though the real concrete
// writer underneath almost always implements it. Without this,
// *statusRecorder failed the `w.(http.Flusher)` type assertion in
// internal/agent/sse.go, silently breaking every SSE (coach streaming)
// response once withLogging wrapped it — found by actually running a
// real coaching message through the live server.
func (rec *statusRecorder) Flush() {
	if f, ok := rec.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if userID, ok := UserID(r.Context()); ok {
			attrs = append(attrs, "user_id", userID.String())
		}
		s.log.Info("request", attrs...)
	})
}

// maxBodyBytes caps every request body. All payloads here are JSON text
// (auth, ideas, PRD sections, coach messages); 1 MiB is generous for any
// real request and stops an unauthenticated client from parking a
// slow-drip multi-GB body on an open connection. Complements the
// ReadHeaderTimeout/ReadTimeout/IdleTimeout set on the http.Server.
const maxBodyBytes = 1 << 20 // 1 MiB

// withMaxBody wraps the request body in an http.MaxBytesReader so an
// oversized body fails the decode instead of being buffered without bound.
// ponytail: returns the generic decode 400 rather than a precise 413 —
// the connection is cut at the cap either way, which is the whole point.
func (s *Server) withMaxBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("panic", "error", rec, "path", r.URL.Path)
				// Only write a JSON error if the response hasn't been
				// committed. Once headers are out (e.g. an SSE stream is
				// mid-flight) a JSON error body would be appended onto the
				// event stream and corrupt it — log and let the stream's own
				// error handling / client timeout take over instead (F21).
				if sr, ok := w.(*statusRecorder); !ok || !sr.wroteHeader {
					httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// withDevCORS is only mounted when INSIDEOUT_DEV_CORS=1, for hitting the
// API directly with curl/tests without going through the Nuxt proxy.
func (s *Server) withDevCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireAuth resolves the access token (cookie or bearer), verifies it,
// and injects the user id into the request context — or writes 401.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerOrCookie(r)
		if token == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "authentication required", "", nil)
			return
		}
		userID, err := s.tokens.VerifyAccessToken(token)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid or expired session", "", nil)
			return
		}
		setUserID(r.Context(), userID)
		next.ServeHTTP(w, r)
	}
}
