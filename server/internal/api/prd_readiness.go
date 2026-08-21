package api

import (
	"errors"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/readiness"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// handlePrdReadiness discloses per-audience gaps for "form a version
// now" (PRODUCT.md): audience-specific readiness with explained gaps,
// never a completeness score, never a Commit blocker. carryIntoCommit
// is the suggested unresolved list for the Commit the user can make
// regardless.
func (s *Server) handlePrdReadiness(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "id")
	if !ok {
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
		s.log.Error("prd readiness", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"prdId":     p.ID,
		"audiences": readiness.Assess(p.Sections),
	})
}
