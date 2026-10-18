package api

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

func (s *Server) registerConversationRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/conversations/{id}", s.requireAuth(s.handleGetConversation))
	mux.HandleFunc("GET /api/v1/conversations/{id}/messages", s.requireAuth(s.handleListConversationMessages))
	mux.HandleFunc("POST /api/v1/conversations/{id}/messages", s.requireAuth(s.handleSendConversationMessage))
	mux.HandleFunc("POST /api/v1/conversations/{id}/stage", s.requireAuth(s.handleConfirmConversationStage))
	mux.HandleFunc("POST /api/v1/prds/{id}/conversation", s.requireAuth(s.handleStartPrdConversation))
}

type conversationView struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	PrdID       string `json:"prdId"`
	Stage       string `json:"stage"`
	// PendingStage is the proposal waiting for the person's confirmation —
	// the client renders it as proposed, never as current (USER-01).
	PendingStage string `json:"pendingStage,omitempty"`
	Status       string `json:"status"`
	// Locale is the language the coach speaks here ('en' / 'zh'), chosen
	// at creation; the agent package selects its prompts from it.
	Locale    string `json:"locale"`
	UpdatedAt string `json:"updatedAt"`
	// CriticSkipped is why a critic pass did not run, when one was skipped
	// (contention, critic-output-failure); CriticRounds is how many did. A skip
	// opens the finalize gate, so the confirm card reads these to say that no
	// review ran rather than imply one (rv4-AIDEV-02).
	CriticSkipped string `json:"criticSkipped,omitempty"`
	CriticRounds  int    `json:"criticRounds"`
}

func conversationResponse(c store.AgentConversation) conversationView {
	view := conversationView{
		ID: c.ID.String(), WorkspaceID: c.WorkspaceID.String(), PrdID: c.PrdID.String(),
		Stage: c.Stage, Status: c.Status, Locale: c.Locale, UpdatedAt: c.UpdatedAt.Format(timeLayout),
	}
	applyConversationMeta(&view, c.Meta)
	return view
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

type startConversationRequest struct {
	// Locale is the app locale the client coaches in ('en-US', 'zh-CN',
	// ...); the store canonicalizes it. Absent stays zh — the language
	// every pre-locale conversation already coached in.
	Locale string `json:"locale"`
}

// handleStartPrdConversation starts the caller's own coach conversation on a
// PRD, or returns the one they already have. GET /prds/{id}/conversation only
// ever finds the caller's (conversations are owner-only) and ConvertIdea makes
// just the converter's, so this is how any other author or admin gets one.
func (s *Server) handleStartPrdConversation(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req startConversationRequest
	if err := httpx.DecodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	conv, created, err := s.store.EnsureConversationForPrd(r.Context(), userID, pid, req.Locale)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "PRD not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the author or a workspace admin can coach this PRD", "", nil)
		return
	}
	if err != nil {
		s.log.Error("start prd conversation", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpx.WriteJSON(w, status, conversationResponse(*conv))
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
		// A tool-call-only assistant turn is persisted with empty Content (the
		// tool mechanics live in tool_calls, not shown). Rendering it would
		// surface an empty chat bubble, so skip it here — the same filter
		// loadHistory applies when rebuilding model context (coach.go) (F22).
		if m.Role == "assistant" && m.Content == "" {
			continue
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

type confirmStageRequest struct {
	Stage string `json:"stage"`
}

// handleConfirmConversationStage writes a stage the coach proposed and
// the person accepted. Declining never calls this, so the row stays.
func (s *Server) handleConfirmConversationStage(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	cid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req confirmStageRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.Stage = strings.TrimSpace(req.Stage)
	if req.Stage == "" {
		httpx.WriteError(w, http.StatusBadRequest, "stage is required", "", nil)
		return
	}
	err := s.coach.ConfirmStage(r.Context(), userID, cid, req.Stage)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusConflict, "that stage is not waiting for confirmation", "", nil)
		return
	}
	if err != nil {
		s.log.Error("confirm conversation stage", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	conv, err := s.store.GetConversationForOwner(r.Context(), cid, userID)
	if err != nil {
		s.log.Error("get conversation after confirm", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, conversationResponse(*conv))
}
