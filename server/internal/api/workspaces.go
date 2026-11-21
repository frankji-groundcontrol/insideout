package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
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
	// CanEdit and CanDelete mirror UpdateWorkspace's and DeleteWorkspace's
	// checks for the viewer, so a client hides what would be refused (rv5-FE-05).
	CanEdit   bool   `json:"canEdit"`
	CanDelete bool   `json:"canDelete"`
	CreatedAt string `json:"createdAt"`
}

// workspaceResponse is w as viewerID sees it.
func workspaceResponse(w store.WorkspaceSummary, viewerID uuid.UUID) workspaceView {
	creator := w.CreatorID == viewerID
	return workspaceView{
		ID: w.ID.String(), Title: w.Title, Description: w.Description, CoverURL: w.CoverURL,
		Code: w.Code, Status: w.Status, MemberCount: w.MemberCount, MyRole: w.MyRole,
		CanEdit: creator || w.MyRole == "admin", CanDelete: creator,
		CreatedAt: w.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (s *Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	list, err := s.store.ListWorkspacesForUser(r.Context(), userID)
	if err != nil {
		s.writeStoreError(w, "list workspaces", err, nil)
		return
	}
	views := make([]workspaceView, len(list))
	for i, ws := range list {
		views[i] = workspaceResponse(ws, userID)
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
	if req.Title == "" || tooLong(req.Title, 200) {
		httpx.WriteError(w, http.StatusBadRequest, "title is required (max 200 characters)", "", nil)
		return
	}

	ws, err := s.store.CreateWorkspace(r.Context(), userID, req.Title, req.Description)
	if err != nil {
		s.writeStoreError(w, "create workspace", err, nil)
		return
	}

	summary, err := s.store.GetWorkspaceForMember(r.Context(), ws.ID, userID)
	if err != nil {
		s.writeStoreError(w, "get created workspace", err, nil)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, workspaceResponse(*summary, userID))
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
	if err != nil {
		s.writeStoreError(w, "join workspace", err, errMsgs{
			store.ErrNotFound: "no active workspace with this code",
			store.ErrConflict: "already a member of this workspace",
		})
		return
	}

	summary, err := s.store.GetWorkspaceForMember(r.Context(), ws.ID, userID)
	if err != nil {
		s.writeStoreError(w, "get joined workspace", err, nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, workspaceResponse(*summary, userID))
}

func (s *Server) handleGetWorkspace(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	wsID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}

	ws, err := s.store.GetWorkspaceForMember(r.Context(), wsID, userID)
	if err != nil {
		s.writeStoreError(w, "get workspace", err, errMsgs{store.ErrNotFound: "workspace not found"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, workspaceResponse(*ws, userID))
}

type updateWorkspaceRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	// CoverURL: absent keeps the stored cover; null or "" clears it.
	CoverURL json.RawMessage `json:"coverUrl"`
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
	if req.Title == "" || tooLong(req.Title, 200) {
		httpx.WriteError(w, http.StatusBadRequest, "title is required (max 200 characters)", "", nil)
		return
	}
	cover, ok := storedURLPatch(w, "coverUrl", req.CoverURL)
	if !ok {
		return
	}

	_, err := s.store.UpdateWorkspace(r.Context(), userID, wsID, req.Title, req.Description, cover)
	if err != nil {
		s.writeStoreError(w, "update workspace", err, errMsgs{
			store.ErrNotFound:  "workspace not found",
			store.ErrForbidden: "only the workspace admin or creator can do this",
		})
		return
	}

	summary, err := s.store.GetWorkspaceForMember(r.Context(), wsID, userID)
	if err != nil {
		s.writeStoreError(w, "get updated workspace", err, nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, workspaceResponse(*summary, userID))
}

func (s *Server) handleDeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	wsID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	err := s.store.DeleteWorkspace(r.Context(), userID, wsID)
	if err != nil {
		s.writeStoreError(w, "delete workspace", err, errMsgs{store.ErrForbidden: "only the workspace creator can delete it"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
