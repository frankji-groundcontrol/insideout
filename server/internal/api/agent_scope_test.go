package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// rv4-FEAT-01 / rv4-AIDEV-04: an agent-scope token against real PostgreSQL,
// through the whole handler chain. Each test is the same person twice: the
// human session (w.owner, a full token) and the agent token the MCP server
// holds for them. The agent files proposals and does the day-to-day edits; the
// decisions stay with the human. Needs DATABASE_URL; the admin test also
// needs DATABASE_OWNER_URL, like the rest of the admin suite.

// asAgent is the same person holding an agent-scope token.
func (w *routeWorld) asAgent(t *testing.T, who routeUser) routeUser {
	t.Helper()
	return routeUser{who.User, agentTokenFor(t, who.ID)}
}

func requireScopeRefusal(t *testing.T, what string, status int, body string) {
	t.Helper()
	if !scopeRefused(status, body) {
		t.Fatalf("%s: status %d body %s, want 403 agent_scope_denied", what, status, body)
	}
}

// claimsOf decodes a JWT's payload without checking the signature.
func claimsOf(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token %q is not a JWT", token)
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode claims: %v", err)
	}
	claims := map[string]any{}
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatalf("parse claims %s: %v", raw, err)
	}
	return claims
}

// The headline: an agent files a proposal under the user's own id, and the
// same id decides it. With a scoped token the author actor never can.
func TestAnAgentTokenProposesButNeverDecides(t *testing.T) {
	w := newRouteWorld(t)
	agent := w.asAgent(t, w.owner)
	title := "Agent filed " + uuid.NewString()[:8]

	status, body := w.propose(agent, title)
	expect(t, "agent proposes", status, body, http.StatusCreated)
	id := decodeAs[struct{ ID string }](t, body).ID

	for _, decision := range []string{acceptAndApply, `{"decision":"accepted"}`, `{"decision":"rejected","reason":"no"}`} {
		status, body = w.decide(agent, id, decision)
		requireScopeRefusal(t, "agent decides its own proposal with "+decision, status, body)
	}
	if got := w.decisionOf(t, id); got != "" {
		t.Fatalf("the proposal carries decision %q after the agent token was refused", got)
	}
	if w.nodeTitled(t, title) {
		t.Fatalf("a roadmap node titled %q exists although every decision by the agent token was refused", title)
	}

	status, body = w.decide(w.owner, id, acceptAndApply)
	expect(t, "the human decides", status, body, http.StatusOK)
	if !w.nodeTitled(t, title) {
		t.Fatalf("the human accepted with apply:true and no roadmap node titled %q exists", title)
	}
}

// Commit freezes an immutable version; PRODUCT.md reserves it for a human.
func TestAnAgentTokenCannotCommitAVersion(t *testing.T) {
	w := newRouteWorld(t)
	agent := w.asAgent(t, w.owner)
	prd := w.prd(t)
	commit := "/api/v1/prds/" + prd + "/commit"

	status, body := w.call(agent, http.MethodPost, commit, `{"name":"agent cut","primaryAudience":"decision"}`)
	requireScopeRefusal(t, "agent commits", status, body)
	status, body = w.call(w.owner, http.MethodGet, "/api/v1/prds/"+prd+"/commits", "")
	expect(t, "list commits", status, body, http.StatusOK)
	if versions := decodeAs[[]map[string]any](t, body); len(versions) != 0 {
		t.Fatalf("the refused agent commit left %d versions: %s", len(versions), body)
	}

	status, body = w.call(w.owner, http.MethodPost, commit, `{"name":"human cut","primaryAudience":"decision"}`)
	expect(t, "the human commits", status, body, http.StatusCreated)
}

func TestAnAgentTokenCannotDelete(t *testing.T) {
	w := newRouteWorld(t)
	agent := w.asAgent(t, w.owner)
	idea, node := w.idea(t), w.roadmapNode(t, "Keep me "+uuid.NewString()[:8])

	for _, path := range []string{"/api/v1/ideas/" + idea, "/api/v1/roadmap/" + node} {
		status, body := w.call(agent, http.MethodDelete, path, "")
		requireScopeRefusal(t, "agent deletes "+path, status, body)
	}
	status, body := w.call(w.owner, http.MethodGet, "/api/v1/ideas/"+idea, "")
	expect(t, "the idea is still there", status, body, http.StatusOK)
	status, body = w.call(w.owner, http.MethodGet, "/api/v1/projects/"+w.project.ID.String()+"/roadmap", "")
	expect(t, "list the roadmap", status, body, http.StatusOK)
	if !strings.Contains(body, node) {
		t.Fatalf("the node %s is gone from the roadmap although the agent token was refused: %s", node, body)
	}

	status, body = w.call(w.owner, http.MethodDelete, "/api/v1/roadmap/"+node, "")
	expect(t, "the human deletes the node", status, body, http.StatusOK)
}

// The instance switches (signup mode, invitations) are not an agent's: not
// even for a user who is a system admin when signed in themselves.
func TestAnAgentTokenCannotReachTheAdminRoutes(t *testing.T) {
	w, admin := newAdminWorld(t)
	agent := w.asAgent(t, admin)
	ctx := context.Background()
	updatedAt := func() time.Time {
		var at time.Time
		if err := w.st.Pool.QueryRow(ctx, `SELECT updated_at FROM insideout.settings WHERE key = 'signup_mode'`).Scan(&at); err != nil {
			t.Fatalf("read signup_mode row: %v", err)
		}
		return at
	}
	status, body := w.call(admin, http.MethodGet, "/api/v1/admin/settings", "")
	expect(t, "the human reads the signup mode", status, body, http.StatusOK)
	mode := decodeAs[map[string]string](t, body)["signupMode"]
	before := updatedAt()
	email := "agent-invited-" + uuid.NewString() + "@example.com"

	for _, rt := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/admin/settings", ""},
		// The mode it already has: a missing guard changes updated_at, not the mode.
		{http.MethodPut, "/api/v1/admin/settings", `{"signupMode":"` + mode + `"}`},
		{http.MethodPost, "/api/v1/admin/invitations", `{"email":"` + email + `"}`},
		{http.MethodGet, "/api/v1/admin/invitations", ""},
		{http.MethodPost, "/api/v1/admin/invitations/" + uuid.NewString() + "/revoke", ""},
		{http.MethodPost, "/api/v1/admin/mail/probe", `{"email":"probe-` + uuid.NewString() + `@example.com"}`},
	} {
		status, body := w.call(agent, rt.method, rt.path, rt.body)
		requireScopeRefusal(t, "agent "+rt.method+" "+rt.path, status, body)
	}
	if !updatedAt().Equal(before) {
		t.Fatal("the agent token wrote the signup mode")
	}
	status, body = w.call(admin, http.MethodGet, "/api/v1/admin/invitations", "")
	expect(t, "the human lists invitations", status, body, http.StatusOK)
	if strings.Contains(body, email) {
		t.Fatalf("the agent token created an invitation for %s", email)
	}
}

func TestAHumanMintsAnAgentTokenAndItCannotMintAnother(t *testing.T) {
	w := newRouteWorld(t)
	mint := func(token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/agent/token", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		w.h.ServeHTTP(rec, r)
		return rec
	}

	before := time.Now()
	rec := mint(w.owner.token)
	expect(t, "the human mints", rec.Code, rec.Body.String(), http.StatusOK)
	if cookies := rec.Result().Cookies(); len(cookies) != 0 {
		t.Errorf("minting set cookies %v; an agent token travels in a header, never a cookie", cookies)
	}
	got := decodeAs[struct {
		AccessToken  string    `json:"accessToken"`
		RefreshToken string    `json:"refreshToken"`
		ExpiresAt    time.Time `json:"expiresAt"`
		Scope        string    `json:"scope"`
	}](t, rec.Body.String())
	if got.Scope != "agent" || got.AccessToken == "" || got.RefreshToken != "" {
		t.Fatalf("mint answered %s, want an agent-scope accessToken and no refresh token", rec.Body)
	}
	claims := claimsOf(t, got.AccessToken)
	if claims["scope"] != "agent" || claims["sub"] != w.owner.ID.String() {
		t.Errorf("minted token claims = %v, want scope agent for the caller", claims)
	}
	// newRouteWorld issues one-hour access tokens; the agent token lives as long, no longer.
	if exp, ok := claims["exp"].(float64); !ok || int64(exp) != got.ExpiresAt.Unix() ||
		got.ExpiresAt.Before(before.Add(59*time.Minute)) || got.ExpiresAt.After(time.Now().Add(61*time.Minute)) {
		t.Errorf("minted token exp = %v, expiresAt = %v, want the two to agree and sit about an hour out", claims["exp"], got.ExpiresAt)
	}

	agent := routeUser{w.owner.User, got.AccessToken}
	status, body := w.call(agent, http.MethodGet, "/api/v1/me", "")
	expect(t, "the minted token reads /me", status, body, http.StatusOK)
	status, body = w.decide(agent, uuid.NewString(), acceptAndApply)
	requireScopeRefusal(t, "the minted token decides", status, body)

	rec = mint(got.AccessToken)
	requireScopeRefusal(t, "the minted token mints another", rec.Code, rec.Body.String())
}

// What an agent does all day keeps working on a scoped token.
func TestAnAgentTokenStillDoesTheAgentsWork(t *testing.T) {
	w := newRouteWorld(t)
	agent := w.asAgent(t, w.owner)
	project, prd := w.project.ID.String(), w.prd(t)
	title := "Agent node " + uuid.NewString()[:8]

	status, body := w.call(agent, http.MethodGet, "/api/v1/me", "")
	expect(t, "whoami", status, body, http.StatusOK)
	status, body = w.call(agent, http.MethodGet, "/api/v1/workspaces", "")
	expect(t, "workspaces", status, body, http.StatusOK)
	status, body = w.call(agent, http.MethodGet, "/api/v1/agent/context?project_id="+project, "")
	expect(t, "agent context", status, body, http.StatusOK)
	status, body = w.call(agent, http.MethodPost, "/api/v1/agent/checkpoint", `{"projectId":"`+project+`","summary":"did a thing"}`)
	expect(t, "checkpoint", status, body, http.StatusCreated)
	status, body = w.call(agent, http.MethodPost, "/api/v1/projects/"+project+"/roadmap", `{"title":"`+title+`"}`)
	expect(t, "roadmap add", status, body, http.StatusCreated)
	node := decodeAs[struct{ ID string }](t, body).ID
	status, body = w.call(agent, http.MethodPatch, "/api/v1/roadmap/"+node, `{"status":"in_progress"}`)
	expect(t, "roadmap update", status, body, http.StatusOK)
	status, body = w.call(agent, http.MethodPatch, "/api/v1/prds/"+prd, `{"sections":{"background":"written by the agent"}}`)
	expect(t, "prd update", status, body, http.StatusOK)
	status, body = w.call(agent, http.MethodPost, "/api/v1/prds/"+prd+"/revisions", `{"note":"agent snapshot"}`)
	expect(t, "snapshot", status, body, http.StatusCreated)
	status, body = w.call(agent, http.MethodGet, "/api/v1/prds/"+prd+"/readiness", "")
	expect(t, "readiness", status, body, http.StatusOK)

	// Submitting for review is the author's; approving is the gate and stays human.
	status, body = w.call(agent, http.MethodPost, "/api/v1/prds/"+prd+"/status", `{"status":"reviewing"}`)
	expect(t, "submit for review", status, body, http.StatusOK)
	status, body = w.call(agent, http.MethodPost, "/api/v1/prds/"+prd+"/status", `{"status":"approved"}`)
	requireScopeRefusal(t, "agent approves", status, body)
	status, body = w.call(w.owner, http.MethodGet, "/api/v1/prds/"+prd, "")
	expect(t, "read the PRD", status, body, http.StatusOK)
	if got := decodeAs[struct{ Status string }](t, body).Status; got != "reviewing" {
		t.Fatalf("PRD status = %q after the agent's refused approval, want reviewing", got)
	}
}
