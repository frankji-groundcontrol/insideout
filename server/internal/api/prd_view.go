package api

import (
	"errors"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/audienceview"
	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/readiness"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// handlePrdView returns an audience's projection of the PRD core: the
// ordered section picks with whys, that audience's readiness gaps, and
// the latest committed version for context.
func (s *Server) handlePrdView(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	audience := r.URL.Query().Get("audience")
	if !audienceview.Valid(audience) {
		httpx.WriteError(w, http.StatusBadRequest, "audience must be one of decision, management, delivery, validation", "", nil)
		return
	}
	proj, _ := audienceview.Get(audience)
	p, err := s.store.GetPrdForMember(r.Context(), pid, userID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "PRD not found", "", nil)
		return
	}
	if err != nil {
		s.log.Error("prd view", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	commits, err := s.store.ListPrdCommits(r.Context(), userID, pid)
	if err != nil {
		s.log.Error("prd view: commits", "error", err)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"prdId":        p.ID,
		"title":        p.Title,
		"projection":   proj,
		"readiness":    readiness.Assess(p.Sections)[audience],
		"latestCommit": firstCommit(commits),
	})
}

func firstCommit(commits []store.PrdCommit) *store.PrdCommit {
	if len(commits) == 0 {
		return nil
	}
	return &commits[0]
}
