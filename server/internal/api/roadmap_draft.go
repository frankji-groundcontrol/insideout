package api

import (
	"errors"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// planNodeView is a drafted roadmap tree the client can show and send back
// unchanged. Nothing here is a stored node.
type planNodeView struct {
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Children    []planNodeView `json:"children,omitempty"`
}

func planNodeResponse(n store.RoadmapPlanNode) planNodeView {
	v := planNodeView{Title: n.Title, Description: n.Description}
	if len(n.Children) > 0 {
		v.Children = make([]planNodeView, len(n.Children))
		for i, c := range n.Children {
			v.Children[i] = planNodeResponse(c)
		}
	}
	return v
}

func planNodeFromView(v planNodeView) store.RoadmapPlanNode {
	n := store.RoadmapPlanNode{Title: v.Title, Description: v.Description}
	if len(v.Children) > 0 {
		n.Children = make([]store.RoadmapPlanNode, len(v.Children))
		for i, c := range v.Children {
			n.Children[i] = planNodeFromView(c)
		}
	}
	return n
}

// buildablePrd is the PRD being built and the project it builds into.
// ok is false when the handler has already written the error.
type buildablePrd struct {
	prd     *store.Prd
	project *store.Project
}

// loadBuildablePrd loads the PRD and ensures its project, the same lookups
// HEAD's handleBuildFromPrd did before drafting. It does not gate on PRD
// status or a commit: that check was not part of the draft/confirm split.
func (s *Server) loadBuildablePrd(w http.ResponseWriter, r *http.Request, userID, prdID uuid.UUID) (buildablePrd, bool) {
	prd, err := s.store.GetPrdForMember(r.Context(), prdID, userID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "prd not found", "", nil)
		return buildablePrd{}, false
	}
	if err != nil {
		s.log.Error("get prd", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return buildablePrd{}, false
	}

	project, err := s.store.EnsureProjectForPrd(r.Context(), userID, prdID)
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return buildablePrd{}, false
	}
	if err != nil {
		s.log.Error("ensure project for prd", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return buildablePrd{}, false
	}
	return buildablePrd{prd: prd, project: project}, true
}
