package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

func (s *Server) registerIdeaRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/workspaces/{id}/ideas", s.requireAuth(s.handleListIdeas))
	mux.HandleFunc("POST /api/v1/workspaces/{id}/ideas", s.requireAuth(s.handleCreateIdea))
	mux.HandleFunc("GET /api/v1/ideas/{iid}", s.requireAuth(s.handleGetIdea))
	mux.HandleFunc("PATCH /api/v1/ideas/{iid}", s.requireAuth(s.handleUpdateIdea))
	mux.HandleFunc("DELETE /api/v1/ideas/{iid}", s.requireAuth(s.handleDropIdea))
	mux.HandleFunc("POST /api/v1/ideas/{iid}/convert", s.requireAuth(s.handleConvertIdea))
}

type ideaView struct {
	ID          string  `json:"id"`
	WorkspaceID string  `json:"workspaceId"`
	AuthorID    string  `json:"authorId"`
	Title       string  `json:"title"`
	Content     string  `json:"content"`
	Status      string  `json:"status"`
	PrdID       *string `json:"prdId,omitempty"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

func ideaResponse(i store.Idea) ideaView {
	v := ideaView{
		ID: i.ID.String(), WorkspaceID: i.WorkspaceID.String(), AuthorID: i.AuthorID.String(),
		Title: i.Title, Content: i.Content, Status: i.Status,
		CreatedAt: i.CreatedAt.Format(timeLayout), UpdatedAt: i.UpdatedAt.Format(timeLayout),
	}
	if i.PrdID != nil {
		id := i.PrdID.String()
		v.PrdID = &id
	}
	return v
}

func (s *Server) handleListIdeas(w http.ResponseWriter, r *http.Request) {
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

	ideas, err := s.store.ListIdeas(r.Context(), userID, wsID)
	if err != nil {
		s.log.Error("list ideas", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	views := make([]ideaView, len(ideas))
	for i, idea := range ideas {
		views[i] = ideaResponse(idea)
	}
	httpx.WriteJSON(w, http.StatusOK, views)
}

type createIdeaRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func (s *Server) handleCreateIdea(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	wsID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req createIdeaRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || len(req.Title) > 200 {
		httpx.WriteError(w, http.StatusBadRequest, "title is required (max 200 characters)", "", nil)
		return
	}

	idea, err := s.store.CreateIdea(r.Context(), userID, wsID, req.Title, req.Content)
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	if err != nil {
		s.log.Error("create idea", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, ideaResponse(*idea))
}

func (s *Server) handleGetIdea(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	iid, ok := pathUUID(w, r, "iid")
	if !ok {
		return
	}
	idea, err := s.store.GetIdeaForMember(r.Context(), iid, userID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "idea not found", "", nil)
		return
	}
	if err != nil {
		s.log.Error("get idea", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ideaResponse(*idea))
}

type updateIdeaRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func (s *Server) handleUpdateIdea(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	iid, ok := pathUUID(w, r, "iid")
	if !ok {
		return
	}
	var req updateIdeaRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || len(req.Title) > 200 {
		httpx.WriteError(w, http.StatusBadRequest, "title is required (max 200 characters)", "", nil)
		return
	}

	idea, err := s.store.UpdateIdea(r.Context(), userID, iid, req.Title, req.Content)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "idea not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the author can edit this idea", "", nil)
		return
	}
	if err != nil {
		s.log.Error("update idea", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ideaResponse(*idea))
}

func (s *Server) handleDropIdea(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	iid, ok := pathUUID(w, r, "iid")
	if !ok {
		return
	}
	err := s.store.DropIdea(r.Context(), userID, iid)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "idea not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the author or a workspace admin can drop this idea", "", nil)
		return
	}
	if err != nil {
		s.log.Error("drop idea", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleConvertIdea(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	iid, ok := pathUUID(w, r, "iid")
	if !ok {
		return
	}
	prd, conv, err := s.store.ConvertIdea(r.Context(), userID, iid)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "idea not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the author can convert this idea", "", nil)
		return
	}
	if errors.Is(err, store.ErrConflict) {
		httpx.WriteError(w, http.StatusConflict, "this idea was already converted", "", nil)
		return
	}
	if err != nil {
		s.log.Error("convert idea", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]string{
		"prdId":          prd.ID.String(),
		"conversationId": conv.ID.String(),
	})
}
