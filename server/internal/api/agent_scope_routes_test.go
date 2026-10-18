package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
)

// rv4-FEAT-01 / rv4-AIDEV-04: the agent token scope, route by route. No
// database: the point is which door opens, not what is behind it, so the
// server runs with a real token issuer and a nil store. A handler that is let
// through then fails inside (a recovered 500); only the scope verdict matters.
//
// The two lists below are the classification a reviewer signs off on. They are
// deliberately spelled out here as well as in agentRoutePolicy (agent_scope.go): a
// route added to the package without a decision in both places fails this
// test, and a decision that disagrees with the table fails the sweep.

// agentAllowedRoutes are the routes behind the MCP tools an agent needs.
var agentAllowedRoutes = []string{
	"GET /api/v1/me",
	"GET /api/v1/workspaces",
	"POST /api/v1/workspaces",
	"POST /api/v1/workspaces/join",
	"GET /api/v1/workspaces/{id}/projects",
	"POST /api/v1/workspaces/{id}/projects",
	"GET /api/v1/workspaces/{id}/ideas",
	"POST /api/v1/workspaces/{id}/ideas",
	"PATCH /api/v1/ideas/{iid}",
	"POST /api/v1/ideas/{iid}/convert",
	"GET /api/v1/prds/{id}",
	"PATCH /api/v1/prds/{id}",
	"GET /api/v1/prds/{id}/view",
	"GET /api/v1/prds/{id}/readiness",
	"GET /api/v1/prds/{id}/export",
	"GET /api/v1/prds/{id}/commits",
	"GET /api/v1/prds/{id}/revisions",
	"POST /api/v1/prds/{id}/revisions",
	"POST /api/v1/prds/{id}/status", // only to draft or reviewing, see agent_scope_status_test.go
	"GET /api/v1/projects/{pid}/roadmap",
	"POST /api/v1/projects/{pid}/roadmap",
	"PATCH /api/v1/roadmap/{nid}",
	"POST /api/v1/roadmap/{nid}/move",
	"GET /api/v1/projects/{pid}/guide",
	"GET /api/v1/projects/{pid}/roadmap/progress",
	"GET /api/v1/roadmap/{nid}/evidence",
	"POST /api/v1/prds/{pid}/build",
	"POST /api/v1/prds/{pid}/build/confirm",
	"POST /api/v1/roadmap/{nid}/expand",
	"POST /api/v1/roadmap/{nid}/expand/confirm",
	"PUT /api/v1/projects/{pid}/repo",
	"POST /api/v1/projects/{pid}/sync-github",
	"GET /api/v1/agent/context",
	"POST /api/v1/agent/checkpoint",
	"POST /api/v1/agent/propose",
	"GET /api/v1/projects/{pid}/presence",
}

// humanOnlyRoutes need the human's own signed-in session: the decisions
// (proposal decision, Commit, approve or reject), the instance (admin), every
// delete, and anything no agent tool calls.
var humanOnlyRoutes = []string{
	"POST /api/v1/agent/proposals/{uid}/decision",
	"POST /api/v1/prds/{id}/commit",
	"POST /api/v1/agent/token",
	"GET /api/v1/admin/settings",
	"PUT /api/v1/admin/settings",
	"GET /api/v1/admin/invitations",
	"POST /api/v1/admin/invitations",
	"POST /api/v1/admin/invitations/{id}/revoke",
	"POST /api/v1/admin/mail/probe",
	"DELETE /api/v1/ideas/{iid}",
	"DELETE /api/v1/projects/{pid}",
	"DELETE /api/v1/roadmap/{nid}",
	"DELETE /api/v1/updates/{uid}",
	"DELETE /api/v1/workspaces/{id}",
	"DELETE /api/v1/workspaces/{id}/members/{userId}",
	"GET /api/v1/conversations/{id}",
	"GET /api/v1/conversations/{id}/messages",
	"POST /api/v1/conversations/{id}/messages",
	"POST /api/v1/conversations/{id}/stage",
	"GET /api/v1/prds/{id}/conversation",
	"POST /api/v1/prds/{id}/conversation",
	"GET /api/v1/ideas/{iid}",
	"PATCH /api/v1/me",
	"PATCH /api/v1/projects/{pid}",
	"PATCH /api/v1/updates/{uid}",
	"PATCH /api/v1/workspaces/{id}",
	"PATCH /api/v1/workspaces/{id}/members/{userId}",
	"GET /api/v1/projects/{pid}",
	"GET /api/v1/projects/{pid}/presence/stream",
	"GET /api/v1/projects/{pid}/roadmap/progress/stream",
	"GET /api/v1/workspaces/{id}",
	"GET /api/v1/workspaces/{id}/members",
	"POST /api/v1/projects/{pid}/cursor",
	"POST /api/v1/projects/{pid}/updates",
	"POST /api/v1/roadmap/{nid}/evidence",
	"POST /api/v1/roadmap/{nid}/validate",
}

// scopeServer is a server with a real token issuer and no store.
func scopeServer() *Server {
	return NewServer(nil, auth.NewTokenIssuer(routeTestSecret, time.Hour), &config.Config{},
		slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
}

// agentTokenFor hand-builds a signed agent-scope access token. It does not use
// the issuer's mint method, so these tests describe the wire format (a
// "scope":"agent" claim) rather than echo the implementation.
func agentTokenFor(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	return scopedToken(t, userID, "agent")
}

func scopedToken(t *testing.T, userID uuid.UUID, scope string) string {
	t.Helper()
	now := time.Now()
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID.String(),
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
		"scope": scope,
	}).SignedString([]byte(routeTestSecret))
	if err != nil {
		t.Fatalf("sign %q token: %v", scope, err)
	}
	return signed
}

func fullTokenFor(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	token, err := auth.NewTokenIssuer(routeTestSecret, time.Hour).MintAccessToken(userID)
	if err != nil {
		t.Fatalf("mint full token: %v", err)
	}
	return token
}

// sendWith serves one request carrying the bearer token straight into the
// handler chain and returns the status and body.
func sendWith(h http.Handler, token, method, path, body string) (int, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	r := httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec.Code, rec.Body.String()
}

// scopeRefused is the one answer that means "your token's scope is not enough".
func scopeRefused(status int, body string) bool {
	return status == http.StatusForbidden && strings.Contains(body, "agent_scope_denied")
}

func TestEveryRouteHasAnAgentScopeDecision(t *testing.T) {
	registered := map[string]bool{}
	for _, pattern := range registeredRoutes(t) {
		registered[pattern] = true
		_, path, _ := strings.Cut(pattern, " ")
		if strings.HasPrefix(path, publicPrefix) || slices.Contains(publicRoutes, pattern) {
			continue
		}
		inAllowed, inHuman := slices.Contains(agentAllowedRoutes, pattern), slices.Contains(humanOnlyRoutes, pattern)
		switch {
		case inAllowed && inHuman:
			t.Errorf("%q is in both lists of agent_scope_routes_test.go", pattern)
		case !inAllowed && !inHuman:
			t.Errorf("%q has no agent-scope decision: add it to agentAllowedRoutes or humanOnlyRoutes here, and to agentRoutePolicy in agent_scope.go (an agent token is refused on it until you do)", pattern)
		}
	}
	for _, listed := range slices.Concat(agentAllowedRoutes, humanOnlyRoutes) {
		if !registered[listed] {
			t.Errorf("%q is classified here but no such route is registered", listed)
		}
	}
	// A scan that found almost nothing would pass vacuously.
	if len(registered) < 70 {
		t.Fatalf("scanned only %d registered routes; the registration scan is broken", len(registered))
	}
}

func TestAgentTokenOpensOnlyTheAgentRoutes(t *testing.T) {
	h := scopeServer().Handler()
	user := uuid.New()
	agent, full := agentTokenFor(t, user), fullTokenFor(t, user)
	bodies := map[string]string{"POST /api/v1/prds/{id}/status": `{"status":"reviewing"}`}

	for _, pattern := range slices.Concat(agentAllowedRoutes, humanOnlyRoutes) {
		method, path, _ := strings.Cut(pattern, " ")
		url := pathPlaceholder.ReplaceAllString(path, uuid.NewString())
		t.Run(pattern, func(t *testing.T) {
			status, body := sendWith(h, agent, method, url, bodies[pattern])
			if slices.Contains(humanOnlyRoutes, pattern) {
				if !scopeRefused(status, body) {
					t.Errorf("agent token on %s: status %d body %s, want 403 agent_scope_denied", pattern, status, body)
				}
			} else if scopeRefused(status, body) || status == http.StatusUnauthorized {
				t.Errorf("agent token on %s: status %d body %s, want the route to open (the agent tools call it)", pattern, status, body)
			}
			// The same route, with a full session: the scope never stands in its way.
			status, body = sendWith(h, full, method, url, bodies[pattern])
			if scopeRefused(status, body) || status == http.StatusUnauthorized {
				t.Errorf("full token on %s: status %d body %s, want no scope refusal", pattern, status, body)
			}
		})
	}
}

// A route whose pattern nobody classified must be refused to an agent token,
// not allowed by default: the fail-closed half of "added without a decision".
func TestAnUnclassifiedRouteIsRefusedToAnAgentToken(t *testing.T) {
	s := scopeServer()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/rv4-test/unclassified", s.requireAuth(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	user := uuid.New()

	if status, body := sendWith(mux, fullTokenFor(t, user), http.MethodGet, "/api/v1/rv4-test/unclassified", ""); status != http.StatusNoContent {
		t.Errorf("full token on an unclassified route: status %d body %s, want 204", status, body)
	}
	if status, body := sendWith(mux, agentTokenFor(t, user), http.MethodGet, "/api/v1/rv4-test/unclassified", ""); !scopeRefused(status, body) {
		t.Errorf("agent token on an unclassified route: status %d body %s, want 403 agent_scope_denied", status, body)
	}
}

// A scope this server does not know is not a session at all.
func TestAnUnknownScopeIsNotASession(t *testing.T) {
	h := scopeServer().Handler()
	for _, scope := range []string{"superuser", "AGENT", "full"} {
		status, body := sendWith(h, scopedToken(t, uuid.New(), scope), http.MethodGet, "/api/v1/me", "")
		if status != http.StatusUnauthorized {
			t.Errorf("scope %q on GET /me: status %d body %s, want 401", scope, status, body)
		}
	}
}

// The decision handler turns an agent-scope request away by itself, whatever
// the route table says: an agent never decides, and that rule sits beside the
// decision as well as in the table.
func TestTheDecisionHandlerRefusesAnAgentScopeOnItsOwn(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/agent/proposals/x/decision", strings.NewReader(`{"decision":"accepted","apply":true}`))
	r.SetPathValue("uid", uuid.NewString())
	ctx := newUserHolderContext(r.Context())
	setUserID(ctx, uuid.New())
	setScope(ctx, auth.ScopeAgent)

	rec := httptest.NewRecorder()
	scopeServer().handleProposalDecision(rec, r.WithContext(ctx)) // the store is nil: reaching it would panic
	if !scopeRefused(rec.Code, rec.Body.String()) {
		t.Errorf("agent scope on the decision handler: status %d body %s, want 403 agent_scope_denied", rec.Code, rec.Body)
	}
}
