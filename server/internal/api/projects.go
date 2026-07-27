package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

func (s *Server) registerProjectRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/workspaces/{id}/projects", s.requireAuth(s.handleListProjects))
	mux.HandleFunc("POST /api/v1/workspaces/{id}/projects", s.requireAuth(s.handleCreateProject))
	mux.HandleFunc("GET /api/v1/projects/{pid}", s.requireAuth(s.handleGetProject))
	mux.HandleFunc("PATCH /api/v1/projects/{pid}", s.requireAuth(s.handleUpdateProject))
	mux.HandleFunc("DELETE /api/v1/projects/{pid}", s.requireAuth(s.handleDeleteProject))
	mux.HandleFunc("POST /api/v1/projects/{pid}/updates", s.requireAuth(s.handleAddProjectUpdate))
	mux.HandleFunc("PATCH /api/v1/updates/{uid}", s.requireAuth(s.handleEditProjectUpdate))
	mux.HandleFunc("DELETE /api/v1/updates/{uid}", s.requireAuth(s.handleDeleteProjectUpdate))
}

type projectView struct {
	ID                  string  `json:"id"`
	WorkspaceID         string  `json:"workspaceId"`
	Title               string  `json:"title"`
	Description         string  `json:"description"`
	OwnerID             *string `json:"ownerId"`
	Status              string  `json:"status"`
	RepoURL             string  `json:"repoUrl"`
	CreatedAt           string  `json:"createdAt"`
	LatestUpdateKind    *string `json:"latestUpdateKind,omitempty"`
	LatestUpdateContent *string `json:"latestUpdateContent,omitempty"`
	LatestUpdateAt      *string `json:"latestUpdateAt,omitempty"`
}

func projectResponse(p store.ProjectWithLatest) projectView {
	v := projectView{
		ID: p.ID.String(), WorkspaceID: p.WorkspaceID.String(), Title: p.Title, Description: p.Description,
		Status: p.Status, RepoURL: p.RepoURL, CreatedAt: p.CreatedAt.Format(timeLayout),
		LatestUpdateKind: p.LatestUpdateKind, LatestUpdateContent: p.LatestUpdateContent,
	}
	if p.OwnerID != nil {
		id := p.OwnerID.String()
		v.OwnerID = &id
	}
	if p.LatestUpdateAt != nil {
		t := p.LatestUpdateAt.Format(timeLayout)
		v.LatestUpdateAt = &t
	}
	return v
}

const timeLayout = "2006-01-02T15:04:05Z07:00"

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	wsID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	if _, err := s.store.GetMembership(r.Context(), wsID, userID); errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "workspace not found", "", nil)
		return
	} else if err != nil {
		s.log.Error("get membership", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	list, err := s.store.ListProjectsForWorkspace(r.Context(), userID, wsID)
	if err != nil {
		s.log.Error("list projects", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	views := make([]projectView, len(list))
	for i, p := range list {
		views[i] = projectResponse(p)
	}
	httpx.WriteJSON(w, http.StatusOK, views)
}

type createProjectRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	wsID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req createProjectRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || len(req.Title) > 200 {
		httpx.WriteError(w, http.StatusBadRequest, "title is required (max 200 characters)", "", nil)
		return
	}

	p, err := s.store.CreateProject(r.Context(), userID, wsID, req.Title, req.Description)
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	if err != nil {
		s.log.Error("create project", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, projectResponse(store.ProjectWithLatest{Project: *p}))
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	p, err := s.store.GetProjectForMember(r.Context(), pid, userID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "project not found", "", nil)
		return
	}
	if err != nil {
		s.log.Error("get project", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	updates, err := s.store.ListProjectUpdates(r.Context(), userID, pid, store.ProjectUpdatesPageSize, nil)
	if err != nil {
		s.log.Error("list project updates", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	// Embed only the newest page; a full page signals there's older history,
	// surfaced as nextCursor for a future load-more (frontend follow-up).
	var nextCursor *string
	if len(updates) == store.ProjectUpdatesPageSize {
		last := updates[len(updates)-1].ID.String()
		nextCursor = &last
	}

	resp := struct {
		projectView
		Updates    []projectUpdateView `json:"updates"`
		NextCursor *string             `json:"nextCursor"`
	}{projectView: projectResponse(store.ProjectWithLatest{Project: *p}), Updates: make([]projectUpdateView, len(updates)), NextCursor: nextCursor}
	for i, u := range updates {
		resp.Updates[i] = projectUpdateResponse(u)
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

type updateProjectRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	OwnerID     *string `json:"ownerId"`
}

var validProjectStatuses = map[string]bool{"planning": true, "active": true, "paused": true, "done": true, "archived": true}

func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	var req updateProjectRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || len(req.Title) > 200 {
		httpx.WriteError(w, http.StatusBadRequest, "title is required (max 200 characters)", "", nil)
		return
	}
	if !validProjectStatuses[req.Status] {
		httpx.WriteError(w, http.StatusBadRequest, "invalid status", "", nil)
		return
	}
	ownerID, ok := parseOptionalUUID(w, req.OwnerID)
	if !ok {
		return
	}

	p, err := s.store.UpdateProject(r.Context(), userID, pid, store.ProjectUpdateFields{
		Title: req.Title, Description: req.Description, Status: req.Status, OwnerID: ownerID,
	})
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "project not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the project owner or a workspace admin can do this", "", nil)
		return
	}
	if err != nil {
		s.log.Error("update project", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, projectResponse(store.ProjectWithLatest{Project: *p}))
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	err := s.store.DeleteProject(r.Context(), userID, pid)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "project not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only a workspace admin can delete projects", "", nil)
		return
	}
	if err != nil {
		s.log.Error("delete project", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func parseOptionalUUID(w http.ResponseWriter, raw *string) (*uuid.UUID, bool) {
	if raw == nil || *raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(*raw)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid ownerId", "", nil)
		return nil, false
	}
	return &id, true
}
