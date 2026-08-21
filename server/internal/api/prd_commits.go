package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// validAudiences mirrors PRODUCT.md's audience views.
var validAudiences = map[string]bool{
	"decision": true, "management": true, "delivery": true, "validation": true,
}

// handleCommitPrd is the human Commit (PRODUCT.md): freeze the working
// version as an immutable named version.
func (s *Server) handleCommitPrd(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		Name         string   `json:"name"`
		Audience     string   `json:"primaryAudience"`
		Summary      string   `json:"changeSummary"`
		Unresolved   []string `json:"unresolved"`
		DecisionNote string   `json:"decisionNote"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > 200 {
		httpx.WriteError(w, http.StatusBadRequest, "name is required (max 200 characters)", "", nil)
		return
	}
	if !validAudiences[req.Audience] {
		httpx.WriteError(w, http.StatusBadRequest, "primaryAudience must be one of decision, management, delivery, validation", "", nil)
		return
	}
	unresolved := mustJSONStringArray(req.Unresolved)

	c, err := s.store.CreatePrdCommit(r.Context(), userID, pid, req.Name, req.Audience, req.Summary, unresolved, req.DecisionNote)
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the PRD author (Driver) or a workspace admin can commit", "", nil)
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "PRD not found", "", nil)
		return
	}
	if err != nil {
		s.log.Error("prd commit", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, c)
}

// handleListPrdCommits returns the version history, newest first.
func (s *Server) handleListPrdCommits(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	rows, err := s.store.ListPrdCommits(r.Context(), userID, pid)
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "PRD not found", "", nil)
		return
	}
	if err != nil {
		s.log.Error("prd commits list", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	if rows == nil {
		rows = []store.PrdCommit{}
	}
	httpx.WriteJSON(w, http.StatusOK, rows)
}

func mustJSONStringArray(v []string) json.RawMessage {
	if len(v) == 0 {
		return json.RawMessage("[]")
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("[]")
	}
	return raw
}
