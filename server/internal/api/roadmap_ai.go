package api

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// RoadmapPlanner is the subset of the agent planner the API layer needs,
// kept as an interface so api has no LLM dependency (mirrors the Coach seam).
// The locale ('en'/'zh') is the coaching conversation's stored language —
// the planner drafts its roadmap in it.
type RoadmapPlanner interface {
	PlanMVP(ctx context.Context, locale, prdTitle string, sections map[string]string) (*store.RoadmapPlanNode, error)
	ExpandNode(ctx context.Context, locale, projectTitle, nodeTitle, nodeDesc string) ([]store.RoadmapPlanNode, error)
}

// AIGate is the optional gate in front of an LLM call. The real coach
// implements it, so the roadmap planner draws on the same rate-limit, circuit
// breaker and permit budget as coach turns. Admit writes its own refusal and
// returns ok=false; otherwise the caller calls done once with the call's error.
// A Coach without it (a test double, none at all) leaves those calls ungated,
// which is why this is a separate interface and not part of Coach.
type AIGate interface {
	Admit(w http.ResponseWriter, r *http.Request, workspaceID, userID uuid.UUID, label string) (done func(error), ok bool)
}

// templateExpandDraft is the mark an ExpandNode error carries when the draft
// came from the planner's built-in template because the provider failed: the
// draft is served as usual, but Admit's done still hears the error so the
// circuit breaker counts the outage. Found structurally, like AIGate.
type templateExpandDraft interface {
	error
	TemplateDraft() []store.RoadmapPlanNode
}

// admitAI gates one planner call through the coach when it can.
func (s *Server) admitAI(w http.ResponseWriter, r *http.Request, workspaceID, userID uuid.UUID, label string) (done func(error), ok bool) {
	if g, gated := s.coach.(AIGate); gated {
		return g.Admit(w, r, workspaceID, userID, label)
	}
	return func(error) {}, true
}

// errPlannerUnwound is what done hears when a planner call never returned: a
// panic in the planner unwinds past the handler's flow, so only the deferred
// done runs and fails the run with it. It is a bug in our process, not a
// provider-health signal, so it never feeds the circuit breaker.
var errPlannerUnwound = errors.New("planner call did not return")

// conversationLocale is the language the caller coaches this PRD in: the
// locale their own conversation on it was created with. No conversation
// (nobody coached it yet) stays zh, the prompts' long-standing language.
func (s *Server) conversationLocale(ctx context.Context, userID, prdID uuid.UUID) string {
	conv, err := s.store.GetLatestConversationForPrd(ctx, prdID, userID)
	if err != nil || conv.Locale == "" {
		return store.LocaleZH
	}
	return conv.Locale
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

	done, ok := s.admitAI(w, r, built.project.WorkspaceID, userID, "roadmap:plan-mvp")
	if !ok {
		return
	}
	// The deferred done is the only one that runs when the planner panics
	// (done is idempotent): the permit comes back and the run fails with
	// errPlannerUnwound instead of staying "running".
	planErr := errPlannerUnwound
	defer func() { done(planErr) }()
	tree, err := s.planner.PlanMVP(r.Context(), s.conversationLocale(r.Context(), userID, prdID), built.prd.Title, built.prd.Sections)
	planErr = err
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
		httpx.WriteError(w, http.StatusForbidden, "only the project owner or a workspace admin can change the roadmap", "", nil)
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

	done, ok := s.admitAI(w, r, project.WorkspaceID, userID, "roadmap:expand")
	if !ok {
		return
	}
	// Same deferred-done shape as the build handler: a panicking planner
	// still releases the permit and fails the run.
	expandErr := errPlannerUnwound
	defer func() { done(expandErr) }()
	// A node's language follows the PRD its project was built from: the
	// conversation that coached that PRD chose it. A project with no PRD
	// behind it (never built) has no conversation, so zh — the planner's
	// long-standing only language.
	locale := store.LocaleZH
	if prdID, err := s.store.PrdIDForProject(r.Context(), userID, project.ID); err == nil {
		locale = s.conversationLocale(r.Context(), userID, prdID)
	}
	children, err := s.planner.ExpandNode(r.Context(), locale, project.Title, node.Title, node.Description)
	expandErr = err
	// A fallback-served expand answers with the planner's built-in template
	// but still reports the provider error: done must count the failure for
	// the breaker, and the HTTP answer stays the template.
	var fromTemplate templateExpandDraft
	if errors.As(err, &fromTemplate) {
		children = fromTemplate.TemplateDraft()
	}
	if err != nil && fromTemplate == nil {
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
	// The caps every other node write has, counted in characters (COMP-01): the
	// planner returns at most 40 subtasks, a node title is at most 200.
	const maxChildren, maxTitle, maxDescription = 40, 200, 2000
	if len(req.Children) > maxChildren {
		httpx.WriteError(w, http.StatusBadRequest, "too many drafted subtasks (max 40)", "", nil)
		return
	}

	children := make([]store.RoadmapPlanNode, len(req.Children))
	for i, c := range req.Children {
		if c.Title == "" {
			httpx.WriteError(w, http.StatusBadRequest, "each drafted subtask needs a title", "", nil)
			return
		}
		if tooLong(c.Title, maxTitle) {
			httpx.WriteError(w, http.StatusBadRequest, "a drafted subtask title is too long (max 200 characters)", "", nil)
			return
		}
		if tooLong(c.Description, maxDescription) {
			httpx.WriteError(w, http.StatusBadRequest, "a drafted subtask description is too long (max 2000 characters)", "", nil)
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
		httpx.WriteError(w, http.StatusForbidden, "only the project owner or a workspace admin can change the roadmap", "", nil)
		return
	}
	if err != nil {
		s.log.Error("expand roadmap node", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	now := s.now()
	views := make([]roadmapNodeView, len(created))
	for i, n := range created {
		views[i] = roadmapNodeResponse(n, now)
	}
	httpx.WriteJSON(w, http.StatusCreated, views)
}
