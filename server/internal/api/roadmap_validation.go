package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// FEAT-04 (review 3): the human half of the evidence pipeline. The
// webhook kinds record what the repo did; these two routes record what
// people saw — an observation on a leaf, then the confirm with a reason
// that the outcome held.

type addObservationRequest struct {
	Detail     string `json:"detail"`
	SourceURL  string `json:"sourceUrl"`
	ObservedAt string `json:"observedAt"`
}

// handleAddObservation records one human-entered validation observation
// (POST /api/v1/roadmap/{nid}/evidence): what was observed, its source,
// and the date it was seen. Typed by a person — nothing here invites
// users or collects metrics.
func (s *Server) handleAddObservation(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	nid, ok := pathUUID(w, r, "nid")
	if !ok {
		return
	}
	var req addObservationRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.Detail = strings.TrimSpace(req.Detail)
	if req.Detail == "" || len(req.Detail) > 2000 {
		httpx.WriteError(w, http.StatusBadRequest, "detail is required (max 2000 characters)", "", nil)
		return
	}
	req.SourceURL = strings.TrimSpace(req.SourceURL)
	if len(req.SourceURL) > 2000 {
		httpx.WriteError(w, http.StatusBadRequest, "sourceUrl is too long (max 2000 characters)", "", nil)
		return
	}
	observedAt, err := time.Parse("2006-01-02", req.ObservedAt)
	if err != nil || observedAt.Year() < 1970 || observedAt.Year() > 9999 {
		httpx.WriteError(w, http.StatusBadRequest, "observedAt must be a date (YYYY-MM-DD)", "", nil)
		return
	}
	err = s.store.AddValidationObservation(r.Context(), userID, nid, req.Detail, req.SourceURL, observedAt)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "node not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the project owner or a workspace admin can record evidence", "", nil)
		return
	}
	if errors.Is(err, store.ErrConflict) {
		httpx.WriteError(w, http.StatusConflict, "observations attach to leaf nodes", "", nil)
		return
	}
	if err != nil {
		s.log.Error("add observation", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]bool{"ok": true})
}

type validateNodeRequest struct {
	Reason string `json:"reason"`
}

// handleValidateNode is the human confirm into 'validated' (POST
// /api/v1/roadmap/{nid}/validate): a done node plus the reason it counts
// as validated. The plain status PATCH cannot make this move — it carries
// no reason, so it is not a confirm.
func (s *Server) handleValidateNode(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	nid, ok := pathUUID(w, r, "nid")
	if !ok {
		return
	}
	var req validateNodeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" || len(req.Reason) > 2000 {
		httpx.WriteError(w, http.StatusBadRequest, "reason is required (max 2000 characters)", "", nil)
		return
	}
	n, err := s.store.ValidateRoadmapNode(r.Context(), userID, nid, req.Reason)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "node not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the project owner or a workspace admin can validate", "", nil)
		return
	}
	if errors.Is(err, store.ErrInvalidTransition) {
		httpx.WriteError(w, http.StatusBadRequest, err.Error(), "", nil)
		return
	}
	if err != nil {
		s.log.Error("validate node", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, roadmapNodeResponse(*n, s.now()))
}
