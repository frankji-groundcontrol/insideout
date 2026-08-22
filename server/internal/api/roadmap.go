package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/roadmaptime"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

func (s *Server) registerRoadmapRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/projects/{pid}/roadmap", s.requireAuth(s.handleListRoadmap))
	mux.HandleFunc("POST /api/v1/projects/{pid}/roadmap", s.requireAuth(s.handleCreateRoadmapNode))
	mux.HandleFunc("PATCH /api/v1/roadmap/{nid}", s.requireAuth(s.handleUpdateRoadmapNode))
	mux.HandleFunc("POST /api/v1/roadmap/{nid}/move", s.requireAuth(s.handleMoveRoadmapNode))
	mux.HandleFunc("DELETE /api/v1/roadmap/{nid}", s.requireAuth(s.handleDeleteRoadmapNode))
	mux.HandleFunc("GET /api/v1/projects/{pid}/guide", s.requireAuth(s.handleProjectGuide))
	mux.HandleFunc("GET /api/v1/projects/{pid}/roadmap/progress", s.requireAuth(s.handleRoadmapProgress))
	mux.HandleFunc("GET /api/v1/roadmap/{nid}/evidence", s.requireAuth(s.handleListEvidence))
}

type roadmapNodeView struct {
	ID          string  `json:"id"`
	ProjectID   string  `json:"projectId"`
	ParentID    *string `json:"parentId"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Position    int     `json:"position"`
	Deadline    *string `json:"deadline,omitempty"`
	Pressure    string  `json:"pressure,omitempty"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
	// B3 attribution (D10): display names, not ids — the card shows the last
	// editor's initial plus a "created by X · edited by Y" tooltip. Omitted when
	// unknown (pre-migration rows, or a removed author → ON DELETE SET NULL).
	CreatorName *string `json:"creatorName,omitempty"`
	EditorName  *string `json:"editorName,omitempty"`
}

func roadmapNodeResponse(n store.RoadmapNode) roadmapNodeView {
	v := roadmapNodeView{
		ID: n.ID.String(), ProjectID: n.ProjectID.String(), Title: n.Title,
		Description: n.Description, Status: n.Status, Position: n.Position,
		CreatedAt: n.CreatedAt.Format(timeLayout), UpdatedAt: n.UpdatedAt.Format(timeLayout),
	}
	if n.ParentID != nil {
		id := n.ParentID.String()
		v.ParentID = &id
	}
	if n.Deadline != nil {
		dl := n.Deadline.Format(time.RFC3339)
		v.Deadline = &dl
		v.Pressure = roadmaptime.Pressure(*n.Deadline, time.Now())
	}
	v.CreatorName = n.CreatorName
	v.EditorName = n.EditorName
	return v
}

var validRoadmapStatuses = map[string]bool{"locked": true, "pending": true, "in_progress": true, "done": true}

func (s *Server) handleListRoadmap(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	nodes, err := s.store.ListRoadmap(r.Context(), userID, pid)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "project not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	if err != nil {
		s.log.Error("list roadmap", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	views := make([]roadmapNodeView, len(nodes))
	for i, n := range nodes {
		views[i] = roadmapNodeResponse(n)
	}
	httpx.WriteJSON(w, http.StatusOK, views)
}

type createRoadmapNodeRequest struct {
	ParentID    *string `json:"parentId"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
}

func (s *Server) handleCreateRoadmapNode(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	var req createRoadmapNodeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || len(req.Title) > 200 {
		httpx.WriteError(w, http.StatusBadRequest, "title is required (max 200 characters)", "", nil)
		return
	}
	parentID, ok := parseOptionalUUID(w, req.ParentID)
	if !ok {
		return
	}

	n, err := s.store.CreateRoadmapNode(r.Context(), userID, pid, parentID, req.Title, req.Description)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "project or parent node not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	if errors.Is(err, store.ErrConflict) {
		httpx.WriteError(w, http.StatusConflict, "parent node belongs to a different project", "", nil)
		return
	}
	if err != nil {
		s.log.Error("create roadmap node", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, roadmapNodeResponse(*n))
}

type updateRoadmapNodeRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	// Deadline: RFC3339 sets it, "" clears it, absent leaves it.
	Deadline *string `json:"deadline"`
}

func (s *Server) handleUpdateRoadmapNode(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	nid, ok := pathUUID(w, r, "nid")
	if !ok {
		return
	}
	var req updateRoadmapNodeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	// Partial update: at least one field must be present. Only present fields
	// are validated and written; absent fields are left untouched (D1/D9).
	if req.Title == nil && req.Description == nil && req.Status == nil && req.Deadline == nil {
		httpx.WriteError(w, http.StatusBadRequest, "at least one of title, description, status, or deadline is required", "", nil)
		return
	}
	if req.Title != nil {
		*req.Title = strings.TrimSpace(*req.Title)
		if *req.Title == "" || len(*req.Title) > 200 {
			httpx.WriteError(w, http.StatusBadRequest, "title is required (max 200 characters)", "", nil)
			return
		}
	}
	if req.Status != nil && !validRoadmapStatuses[*req.Status] {
		httpx.WriteError(w, http.StatusBadRequest, "invalid status", "", nil)
		return
	}

	fields := store.RoadmapNodeFields{Title: req.Title, Description: req.Description, Status: req.Status}
	if req.Deadline != nil {
		if *req.Deadline == "" {
			fields.ClearDeadline = true
		} else {
			dl, err := time.Parse(time.RFC3339, *req.Deadline)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, "deadline must be RFC3339 (or \"\" to clear)", "", nil)
				return
			}
			fields.Deadline = &dl
		}
	}
	n, err := s.store.UpdateRoadmapNode(r.Context(), userID, nid, fields)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "node not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	if err != nil {
		s.log.Error("update roadmap node", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, roadmapNodeResponse(*n))
}

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
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
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
	httpx.WriteJSON(w, http.StatusOK, roadmapNodeResponse(*n))
}

func (s *Server) handleDeleteRoadmapNode(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	nid, ok := pathUUID(w, r, "nid")
	if !ok {
		return
	}
	err := s.store.DeleteRoadmapNode(r.Context(), userID, nid)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "node not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	if err != nil {
		s.log.Error("delete roadmap node", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
