package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

func (s *Server) registerWorkspaceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/workspaces", s.requireAuth(s.handleListWorkspaces))
	mux.HandleFunc("POST /api/v1/workspaces", s.requireAuth(s.handleCreateWorkspace))
	mux.HandleFunc("POST /api/v1/workspaces/join", s.requireAuth(s.handleJoinWorkspace))
	mux.HandleFunc("GET /api/v1/workspaces/{id}", s.requireAuth(s.handleGetWorkspace))
	mux.HandleFunc("PATCH /api/v1/workspaces/{id}", s.requireAuth(s.handleUpdateWorkspace))
	mux.HandleFunc("DELETE /api/v1/workspaces/{id}", s.requireAuth(s.handleDeleteWorkspace))
	mux.HandleFunc("GET /api/v1/workspaces/{id}/members", s.requireAuth(s.handleListMembers))
	mux.HandleFunc("PATCH /api/v1/workspaces/{id}/members/{userId}", s.requireAuth(s.handleUpdateMemberRole))
	mux.HandleFunc("DELETE /api/v1/workspaces/{id}/members/{userId}", s.requireAuth(s.handleRemoveMember))
}

type workspaceView struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	CoverURL    *string `json:"coverUrl"`
	Code        string  `json:"code"`
	Status      string  `json:"status"`
	MemberCount int     `json:"memberCount"`
	MyRole      string  `json:"myRole"`
	CreatedAt   string  `json:"createdAt"`
}

func workspaceResponse(w store.WorkspaceSummary) workspaceView {
	return workspaceView{
		ID: w.ID.String(), Title: w.Title, Description: w.Description, CoverURL: w.CoverURL,
		Code: w.Code, Status: w.Status, MemberCount: w.MemberCount, MyRole: w.MyRole,
		CreatedAt: w.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (s *Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	list, err := s.store.ListWorkspacesForUser(r.Context(), userID)
	if err != nil {
		s.log.Error("list workspaces", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	views := make([]workspaceView, len(list))
	for i, ws := range list {
		views[i] = workspaceResponse(ws)
	}
	httpx.WriteJSON(w, http.StatusOK, views)
}

type createWorkspaceRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (s *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	var req createWorkspaceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || len(req.Title) > 200 {
		httpx.WriteError(w, http.StatusBadRequest, "title is required (max 200 characters)", "", nil)
		return
	}

	ws, err := s.store.CreateWorkspace(r.Context(), userID, req.Title, req.Description)
	if err != nil {
		s.log.Error("create workspace", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	summary, err := s.store.GetWorkspaceForMember(r.Context(), ws.ID, userID)
	if err != nil {
		s.log.Error("get created workspace", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, workspaceResponse(*summary))
}

type joinWorkspaceRequest struct {
	Code string `json:"code"`
}

func (s *Server) handleJoinWorkspace(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	var req joinWorkspaceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		httpx.WriteError(w, http.StatusBadRequest, "code is required", "", nil)
		return
	}

	ws, err := s.store.JoinWorkspace(r.Context(), userID, code)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "no active workspace with this code", "", nil)
		return
	}
	if errors.Is(err, store.ErrConflict) {
		httpx.WriteError(w, http.StatusConflict, "already a member of this workspace", "", nil)
		return
	}
	if err != nil {
		s.log.Error("join workspace", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	summary, err := s.store.GetWorkspaceForMember(r.Context(), ws.ID, userID)
	if err != nil {
		s.log.Error("get joined workspace", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, workspaceResponse(*summary))
}

func (s *Server) handleGetWorkspace(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	wsID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}

	ws, err := s.store.GetWorkspaceForMember(r.Context(), wsID, userID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "workspace not found", "", nil)
		return
	}
	if err != nil {
		s.log.Error("get workspace", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, workspaceResponse(*ws))
}

type updateWorkspaceRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	CoverURL    *string `json:"coverUrl"`
}

func (s *Server) handleUpdateWorkspace(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	wsID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req updateWorkspaceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || len(req.Title) > 200 {
		httpx.WriteError(w, http.StatusBadRequest, "title is required (max 200 characters)", "", nil)
		return
	}

	_, err := s.store.UpdateWorkspace(r.Context(), userID, wsID, req.Title, req.Description, req.CoverURL)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "workspace not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the workspace admin or creator can do this", "", nil)
		return
	}
	if err != nil {
		s.log.Error("update workspace", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	summary, err := s.store.GetWorkspaceForMember(r.Context(), wsID, userID)
	if err != nil {
		s.log.Error("get updated workspace", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, workspaceResponse(*summary))
}

func (s *Server) handleDeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	wsID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	err := s.store.DeleteWorkspace(r.Context(), userID, wsID)
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the workspace creator can delete it", "", nil)
		return
	}
	if err != nil {
		s.log.Error("delete workspace", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
