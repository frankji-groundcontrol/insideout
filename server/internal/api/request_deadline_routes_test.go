package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// The exemptions are keyed on mux patterns, so a renamed route would quietly
// lose its exemption and start being cut at the deadline: a streamed model
// turn or a passive stream dying 25s in. Every listed pattern must still be a
// route this package registers.
func TestRequestDeadline_ExemptionsNameRegisteredRoutes(t *testing.T) {
	registered := map[string]bool{}
	for _, route := range registeredRoutes(t) {
		registered[route] = true
	}
	for pattern := range deadlineExempt {
		if !registered[pattern] {
			t.Errorf("deadlineExempt lists %q, which no mux registers: the route was renamed or removed", pattern)
		}
	}
}

// The middleware itself, with no database: a plain handler is cut at the
// deadline and still sees its path values, an exempt pattern gets no deadline.
func TestRequestDeadline_CutsPlainHandlersAndSparesExemptOnes(t *testing.T) {
	old := requestDeadline
	requestDeadline = 50 * time.Millisecond
	t.Cleanup(func() { requestDeadline = old })

	mux := http.NewServeMux()
	var plainID string
	var plainErr error
	mux.HandleFunc("GET /plain/{id}", func(w http.ResponseWriter, r *http.Request) {
		plainID = r.PathValue("id")
		<-r.Context().Done()
		plainErr = r.Context().Err()
	})
	hasDeadline := true
	mux.HandleFunc("GET /api/v1/projects/{pid}/presence/stream", func(w http.ResponseWriter, r *http.Request) {
		_, hasDeadline = r.Context().Deadline()
		time.Sleep(100 * time.Millisecond) // longer than the plain deadline
		if r.Context().Err() != nil {
			t.Errorf("the exempt stream was cancelled: %v", r.Context().Err())
		}
	})
	h := (&Server{}).withRequestDeadline(mux)

	start := time.Now()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/plain/abc", nil))
	if plainErr == nil || time.Since(start) > 2*time.Second {
		t.Errorf("the plain handler's context ended with %v after %s, want a deadline error near 50ms", plainErr, time.Since(start))
	}
	if plainID != "abc" {
		t.Errorf("the plain handler saw path value %q, want abc: the middleware must not lose the mux's path values", plainID)
	}

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/projects/p1/presence/stream", nil))
	if hasDeadline {
		t.Error("the exempt stream route ran under a request deadline")
	}
}
