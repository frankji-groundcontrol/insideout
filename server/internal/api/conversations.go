package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

func (s *Server) registerConversationRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/conversations/{id}", s.requireAuth(s.handleGetConversation))
	mux.HandleFunc("GET /api/v1/conversations/{id}/messages", s.requireAuth(s.handleListConversationMessages))
	mux.HandleFunc("POST /api/v1/conversations/{id}/messages", s.requireAuth(s.handleSendConversationMessage))
}

type conversationView struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	PrdID       string `json:"prdId"`
	Stage       string `json:"stage"`
	Status      string `json:"status"`
	UpdatedAt   string `json:"updatedAt"`
}

func conversationResponse(c store.AgentConversation) conversationView {
	return conversationView{
		ID: c.ID.String(), WorkspaceID: c.WorkspaceID.String(), PrdID: c.PrdID.String(),
		Stage: c.Stage, Status: c.Status, UpdatedAt: c.UpdatedAt.Format(timeLayout),
	}
}

func (s *Server) handleGetConversation(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	cid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	conv, err := s.store.GetConversationForOwner(r.Context(), cid, userID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "conversation not found", "", nil)
		return
	}
	if err != nil {
		s.log.Error("get conversation", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, conversationResponse(*conv))
}

type messageView struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

func (s *Server) handleListConversationMessages(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	cid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	if _, err := s.store.GetConversationForOwner(r.Context(), cid, userID); errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "conversation not found", "", nil)
		return
	} else if err != nil {
		s.log.Error("get conversation", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	msgs, err := s.store.ListAgentMessages(r.Context(), userID, cid)
	if err != nil {
		s.log.Error("list messages", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	views := make([]messageView, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == "tool" {
			continue // internal tool-call bookkeeping, not shown in the chat UI / 内部工具调用记录，不在聊天界面展示
		}
		views = append(views, messageView{ID: m.ID.String(), Role: m.Role, Content: m.Content, CreatedAt: m.CreatedAt.Format(timeLayout)})
	}
	httpx.WriteJSON(w, http.StatusOK, views)
}

type sendMessageRequest struct {
	Content string `json:"content"`
}

// handleSendConversationMessage validates ownership and the message body,
// then hands off to the Coach implementation to stream the SSE reply —
// the actual agent loop lives in internal/agent (P4), kept out of the api
// package so api has no LLM-library dependency.
func (s *Server) handleSendConversationMessage(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	cid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}

	conv, err := s.store.GetConversationForOwner(r.Context(), cid, userID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "conversation not found", "", nil)
		return
	}
	if err != nil {
		s.log.Error("get conversation", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	if conv.Status != "active" {
		httpx.WriteError(w, http.StatusConflict, "this conversation has ended", "", nil)
		return
	}

	var req sendMessageRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		httpx.WriteError(w, http.StatusBadRequest, "content is required", "", nil)
		return
	}
	if len(req.Content) > 10_000 {
		httpx.WriteError(w, http.StatusBadRequest, "message too long", "", map[string]interface{}{"max_length": 10000})
		return
	}

	if s.coach == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "coach is not configured", "", nil)
		return
	}
	s.coach.HandleMessage(w, r, cid.String(), userID.String(), conv.WorkspaceID.String(), req.Content)
}
