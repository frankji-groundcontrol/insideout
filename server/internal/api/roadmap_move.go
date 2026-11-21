package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

type moveRoadmapNodeRequest struct {
	// ParentID absent keeps the current parent (a position-only move); null or
	// "" makes the node a root; an id re-parents it (rv5-PARC-05).
	ParentID json.RawMessage `json:"parentId"`
	Position *int            `json:"position"`
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
	var parentID *uuid.UUID
	var err error
	if req.ParentID == nil {
		// shortcut: the parent is read outside the move's lock, so a re-parent
		// landing in between is undone; keep it in MoveRoadmapNode if that matters.
		var cur *store.RoadmapNode
		if cur, err = s.store.GetRoadmapNode(r.Context(), userID, nid); err == nil {
			parentID = cur.ParentID
		}
	} else {
		var raw *string
		if json.Unmarshal(req.ParentID, &raw) != nil {
			httpx.WriteError(w, http.StatusBadRequest, "parentId must be a node id or null", "", nil)
			return
		}
		if parentID, ok = parseOptionalUUID(w, raw); !ok {
			return
		}
	}
	position := 0
	if req.Position != nil {
		position = *req.Position
	}

	var n *store.RoadmapNode
	if err == nil {
		n, err = s.store.MoveRoadmapNode(r.Context(), userID, nid, parentID, position)
	}
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "node not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the project owner, a workspace admin, or the owner of this branch can change this node", "", nil)
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
