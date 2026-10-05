package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/readiness"
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
	p, err := s.store.GetPrdForMember(r.Context(), pid, userID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "PRD not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	if err != nil {
		s.log.Error("prd commit readiness", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	unresolved := mustJSONStringArray(mergeCarriedGaps(req.Unresolved, p.Sections, req.Audience))

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

// mergeCarriedGaps keeps the gaps readiness says this commit carries.
// A caller may name extra unresolved items; it cannot drop a gap the
// audience still has open.
func mergeCarriedGaps(given []string, sections map[string]string, audience string) []string {
	ar := readiness.Assess(sections)[audience]
	out := make([]string, 0, len(given)+len(ar.CarryIntoCommit))
	seen := map[string]bool{}
	for _, item := range append(append([]string{}, given...), ar.CarryIntoCommit...) {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
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
