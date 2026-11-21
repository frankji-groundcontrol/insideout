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

// loadBuildablePrd loads the PRD a build reads and refuses one that is not
// approved (rv5-PARB-01): REST, CLI, MCP and web all meet the admin's review
// gate here. It writes nothing; only the confirm creates a project.
func (s *Server) loadBuildablePrd(w http.ResponseWriter, r *http.Request, userID, prdID uuid.UUID) (*store.Prd, bool) {
	prd, err := s.store.GetPrdForMember(r.Context(), prdID, userID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "prd not found", "", nil)
		return nil, false
	}
	if err != nil {
		s.log.Error("get prd", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return nil, false
	}
	if prd.Status != "approved" {
		httpx.WriteError(w, http.StatusConflict, "only an approved PRD can be built into a roadmap", "prd_not_approved", nil)
		return nil, false
	}
	return prd, true
}

// ensureBuildProject returns the project a confirmed build writes into,
// creating and linking it on the first build, which only the PRD's author or
// a workspace admin may do (rv5-FEAT-01).
func (s *Server) ensureBuildProject(w http.ResponseWriter, r *http.Request, userID, prdID uuid.UUID) (*store.Project, bool) {
	project, err := s.store.EnsureProjectForPrd(r.Context(), userID, prdID)
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the PRD's author or a workspace admin can build it", "", nil)
		return nil, false
	}
	if err != nil {
		s.log.Error("ensure project for prd", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return nil, false
	}
	return project, true
}
