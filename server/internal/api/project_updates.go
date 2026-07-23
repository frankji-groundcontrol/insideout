package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

type projectUpdateView struct {
	ID        string `json:"id"`
	ProjectID string `json:"projectId"`
	AuthorID  string `json:"authorId"`
	Kind      string `json:"kind"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

func projectUpdateResponse(u store.ProjectUpdate) projectUpdateView {
	return projectUpdateView{
		ID: u.ID.String(), ProjectID: u.ProjectID.String(), AuthorID: u.AuthorID.String(),
		Kind: u.Kind, Content: u.Content, CreatedAt: u.CreatedAt.Format(timeLayout),
	}
}

var validUpdateKinds = map[string]bool{"progress": true, "blocker": true, "note": true}

type addProjectUpdateRequest struct {
	Kind    string `json:"kind"`
	Content string `json:"content"`
}

func (s *Server) handleAddProjectUpdate(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	var req addProjectUpdateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	if !validUpdateKinds[req.Kind] {
		httpx.WriteError(w, http.StatusBadRequest, "kind must be \"progress\", \"blocker\", or \"note\"", "", nil)
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		httpx.WriteError(w, http.StatusBadRequest, "content is required", "", nil)
		return
	}

	u, err := s.store.AddProjectUpdate(r.Context(), userID, pid, req.Kind, req.Content)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "project not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	if err != nil {
		s.log.Error("add project update", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, projectUpdateResponse(*u))
}

type editProjectUpdateRequest struct {
	Content string `json:"content"`
}

func (s *Server) handleEditProjectUpdate(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	uid, ok := pathUUID(w, r, "uid")
	if !ok {
		return
	}
	var req editProjectUpdateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		httpx.WriteError(w, http.StatusBadRequest, "content is required", "", nil)
		return
	}

	u, err := s.store.UpdateProjectUpdate(r.Context(), userID, uid, req.Content)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "update not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the author or a workspace admin can edit this", "", nil)
		return
	}
	if err != nil {
		s.log.Error("edit project update", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, projectUpdateResponse(*u))
}

func (s *Server) handleDeleteProjectUpdate(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	uid, ok := pathUUID(w, r, "uid")
	if !ok {
		return
	}
	err := s.store.DeleteProjectUpdate(r.Context(), userID, uid)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "update not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the author or a workspace admin can delete this", "", nil)
		return
	}
	if err != nil {
		s.log.Error("delete project update", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
