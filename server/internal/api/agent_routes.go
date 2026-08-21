package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/agentcontext"
	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// Agent vocabulary v1 (docs/plans/2026-08-22-agent-vocabulary.md):
// context is compact and focus-scoped; checkpoint and propose write
// typed timeline records. Agents never apply strategic changes.
func (s *Server) registerAgentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/agent/context", s.requireAuth(s.handleAgentContext))
	mux.HandleFunc("POST /api/v1/agent/checkpoint", s.requireAuth(s.handleAgentCheckpoint))
	mux.HandleFunc("POST /api/v1/agent/propose", s.requireAuth(s.handleAgentPropose))
}

func (s *Server) handleAgentContext(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := parseUUIDParam(w, "project_id", r.URL.Query().Get("project_id"))
	if !ok {
		return
	}
	mode := r.URL.Query().Get("mode")
	switch mode {
	case "brainstorming", "implementation", "review":
	default:
		mode = "implementation"
	}
	focus := r.URL.Query().Get("focus")

	project, err := s.store.GetProjectForMember(r.Context(), pid, userID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "project not found", "", nil)
		return
	}
	if err != nil {
		s.log.Error("agent context: project", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	nodes, err := s.store.ListRoadmap(r.Context(), userID, pid)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		s.log.Error("agent context: roadmap", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	evidence, err := s.store.EvidenceCountsByProject(r.Context(), userID, pid)
	if err != nil {
		s.log.Error("agent context: evidence", "error", err)
		evidence = map[string]int{}
	}

	in := agentcontext.Inputs{
		ProjectTitle:   project.Title,
		ProjectID:      pid.String(),
		Mode:           mode,
		FocusNodeID:    focus,
		Nodes:          make([]agentcontext.NodeIn, 0, len(nodes)),
		EvidenceCounts: evidence,
	}
	for _, n := range nodes {
		parent := ""
		if n.ParentID != nil {
			parent = n.ParentID.String()
		}
		in.Nodes = append(in.Nodes, agentcontext.NodeIn{ID: n.ID.String(), ParentID: parent, Title: n.Title, Status: n.Status})
	}
	if project.ID != uuid.Nil {
		if prd, err := s.store.PrdByProject(r.Context(), userID, pid); err == nil && prd != nil {
			in.PrdTitle = prd.Title
			in.PrdSections = prd.Sections
			if commits, err := s.store.ListPrdCommits(r.Context(), userID, prd.ID); err == nil && len(commits) > 0 {
				in.LatestCommit = &agentcontext.CommitIn{
					Revision: commits[0].Revision, Name: commits[0].Name, Audience: commits[0].PrimaryAudience,
				}
			}
		}
	}
	httpx.WriteJSON(w, http.StatusOK, agentcontext.Assemble(in))
}

func (s *Server) handleAgentCheckpoint(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	var req struct {
		ProjectID string `json:"projectId"`
		NodeID    string `json:"nodeId"`
		Summary   string `json:"summary"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.Summary = strings.TrimSpace(req.Summary)
	if req.Summary == "" || len(req.Summary) > 2000 {
		httpx.WriteError(w, http.StatusBadRequest, "summary is required (max 2000 characters)", "", nil)
		return
	}
	pid, ok := parseUUIDParam(w, "projectId", req.ProjectID)
	if !ok {
		return
	}
	content := "[agent checkpoint] " + req.Summary
	if req.NodeID != "" {
		content += " (node " + req.NodeID + ")"
	}
	u, err := s.store.AddProjectUpdate(r.Context(), userID, pid, "agent_checkpoint", content)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "project not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	if err != nil {
		s.log.Error("agent checkpoint", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": u.ID, "kind": u.Kind, "recorded": true})
}

func (s *Server) handleAgentPropose(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	var req struct {
		ProjectID string `json:"projectId"`
		Kind      string `json:"kind"`
		Summary   string `json:"summary"`
		Detail    string `json:"detail"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	switch req.Kind {
	case "structure", "scope", "priority":
	default:
		httpx.WriteError(w, http.StatusBadRequest, "kind must be structure, scope, or priority", "", nil)
		return
	}
	req.Summary = strings.TrimSpace(req.Summary)
	if req.Summary == "" || len(req.Summary) > 2000 {
		httpx.WriteError(w, http.StatusBadRequest, "summary is required (max 2000 characters)", "", nil)
		return
	}
	pid, ok := parseUUIDParam(w, "projectId", req.ProjectID)
	if !ok {
		return
	}
	content := "[agent proposal/" + req.Kind + "] " + req.Summary
	if d := strings.TrimSpace(req.Detail); d != "" {
		content += "\n" + d
	}
	u, err := s.store.AddProjectUpdate(r.Context(), userID, pid, "agent_proposal", content)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "project not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	if err != nil {
		s.log.Error("agent propose", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": u.ID, "kind": u.Kind, "proposed": true, "accepted": false})
}
