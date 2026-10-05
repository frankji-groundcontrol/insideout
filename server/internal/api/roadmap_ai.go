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
	mux.HandleFunc("POST /api/v1/prds/{pid}/build/confirm", s.requireAuth(s.handleConfirmBuildFromPrd))
	mux.HandleFunc("POST /api/v1/roadmap/{nid}/expand", s.requireAuth(s.handleExpandNode))
	mux.HandleFunc("POST /api/v1/roadmap/{nid}/expand/confirm", s.requireAuth(s.handleConfirmExpandNode))
}

// handleBuildFromPrd drafts a branched roadmap from a refined PRD. It does
// not write the tree; the caller confirms with handleConfirmBuildFromPrd.
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

	built, ok := s.loadBuildablePrd(w, r, userID, prdID)
	if !ok {
		return
	}

	tree, err := s.planner.PlanMVP(r.Context(), built.prd.Title, built.prd.Sections)
	if err != nil {
		s.log.Error("plan mvp", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"projectId": built.project.ID.String(),
		"tree":      planNodeResponse(*tree),
	})
}

type confirmBuildRequest struct {
	ExpectedCount *int         `json:"expectedCount"`
	Tree          planNodeView `json:"tree"`
}

// handleConfirmBuildFromPrd writes a tree the caller already drafted.
func (s *Server) handleConfirmBuildFromPrd(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	prdID, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}

	var req confirmBuildRequest
	if err := httpx.DecodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	if req.Tree.Title == "" {
		httpx.WriteError(w, http.StatusBadRequest, "a drafted tree is required", "", nil)
		return
	}

	built, ok := s.loadBuildablePrd(w, r, userID, prdID)
	if !ok {
		return
	}

	count, err := s.store.ReplaceRoadmapTree(r.Context(), userID, built.project.ID, req.ExpectedCount, planNodeFromView(req.Tree))
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

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"projectId": built.project.ID.String(), "nodeCount": count})
}

type expandDraftView struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// handleExpandNode drafts subtasks for one roadmap node. It does not write
// them; the caller confirms with handleConfirmExpandNode.
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

	views := make([]expandDraftView, len(children))
	for i, c := range children {
		views[i] = expandDraftView{Title: c.Title, Description: c.Description}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"children": views})
}

type confirmExpandRequest struct {
	Children []expandDraftView `json:"children"`
}

// handleConfirmExpandNode writes subtasks the caller already drafted.
func (s *Server) handleConfirmExpandNode(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	nid, ok := pathUUID(w, r, "nid")
	if !ok {
		return
	}

	var req confirmExpandRequest
	if err := httpx.DecodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	if len(req.Children) == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "drafted subtasks are required", "", nil)
		return
	}

	children := make([]store.RoadmapPlanNode, len(req.Children))
	for i, c := range req.Children {
		if c.Title == "" {
			httpx.WriteError(w, http.StatusBadRequest, "each drafted subtask needs a title", "", nil)
			return
		}
		children[i] = store.RoadmapPlanNode{Title: c.Title, Description: c.Description}
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
