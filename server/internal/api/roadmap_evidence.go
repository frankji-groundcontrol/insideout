package api

import (
	"errors"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// handleListEvidence returns a node's delivery evidence, newest first.
func (s *Server) handleListEvidence(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	nid, ok := pathUUID(w, r, "nid")
	if !ok {
		return
	}
	rows, err := s.store.ListRoadmapEvidence(r.Context(), userID, nid)
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	if err != nil {
		s.log.Error("evidence list", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	if rows == nil {
		rows = []store.RoadmapEvidence{}
	}
	httpx.WriteJSON(w, http.StatusOK, rows)
}
