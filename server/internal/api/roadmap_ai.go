package api

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// RoadmapPlanner is the subset of the agent planner the API layer needs,
// kept as an interface so api has no LLM dependency (mirrors the Coach seam).
type RoadmapPlanner interface {
	PlanMVP(ctx context.Context, prdTitle string, sections map[string]string) (*store.RoadmapPlanNode, error)
	ExpandNode(ctx context.Context, projectTitle, nodeTitle, nodeDesc string) ([]store.RoadmapPlanNode, error)
}

func (s *Server) registerRoadmapAIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/prds/{pid}/build", s.requireAuth(s.handleBuildFromPrd))
	mux.HandleFunc("POST /api/v1/roadmap/{nid}/expand", s.requireAuth(s.handleExpandNode))
}

type buildFromPrdRequest struct {
	ExpectedCount *int `json:"expectedCount"`
}

// handleBuildFromPrd turns a refined PRD into a project with a generated
// branched roadmap — the "build the MVP" step that carries an idea past the
// PRD into execution.
func (s *Server) handleBuildFromPrd(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	prdID, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	if s.planner == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "roadmap planner is not configured", "", nil)
		return
	}

	// The body is optional: the first build call omits it, and a confirm retry
	// sends the live count the user agreed to replace.
	var req buildFromPrdRequest
	if err := httpx.DecodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}

	prd, err := s.store.GetPrdForMember(r.Context(), prdID, userID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "prd not found", "", nil)
		return
	}
	if err != nil {
		s.log.Error("get prd", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	project, err := s.store.EnsureProjectForPrd(r.Context(), userID, prdID)
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	if err != nil {
		s.log.Error("ensure project for prd", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	tree, err := s.planner.PlanMVP(r.Context(), prd.Title, prd.Sections)
	if err != nil {
		s.log.Error("plan mvp", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	count, err := s.store.ReplaceRoadmapTree(r.Context(), userID, project.ID, req.ExpectedCount, *tree)
	var rc *store.ReplaceConflictError
	if errors.As(err, &rc) {
		httpx.WriteError(w, http.StatusConflict, "roadmap has existing nodes; confirm replacement", "replace_conflict", map[string]interface{}{"liveCount": rc.LiveCount})
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	if err != nil {
		s.log.Error("replace roadmap tree", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"projectId": project.ID.String(), "nodeCount": count})
}

// handleExpandNode breaks one roadmap node into AI-proposed subtasks, growing
// the tree where the user is actually working.
func (s *Server) handleExpandNode(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	nid, ok := pathUUID(w, r, "nid")
	if !ok {
		return
	}
	if s.planner == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "roadmap planner is not configured", "", nil)
		return
	}

	node, err := s.store.GetRoadmapNode(r.Context(), userID, nid)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "node not found", "", nil)
		return
	}
	if err != nil {
		s.log.Error("get roadmap node", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	project, err := s.store.GetProjectForMember(r.Context(), node.ProjectID, userID)
	if err != nil {
		s.log.Error("get project for expand", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	children, err := s.planner.ExpandNode(r.Context(), project.Title, node.Title, node.Description)
	if err != nil {
		s.log.Error("expand node", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	created, err := s.store.ExpandRoadmapNode(r.Context(), userID, nid, children)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "node not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	if err != nil {
		s.log.Error("expand roadmap node", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	views := make([]roadmapNodeView, len(created))
	for i, n := range created {
		views[i] = roadmapNodeResponse(n)
	}
	httpx.WriteJSON(w, http.StatusCreated, views)
}
