package api

import (
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// A root-scoped agent token (rv5-TREEB-05): POST /agent/token with a
// rootNodeId mints an agent token whose node writes stay inside that node's
// subtree. Reads stay workspace-wide (owner decision 2); the agent context
// defaults its focus to the root.

// agentRootRefuses answers 403 agent_scope_denied, and reports true, when the
// request rides a root-scoped agent token and a node it writes lies outside
// the root's subtree. A nil node is a new root of the tree: outside.
func (s *Server) agentRootRefuses(w http.ResponseWriter, r *http.Request, nodes ...*uuid.UUID) bool {
	root := RootOf(r.Context())
	if root == nil {
		return false
	}
	userID, _ := UserID(r.Context())
	for _, n := range nodes {
		if n == nil {
			writeAgentScopeDenied(w)
			return true
		}
		in, err := s.store.NodeInSubtree(r.Context(), userID, *root, *n)
		if err != nil {
			s.log.Error("agent root check", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
			return true
		}
		if !in {
			writeAgentScopeDenied(w)
			return true
		}
	}
	return false
}

// handleAgentToken trades the caller's full session for an agent-scope access
// token: same user, same lifetime as an access token, no cookie and no refresh
// token. insideout-mcp calls it so the tools run on a token that cannot decide,
// commit or administer. The route is human-only (agentRoutePolicy), so an agent
// token can never mint another one. An optional rootNodeId, a node the caller
// can see, confines the token's node writes to that subtree.
func (s *Server) handleAgentToken(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	var req struct {
		RootNodeID string `json:"rootNodeId"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	var root *uuid.UUID
	if req.RootNodeID != "" {
		id, ok := parseUUIDParam(w, "rootNodeId", req.RootNodeID)
		if !ok {
			return
		}
		_, err := s.store.GetRoadmapNode(r.Context(), userID, id)
		if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrForbidden) {
			httpx.WriteError(w, http.StatusNotFound, "root node not found", "", nil)
			return
		}
		if err != nil {
			s.log.Error("agent token root", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
			return
		}
		root = &id
	}
	token, expires, err := s.tokens.MintAgentToken(userID, root)
	if err != nil {
		s.log.Error("mint agent token", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	out := map[string]any{"accessToken": token, "expiresAt": expires, "scope": string(auth.ScopeAgent)}
	if root != nil {
		out["rootNodeId"] = root.String()
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
