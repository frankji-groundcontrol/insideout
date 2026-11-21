package api

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/audienceview"
	"github.com/frankji-groundcontrol/insideout/server/internal/export"
	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

func (s *Server) registerPrdRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/prds/{id}/commit", s.requireAuth(s.handleCommitPrd))
	mux.HandleFunc("GET /api/v1/prds/{id}/commits", s.requireAuth(s.handleListPrdCommits))
	mux.HandleFunc("GET /api/v1/prds/{id}/readiness", s.requireAuth(s.handlePrdReadiness))
	mux.HandleFunc("GET /api/v1/prds/{id}/view", s.requireAuth(s.handlePrdView))
	mux.HandleFunc("GET /api/v1/workspaces/{id}/prds", s.requireAuth(s.handleListWorkspacePrds))
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
	if err != nil {
		s.writeStoreError(w, "get prd", err, errMsgs{store.ErrNotFound: "PRD not found"})
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
	// Title is optional: a section-only save omits it (JSON null / absent) and
	// leaves the stored title untouched. Only a payload that deliberately
	// carries a title changes it — so a section save can't clobber a title
	// someone else edited concurrently.
	Title    *string           `json:"title"`
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
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" || tooLong(trimmed, 200) {
			httpx.WriteError(w, http.StatusBadRequest, "title must be 1-200 characters when provided", "", nil)
			return
		}
		req.Title = &trimmed
	}
	for key := range req.Sections {
		if !validSectionKey[key] {
			httpx.WriteError(w, http.StatusBadRequest, "unknown section key: "+key, "", nil)
			return
		}
	}

	prd, err := s.store.UpdateSections(r.Context(), userID, pid, req.Title, req.Sections, nil) // manual save: no CAS, the human always wins
	// The store owns the section cap (rv4-FR-F127); its message is the 400.
	var capErr *store.SectionTooLongError
	if errors.As(err, &capErr) {
		httpx.WriteError(w, http.StatusBadRequest, capErr.Error(), "", nil)
		return
	}
	if err != nil {
		s.writeStoreError(w, "update prd sections", err, errMsgs{
			store.ErrNotFound:  "PRD not found",
			store.ErrForbidden: "only the author or a workspace admin can edit this PRD",
		})
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
	// PRD read pre-check, kept on purpose: an unreadable PRD is a 404 before the list.
	if _, err := s.store.GetPrdForMember(r.Context(), pid, userID); errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "PRD not found", "", nil)
		return
	} else if err != nil {
		s.writeStoreError(w, "get prd", err, nil)
		return
	}

	revs, err := s.store.ListRevisions(r.Context(), userID, pid)
	if err != nil {
		s.writeStoreError(w, "list revisions", err, nil)
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
	// note is optional and a body-less POST is fine (io.EOF) — but a genuinely
	// malformed body is a client error, not a silent no-op (F15).
	var req createRevisionRequest
	if err := httpx.DecodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	if req.Note != nil && refuseTooLong(w, "note", *req.Note) {
		return
	}

	rev, err := s.store.CreateRevision(r.Context(), userID, pid, req.Note)
	if err != nil {
		// ErrConflict is the MAX+1 race: a concurrent snapshot already took
		// this revision number (F14). The winner's row stands; the loser
		// refreshes and retries.
		s.writeStoreError(w, "create revision", err, errMsgs{
			store.ErrNotFound:  "PRD not found",
			store.ErrForbidden: "only the author can snapshot a revision",
			store.ErrConflict:  "a revision was just created — refresh and try again",
		})
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
	if err != nil {
		s.writeStoreError(w, "update prd status", err, errMsgs{
			store.ErrNotFound:   "PRD not found",
			store.ErrForbidden:  "you don't have permission for this transition",
			store.ErrValidation: "invalid status transition",
		})
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
	if err != nil {
		s.writeStoreError(w, "get prd for export", err, errMsgs{store.ErrNotFound: "PRD not found"})
		return
	}

	audience := r.URL.Query().Get("audience")
	if audience != "" && !audienceview.Valid(audience) {
		httpx.WriteError(w, http.StatusBadRequest, "audience must be one of decision, management, delivery, validation", "", nil)
		return
	}

	if format == "markdown" {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		if audience != "" {
			proj, _ := audienceview.Get(audience)
			w.Header().Set("Content-Disposition", "attachment; filename=\"prd-"+audience+"-view.md\"")
			_, _ = w.Write([]byte(export.MarkdownAudience(prd.Title, prd.Sections, proj)))
			return
		}
		w.Header().Set("Content-Disposition", "attachment; filename=\"prd.md\"")
		_, _ = w.Write([]byte(export.Markdown(prd.Title, prd.Sections)))
		return
	}
	if audience != "" {
		httpx.WriteError(w, http.StatusBadRequest, "audience projections export as markdown (format=markdown) for now", "", nil)
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
	// PRD read pre-check, kept on purpose: an unreadable PRD is a 404 before the list.
	if _, err := s.store.GetPrdForMember(r.Context(), pid, userID); errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "PRD not found", "", nil)
		return
	} else if err != nil {
		s.writeStoreError(w, "get prd", err, nil)
		return
	}

	conv, err := s.store.GetLatestConversationForPrd(r.Context(), pid, userID)
	if err != nil {
		s.writeStoreError(w, "get prd conversation", err, errMsgs{store.ErrNotFound: "no coach conversation for this PRD"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, conversationResponse(*conv))
}
