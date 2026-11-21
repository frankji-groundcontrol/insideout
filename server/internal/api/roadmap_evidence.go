package api

import (
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
	if err != nil {
		s.writeStoreError(w, "evidence list", err, errMsgs{store.ErrForbidden: "you must be a member of this workspace"})
		return
	}
	if rows == nil {
		rows = []store.RoadmapEvidence{}
	}
	httpx.WriteJSON(w, http.StatusOK, rows)
}
