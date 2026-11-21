package api

import (
	"errors"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

type moveRoadmapNodeRequest struct {
	ParentID *string `json:"parentId"`
	Position *int    `json:"position"`
}

func (s *Server) handleMoveRoadmapNode(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	nid, ok := pathUUID(w, r, "nid")
	if !ok {
		return
	}
	var req moveRoadmapNodeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	// parentId present-and-null → make it a root; absent → root too (move
	// always sets the parent explicitly).
	parentID, ok := parseOptionalUUID(w, req.ParentID)
	if !ok {
		return
	}
	position := 0
	if req.Position != nil {
		position = *req.Position
	}

	n, err := s.store.MoveRoadmapNode(r.Context(), userID, nid, parentID, position)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "node not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the project owner or a workspace admin can change the roadmap", "", nil)
		return
	}
	if errors.Is(err, store.ErrConflict) {
		httpx.WriteError(w, http.StatusConflict, "cannot move a node under itself or a descendant", "", nil)
		return
	}
	if err != nil {
		s.log.Error("move roadmap node", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, roadmapNodeResponse(*n, s.now()))
}
