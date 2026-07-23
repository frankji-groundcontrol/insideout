package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/export"
	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

func (s *Server) registerPrdRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/prds/{id}", s.requireAuth(s.handleGetPrd))
	mux.HandleFunc("PATCH /api/v1/prds/{id}", s.requireAuth(s.handleUpdatePrd))
	mux.HandleFunc("GET /api/v1/prds/{id}/revisions", s.requireAuth(s.handleListRevisions))
	mux.HandleFunc("POST /api/v1/prds/{id}/revisions", s.requireAuth(s.handleCreateRevision))
	mux.HandleFunc("POST /api/v1/prds/{id}/status", s.requireAuth(s.handleUpdatePrdStatus))
	mux.HandleFunc("GET /api/v1/prds/{id}/export", s.requireAuth(s.handleExportPrd))
	mux.HandleFunc("GET /api/v1/prds/{id}/conversation", s.requireAuth(s.handleGetPrdConversation))
}

type prdView struct {
	ID              string            `json:"id"`
	WorkspaceID     string            `json:"workspaceId"`
	IdeaID          *string           `json:"ideaId,omitempty"`
	ProjectID       *string           `json:"projectId,omitempty"`
	AuthorID        string            `json:"authorId"`
	Title           string            `json:"title"`
	Sections        map[string]string `json:"sections"`
	Status          string            `json:"status"`
	CurrentRevision int               `json:"currentRevision"`
	UpdatedAt       string            `json:"updatedAt"`
}

func prdResponse(p store.Prd) prdView {
	v := prdView{
		ID: p.ID.String(), WorkspaceID: p.WorkspaceID.String(), AuthorID: p.AuthorID.String(),
		Title: p.Title, Sections: p.Sections, Status: p.Status, CurrentRevision: p.CurrentRevision,
		UpdatedAt: p.UpdatedAt.Format(timeLayout),
	}
	if p.IdeaID != nil {
		id := p.IdeaID.String()
		v.IdeaID = &id
	}
	if p.ProjectID != nil {
		id := p.ProjectID.String()
		v.ProjectID = &id
	}
	return v
}

func (s *Server) handleGetPrd(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	prd, err := s.store.GetPrdForMember(r.Context(), pid, userID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "PRD not found", "", nil)
		return
	}
	if err != nil {
		s.log.Error("get prd", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, prdResponse(*prd))
}

var validSectionKey = func() map[string]bool {
	m := make(map[string]bool, len(store.PrdSectionKeys))
	for _, k := range store.PrdSectionKeys {
		m[k] = true
	}
	return m
}()

type updatePrdRequest struct {
	Title    string            `json:"title"`
	Sections map[string]string `json:"sections"`
}

func (s *Server) handleUpdatePrd(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req updatePrdRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || len(req.Title) > 200 {
		httpx.WriteError(w, http.StatusBadRequest, "title is required (max 200 characters)", "", nil)
		return
	}
	for key := range req.Sections {
		if !validSectionKey[key] {
			httpx.WriteError(w, http.StatusBadRequest, "unknown section key: "+key, "", nil)
			return
		}
	}

	prd, err := s.store.UpdateSections(r.Context(), userID, pid, req.Title, req.Sections, nil) // manual save: no CAS, the human always wins
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "PRD not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the author or a workspace admin can edit this PRD", "", nil)
		return
	}
	if err != nil {
		s.log.Error("update prd sections", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, prdResponse(*prd))
}

type revisionView struct {
	ID        string            `json:"id"`
	Revision  int               `json:"revision"`
	Sections  map[string]string `json:"sections"`
	CreatedBy string            `json:"createdBy"`
	Note      *string           `json:"note,omitempty"`
	CreatedAt string            `json:"createdAt"`
}

func revisionResponse(rev store.PrdRevision) revisionView {
	return revisionView{
		ID: rev.ID.String(), Revision: rev.Revision, Sections: rev.Sections,
		CreatedBy: rev.CreatedBy.String(), Note: rev.Note, CreatedAt: rev.CreatedAt.Format(timeLayout),
	}
}

func (s *Server) handleListRevisions(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	if _, err := s.store.GetPrdForMember(r.Context(), pid, userID); errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "PRD not found", "", nil)
		return
	} else if err != nil {
		s.log.Error("get prd", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	revs, err := s.store.ListRevisions(r.Context(), userID, pid)
	if err != nil {
		s.log.Error("list revisions", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	views := make([]revisionView, len(revs))
	for i, rev := range revs {
		views[i] = revisionResponse(rev)
	}
	httpx.WriteJSON(w, http.StatusOK, views)
}

type createRevisionRequest struct {
	Note *string `json:"note"`
}

func (s *Server) handleCreateRevision(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req createRevisionRequest
	_ = httpx.DecodeJSON(r, &req) // note is optional; a body-less POST is fine

	rev, err := s.store.CreateRevision(r.Context(), userID, pid, req.Note)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "PRD not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the author can snapshot a revision", "", nil)
		return
	}
	if err != nil {
		s.log.Error("create revision", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, revisionResponse(*rev))
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

var validPrdStatuses = map[string]bool{"draft": true, "reviewing": true, "approved": true, "rejected": true}

func (s *Server) handleUpdatePrdStatus(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req updateStatusRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	if !validPrdStatuses[req.Status] {
		httpx.WriteError(w, http.StatusBadRequest, "invalid status", "", nil)
		return
	}

	prd, err := s.store.UpdatePrdStatus(r.Context(), userID, pid, req.Status)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "PRD not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you don't have permission for this transition", "", nil)
		return
	}
	if errors.Is(err, store.ErrValidation) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid status transition", "", nil)
		return
	}
	if err != nil {
		s.log.Error("update prd status", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, prdResponse(*prd))
}

func (s *Server) handleExportPrd(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	format := r.URL.Query().Get("format")
	if format != "markdown" && format != "print" {
		httpx.WriteError(w, http.StatusBadRequest, "format must be \"markdown\" or \"print\"", "", nil)
		return
	}

	prd, err := s.store.GetPrdForMember(r.Context(), pid, userID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "PRD not found", "", nil)
		return
	}
	if err != nil {
		s.log.Error("get prd for export", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	if format == "markdown" {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=\"prd.md\"")
		_, _ = w.Write([]byte(export.Markdown(prd.Title, prd.Sections)))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(export.PrintHTML(prd.Title, prd.Sections)))
}

// handleGetPrdConversation lets the PRD workspace page resume its coach
// conversation on revisit, not just right after idea conversion.
func (s *Server) handleGetPrdConversation(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	if _, err := s.store.GetPrdForMember(r.Context(), pid, userID); errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "PRD not found", "", nil)
		return
	} else if err != nil {
		s.log.Error("get prd", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	conv, err := s.store.GetLatestConversationForPrd(r.Context(), pid, userID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "no coach conversation for this PRD", "", nil)
		return
	}
	if err != nil {
		s.log.Error("get prd conversation", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, conversationResponse(*conv))
}
