package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/agentcontext"
	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
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
	mux.HandleFunc("POST /api/v1/agent/proposals/{uid}/decision", s.requireAuth(s.handleProposalDecision))
	mux.HandleFunc("POST /api/v1/agent/token", s.requireAuth(s.handleAgentToken))
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
	if u, err := uuid.Parse(focus); err == nil {
		focus = u.String() // the form node ids are compared in
	}
	// A root-scoped agent token is briefed on its root by default, when the
	// root is a node of this project (rv5-TREEB-05).
	defaulted := false
	if root := RootOf(r.Context()); focus == "" && root != nil {
		focus, defaulted = root.String(), true
	}

	project, err := s.store.GetProjectForMember(r.Context(), pid, userID)
	if err != nil {
		s.writeStoreError(w, "agent context: project", err, errMsgs{store.ErrNotFound: "project not found"})
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
	focusFound := false
	for _, n := range nodes {
		parent := ""
		if n.ParentID != nil {
			parent = n.ParentID.String()
		}
		in.Nodes = append(in.Nodes, agentcontext.NodeIn{
			ID: n.ID.String(), ParentID: parent, Title: n.Title, Description: n.Description, Status: n.Status,
			Deadline: n.Deadline, Acceptance: n.Acceptance, OwnerName: n.OwnerName,
		})
		if n.ID.String() == focus {
			focusFound = true
		}
	}
	// An unknown focus is refused, not ignored: an agent asked for one
	// subtree and must not get the whole project back (rv5-TREEX-05).
	if defaulted && !focusFound {
		focus, in.FocusNodeID = "", ""
	}
	if focus != "" && !focusFound {
		httpx.WriteError(w, http.StatusNotFound, "focus node not found in this project", "", nil)
		return
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
	if req.Summary == "" || tooLong(req.Summary, 2000) {
		httpx.WriteError(w, http.StatusBadRequest, "summary is required (max 2000 characters)", "", nil)
		return
	}
	pid, ok := parseUUIDParam(w, "projectId", req.ProjectID)
	if !ok {
		return
	}
	content := "[agent checkpoint] " + req.Summary
	if req.NodeID != "" {
		// The node a checkpoint names is a node of its project (rv5-PARB-10).
		nid, ok := parseUUIDParam(w, "nodeId", req.NodeID)
		if !ok {
			return
		}
		n, err := s.store.GetRoadmapNode(r.Context(), userID, nid)
		if err == nil && n.ProjectID != pid {
			err = store.ErrNotFound
		}
		if err != nil {
			s.writeStoreError(w, "agent checkpoint node", err, errMsgs{store.ErrNotFound: "node not found in this project"})
			return
		}
		content += " (node " + nid.String() + ")"
	}
	u, err := s.store.AddProjectUpdate(r.Context(), userID, pid, "agent_checkpoint", content)
	if err != nil {
		s.writeStoreError(w, "agent checkpoint", err, errMsgs{
			store.ErrNotFound:  "project not found",
			store.ErrForbidden: "you must be a member of this workspace",
		})
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
		// NodeID (optional) is the node the proposal is about.
		NodeID string `json:"nodeId"`
		// Items are structured actions a human may apply on acceptance
		// (proposal→structure application). Optional, max 10.
		Items []struct {
			Action     string `json:"action"`
			Title      string `json:"title"`
			ParentHint string `json:"parentHint"`
		} `json:"items"`
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
	if req.Summary == "" || tooLong(req.Summary, 2000) {
		httpx.WriteError(w, http.StatusBadRequest, "summary is required (max 2000 characters)", "", nil)
		return
	}
	if refuseTooLong(w, "detail", strings.TrimSpace(req.Detail)) {
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
	// Every item is validated before the first write: a refused propose must
	// leave nothing behind (a 400 once left an orphan proposal a human could
	// still accept).
	items := make([]store.ProposalItem, 0, len(req.Items))
	for _, it := range req.Items {
		title := strings.TrimSpace(it.Title)
		if title == "" || tooLong(title, 200) {
			httpx.WriteError(w, http.StatusBadRequest, "item title is required (max 200 characters)", "", nil)
			return
		}
		action := it.Action
		if action == "" {
			action = "add_node"
		}
		if action != "add_node" {
			httpx.WriteError(w, http.StatusBadRequest, "item action must be add_node", "", nil)
			return
		}
		items = append(items, store.ProposalItem{Action: action, Title: title, ParentHint: strings.TrimSpace(it.ParentHint)})
	}
	if len(items) > 10 {
		httpx.WriteError(w, http.StatusBadRequest, "at most 10 items per proposal", "", nil)
		return
	}
	// A proposal about a node is open to anyone who may write that node (owner
	// decision 7); a root-scoped agent token must name one inside its root.
	var nid *uuid.UUID
	if req.NodeID != "" {
		id, ok := parseUUIDParam(w, "nodeId", req.NodeID)
		if !ok {
			return
		}
		nid = &id
	}
	if s.agentRootRefuses(w, r, nid) {
		return
	}
	// The proposal and its items are one transaction: both land or neither.
	u, err := s.store.AddProposal(r.Context(), userID, pid, nid, content, items)
	if err != nil {
		s.writeStoreError(w, "agent propose", err, errMsgs{
			store.ErrNotFound:  "project or node not found",
			store.ErrForbidden: "only the project owner or a workspace admin can propose a change to the whole plan; anyone who may write a node can propose a change to that node or below it",
		})
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": u.ID, "kind": u.Kind, "proposed": true, "accepted": false, "items": len(items)})
}

// handleProposalDecision is the human accept/reject of an agent
// proposal (the proposal acceptance workflow). Owner or admin only.
func (s *Server) handleProposalDecision(w http.ResponseWriter, r *http.Request) {
	// An agent files its proposals under the user's own id, so a decision under
	// that id would be the author deciding its own proposal. requireAuth already
	// refuses an agent token here (agentRoutePolicy); this keeps the rule beside
	// the decision should the table ever be edited the wrong way (rv4-FEAT-01).
	if ScopeOf(r.Context()) == auth.ScopeAgent {
		writeAgentScopeDenied(w)
		return
	}
	userID, _ := UserID(r.Context())
	uid, ok := pathUUID(w, r, "uid")
	if !ok {
		return
	}
	var req struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
		// Apply executes the proposal's structured items on acceptance
		// (proposal→structure application), once: a repeat applies only
		// what is still unapplied. Rejected decisions ignore it.
		Apply bool `json:"apply"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	if req.Decision != "accepted" && req.Decision != "rejected" {
		httpx.WriteError(w, http.StatusBadRequest, "decision must be \"accepted\" or \"rejected\"", "", nil)
		return
	}
	if refuseTooLong(w, "reason", strings.TrimSpace(req.Reason)) {
		return
	}
	// Decide and apply are one transaction: an apply that fails answers 500 and
	// records no decision, so the proposal stays open to accept again; items
	// already applied by an earlier accept are skipped, never created twice.
	d, nodeIDs, err := s.store.DecideAndApplyProposal(r.Context(), userID, uid, req.Decision, strings.TrimSpace(req.Reason), req.Apply)
	if err != nil {
		s.writeStoreError(w, "proposal decision", err, errMsgs{
			store.ErrNotFound:  "proposal not found",
			store.ErrForbidden: "only agent_proposal records can be decided, by the project owner or a workspace admin",
		})
		return
	}
	applied := make([]string, 0, len(nodeIDs))
	for _, id := range nodeIDs {
		applied = append(applied, id.String())
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"updateId": d.UpdateID, "decision": d.Decision, "reason": d.Reason,
		"decidedBy": d.DecidedBy, "decidedAt": d.DecidedAt, "appliedNodes": applied,
	})
}
