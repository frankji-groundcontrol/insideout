package api

import (
	"context"
	"net/http"
	"time"
)

// requestDeadline bounds every request that is not listed in deadlineExempt.
// A database that stops answering never delivers the statement to its
// server-side statement_timeout, so without a client-side deadline a handler
// waits until the caller hangs up (rv4-REL-02). 25s sits above the 15s
// statement timeout and the 15s GitHub client timeout and below the web
// client's 30s receive timeout, so the server answers before the client gives
// up. A var only so tests can shorten it.
var requestDeadline = 25 * time.Second

// deadlineExempt lists the mux patterns that legitimately outlive
// requestDeadline: the passive SSE streams (they run until the tab leaves and
// arm their own write deadline per frame), the coach turn (one streamed model
// call, bounded by the agent) and the AI planner calls with their confirms,
// which write a whole drafted tree. request_deadline_routes_test.go fails when
// a pattern here stops naming a registered route.
var deadlineExempt = map[string]bool{
	"GET /api/v1/projects/{pid}/presence/stream":         true,
	"GET /api/v1/projects/{pid}/roadmap/progress/stream": true,
	"POST /api/v1/conversations/{id}/messages":           true,
	"POST /api/v1/prds/{pid}/build":                      true,
	"POST /api/v1/prds/{pid}/build/confirm":              true,
	"POST /api/v1/roadmap/{nid}/expand":                  true,
	"POST /api/v1/roadmap/{nid}/expand/confirm":          true,
}

// withRequestDeadline gives every non-exempt request a context that expires
// after requestDeadline, so each store call a handler makes with r.Context()
// stops waiting on a stalled database. The exemption follows the pattern the
// mux would dispatch to.
func (s *Server) withRequestDeadline(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); deadlineExempt[pattern] {
			mux.ServeHTTP(w, r)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), requestDeadline)
		defer cancel()
		mux.ServeHTTP(w, r.WithContext(ctx))
	})
}
