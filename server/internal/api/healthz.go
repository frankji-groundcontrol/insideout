package api

import (
	"context"
	"net/http"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
)

// healthzPingTimeout bounds the health probe's database ping. A database that
// stops answering never delivers the statement to its server-side
// statement_timeout, so only a client-side deadline lets the probe say 503
// inside the platform's own health-check timeout (rv4-REL-02).
const healthzPingTimeout = 2 * time.Second

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), healthzPingTimeout)
	defer cancel()
	if err := s.store.Pool.Ping(ctx); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "database unreachable", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
