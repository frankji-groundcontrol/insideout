package api

import (
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
)

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Pool.Ping(r.Context()); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "database unreachable", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
