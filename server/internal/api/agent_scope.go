package api

import (
	"bytes"
	"io"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
)

// Agent token scope (rv4-FEAT-01, docs/architecture/agent-token-scope.md).
//
// PRODUCT.md reserves proposal decisions, Commit and the instance switches for
// a human. A token whose "scope" claim is "agent" is what insideout-mcp sends
// for its tools; requireAuth checks it against agentRoutePolicy and refuses
// every route that is not marked agent-allowed. The rule lives here, on the
// server, so curl and any other client meet the same wall as the MCP binary.

// agentAccess is how an agent-scope token meets a route.
type agentAccess int

const (
	// humanOnly is the zero value on purpose: a pattern with no entry, or no
	// pattern at all (a handler not served by the mux), is closed to an agent
	// token. A new route is refused to agents until someone opens it here.
	humanOnly agentAccess = iota
	// agentOK is a route behind an MCP tool an agent needs.
	agentOK
	// agentSubmit is the PRD status route: an agent may take a PRD to draft or
	// reviewing (the author's moves), never to approved or rejected (the
	// workspace admin's review gate). The request body decides.
	agentSubmit
)

// agentRoutePolicy classifies every authenticated route by its mux pattern.
// TestAgentRoutePolicyCoversEveryRoute fails when a registered route has no
// entry here, or an entry names no route, so adding a route is a decision.
var agentRoutePolicy = map[string]agentAccess{
	// The agent's own vocabulary: context, checkpoint, propose.
	"GET /api/v1/agent/context":     agentOK,
	"POST /api/v1/agent/checkpoint": agentOK,
	"POST /api/v1/agent/propose":    agentOK,
	// Decisions stay with the signed-in human. An agent proposes under the
	// user's id and a decision under the same id would be the author deciding
	// its own proposal, so the whole actor kind is refused.
	"POST /api/v1/agent/proposals/{uid}/decision": humanOnly,
	"POST /api/v1/prds/{id}/commit":               humanOnly,
	"POST /api/v1/agent/token":                    humanOnly, // an agent token never mints another

	// Reads and edits behind the MCP tools.
	"GET /api/v1/me":                              agentOK,
	"GET /api/v1/workspaces":                      agentOK,
	"POST /api/v1/workspaces":                     agentOK,
	"POST /api/v1/workspaces/join":                agentOK,
	"GET /api/v1/workspaces/{id}/projects":        agentOK,
	"POST /api/v1/workspaces/{id}/projects":       agentOK,
	"GET /api/v1/workspaces/{id}/ideas":           agentOK,
	"POST /api/v1/workspaces/{id}/ideas":          agentOK,
	"PATCH /api/v1/ideas/{iid}":                   agentOK,
	"POST /api/v1/ideas/{iid}/convert":            agentOK,
	"GET /api/v1/prds/{id}":                       agentOK,
	"PATCH /api/v1/prds/{id}":                     agentOK,
	"GET /api/v1/prds/{id}/view":                  agentOK,
	"GET /api/v1/prds/{id}/readiness":             agentOK,
	"GET /api/v1/prds/{id}/export":                agentOK,
	"GET /api/v1/prds/{id}/commits":               agentOK,
	"GET /api/v1/prds/{id}/revisions":             agentOK,
	"POST /api/v1/prds/{id}/revisions":            agentOK,
	"POST /api/v1/prds/{id}/status":               agentSubmit,
	"GET /api/v1/projects/{pid}/roadmap":          agentOK,
	"POST /api/v1/projects/{pid}/roadmap":         agentOK,
	"PATCH /api/v1/roadmap/{nid}":                 agentOK,
	"POST /api/v1/roadmap/{nid}/move":             agentOK,
	"GET /api/v1/projects/{pid}/guide":            agentOK,
	"GET /api/v1/projects/{pid}/roadmap/progress": agentOK,
	"GET /api/v1/roadmap/{nid}/evidence":          agentOK,
	"POST /api/v1/prds/{pid}/build":               agentOK,
	"POST /api/v1/prds/{pid}/build/confirm":       agentOK,
	"POST /api/v1/roadmap/{nid}/expand":           agentOK,
	"POST /api/v1/roadmap/{nid}/expand/confirm":   agentOK,
	"PUT /api/v1/projects/{pid}/repo":             agentOK,
	"POST /api/v1/projects/{pid}/sync-github":     agentOK,
	"GET /api/v1/projects/{pid}/presence":         agentOK,

	// The instance: signup mode, invitations, the mail probe.
	"GET /api/v1/admin/settings":                 humanOnly,
	"PUT /api/v1/admin/settings":                 humanOnly,
	"GET /api/v1/admin/invitations":              humanOnly,
	"POST /api/v1/admin/invitations":             humanOnly,
	"POST /api/v1/admin/invitations/{id}/revoke": humanOnly,
	"POST /api/v1/admin/mail/probe":              humanOnly,

	// Every delete.
	"DELETE /api/v1/ideas/{iid}":                      humanOnly,
	"DELETE /api/v1/projects/{pid}":                   humanOnly,
	"DELETE /api/v1/roadmap/{nid}":                    humanOnly,
	"DELETE /api/v1/updates/{uid}":                    humanOnly,
	"DELETE /api/v1/workspaces/{id}":                  humanOnly,
	"DELETE /api/v1/workspaces/{id}/members/{userId}": humanOnly,

	// No agent tool calls these: the coach conversation (the web client's),
	// edits to people, workspaces and projects, live streams, evidence entry.
	"GET /api/v1/conversations/{id}":                     humanOnly,
	"GET /api/v1/conversations/{id}/messages":            humanOnly,
	"POST /api/v1/conversations/{id}/messages":           humanOnly,
	"POST /api/v1/conversations/{id}/stage":              humanOnly,
	"GET /api/v1/prds/{id}/conversation":                 humanOnly,
	"POST /api/v1/prds/{id}/conversation":                humanOnly,
	"GET /api/v1/ideas/{iid}":                            humanOnly,
	"PATCH /api/v1/me":                                   humanOnly,
	"PATCH /api/v1/projects/{pid}":                       humanOnly,
	"PATCH /api/v1/updates/{uid}":                        humanOnly,
	"PATCH /api/v1/workspaces/{id}":                      humanOnly,
	"PATCH /api/v1/workspaces/{id}/members/{userId}":     humanOnly,
	"GET /api/v1/projects/{pid}":                         humanOnly,
	"GET /api/v1/projects/{pid}/presence/stream":         humanOnly,
	"GET /api/v1/projects/{pid}/roadmap/progress/stream": humanOnly,
	"GET /api/v1/workspaces/{id}":                        humanOnly,
	"GET /api/v1/workspaces/{id}/members":                humanOnly,
	"POST /api/v1/projects/{pid}/cursor":                 humanOnly,
	"POST /api/v1/projects/{pid}/updates":                humanOnly,
	"POST /api/v1/roadmap/{nid}/evidence":                humanOnly,
	"POST /api/v1/roadmap/{nid}/validate":                humanOnly,
}

// agentMayCall reports whether an agent-scope token may call the route the mux
// matched. It only ever widens from the closed default through an explicit entry.
func agentMayCall(r *http.Request) bool {
	switch agentRoutePolicy[r.Pattern] {
	case agentOK:
		return true
	case agentSubmit:
		return submitsForReview(r)
	}
	return false
}

// submitsForReview reports whether the request moves a PRD to draft or
// reviewing. It decodes the body with the handler's own request type and
// decoder, so the two can never read the same bytes as different statuses, and
// puts the bytes back for the handler. The body is already capped (withMaxBody);
// one that cannot be read whole, or understood, is not a submission.
func submitsForReview(r *http.Request) bool {
	if r.Body == nil {
		return false
	}
	body, err := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return false
	}
	var req updateStatusRequest
	if httpx.DecodeJSON(&http.Request{Body: io.NopCloser(bytes.NewReader(body))}, &req) != nil {
		return false
	}
	return req.Status == "draft" || req.Status == "reviewing"
}

// writeAgentScopeDenied answers an agent token on a route that needs the
// human's own session.
func writeAgentScopeDenied(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusForbidden,
		"this needs your own signed-in session; an agent token cannot do it", "agent_scope_denied", nil)
}

// handleAgentToken trades the caller's full session for an agent-scope access
// token: same user, same lifetime as an access token, no cookie and no refresh
// token. insideout-mcp calls it so the tools run on a token that cannot decide,
// commit or administer. The route is human-only (agentRoutePolicy), so an agent
// token can never mint another one.
func (s *Server) handleAgentToken(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	token, expires, err := s.tokens.MintAgentToken(userID)
	if err != nil {
		s.log.Error("mint agent token", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"accessToken": token, "expiresAt": expires, "scope": string(auth.ScopeAgent),
	})
}
