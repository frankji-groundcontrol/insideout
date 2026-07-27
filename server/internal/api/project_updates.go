package api

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

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

// maxUpdateContentRunes bounds a project update's content — a progress note,
// blocker, or comment is a few paragraphs at most. The 1 MiB request-body cap
// is a transport ceiling, not a per-field bound; this check turns an overlong
// field into a clean 400 instead of storing a multi-MB blob.
const maxUpdateContentRunes = 5000

// validateUpdateContent trims raw and returns the cleaned content, or an error
// message if it's empty or overlong. Shared by add and edit so the bound is
// enforced in exactly one place.
func validateUpdateContent(raw string) (string, string) {
	content := strings.TrimSpace(raw)
	if content == "" {
		return "", "content is required"
	}
	if utf8.RuneCountInString(content) > maxUpdateContentRunes {
		return "", "content too long"
	}
	return content, ""
}

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
	content, errMsg := validateUpdateContent(req.Content)
	if errMsg != "" {
		httpx.WriteError(w, http.StatusBadRequest, errMsg, "", nil)
		return
	}

	u, err := s.store.AddProjectUpdate(r.Context(), userID, pid, req.Kind, content)
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
	content, errMsg := validateUpdateContent(req.Content)
	if errMsg != "" {
		httpx.WriteError(w, http.StatusBadRequest, errMsg, "", nil)
		return
	}

	u, err := s.store.UpdateProjectUpdate(r.Context(), userID, uid, content)
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
