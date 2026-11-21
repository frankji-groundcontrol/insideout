package api

import (
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
	if req.Detail == "" || tooLong(req.Detail, 2000) {
		httpx.WriteError(w, http.StatusBadRequest, "detail is required (max 2000 characters)", "", nil)
		return
	}
	req.SourceURL = strings.TrimSpace(req.SourceURL)
	if tooLong(req.SourceURL, 2000) {
		httpx.WriteError(w, http.StatusBadRequest, "sourceUrl is too long (max 2000 characters)", "", nil)
		return
	}
	observedAt, err := time.Parse("2006-01-02", req.ObservedAt)
	if err != nil || observedAt.Year() < 1970 || observedAt.Year() > 9999 {
		httpx.WriteError(w, http.StatusBadRequest, "observedAt must be a date (YYYY-MM-DD)", "", nil)
		return
	}
	err = s.store.AddValidationObservation(r.Context(), userID, nid, req.Detail, req.SourceURL, observedAt)
	if err != nil {
		s.writeStoreError(w, "add observation", err, errMsgs{
			store.ErrNotFound:  "node not found",
			store.ErrForbidden: "only the project owner, a workspace admin, or the owner of this branch can record evidence here",
			store.ErrConflict:  "observations attach to leaf nodes",
		})
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
	if req.Reason == "" || tooLong(req.Reason, 2000) {
		httpx.WriteError(w, http.StatusBadRequest, "reason is required (max 2000 characters)", "", nil)
		return
	}
	n, err := s.store.ValidateRoadmapNode(r.Context(), userID, nid, req.Reason)
	if err != nil {
		s.writeStoreError(w, "validate node", err, errMsgs{
			store.ErrNotFound:  "node not found",
			store.ErrForbidden: "only the project owner or a workspace admin can validate",
		})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, roadmapNodeResponse(*n, s.now()))
}
