package api

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// rv5 row 2.8 (TREEB-05): an agent token may carry a root node. It then
// writes only inside that node's subtree, and its briefing defaults to that
// focus. Real PostgreSQL, the real token issuer.

// rootAgent mints an agent token for who through POST /agent/token with a root.
func (w *treeWorld) rootAgent(t *testing.T, who routeUser, root uuid.UUID) routeUser {
	t.Helper()
	body := w.must(t, who, "POST", "/api/v1/agent/token", `{"rootNodeId":"`+root.String()+`"}`, http.StatusOK)
	tok := decodeAs[map[string]any](t, body)["accessToken"].(string)
	if got := claimsOf(t, tok)["root"]; got != root.String() {
		t.Fatalf("token root claim = %v, want %s", got, root)
	}
	return routeUser{who.User, tok}
}

func TestARootScopedAgentTokenWritesOnlyInsideItsSubtree(t *testing.T) {
	w := newTreeWorld(t)
	agent := w.rootAgent(t, w.owner, w.payments.ID)
	refused := func(what, method, path, body string) {
		t.Helper()
		status, resp := w.call(agent, method, path, body)
		requireScopeRefusal(t, what, status, resp)
	}

	w.must(t, agent, "PATCH", nodePath(w.payments.ID), `{"title":"Payments v2"}`, http.StatusOK)
	w.must(t, agent, "PATCH", nodePath(w.card.ID), `{"description":"cards first"}`, http.StatusOK)
	w.must(t, agent, "POST", w.proj()+"/roadmap", `{"parentId":"`+w.card.ID.String()+`","title":"Visa"}`, http.StatusCreated)
	refused("edit outside the root", "PATCH", nodePath(w.shipping.ID), `{"title":"mine"}`)
	refused("edit the root's parent", "PATCH", nodePath(w.root.ID), `{"title":"mine"}`)
	refused("add under a node outside the root", "POST", w.proj()+"/roadmap", `{"parentId":"`+w.shipping.ID.String()+`","title":"x"}`)
	refused("add a root node", "POST", w.proj()+"/roadmap", `{"title":"x"}`)
	refused("move a node out of the root", "POST", nodePath(w.card.ID)+"/move", `{"parentId":"`+w.shipping.ID.String()+`","position":0}`)
	refused("move a node in from outside", "POST", nodePath(w.shipping.ID)+"/move", `{"parentId":"`+w.payments.ID.String()+`","position":0}`)
	refused("break down outside the root", "POST", nodePath(w.shipping.ID)+"/expand", ``)
	refused("confirm a break-down outside the root", "POST", nodePath(w.shipping.ID)+"/expand/confirm", `{"children":[{"title":"x"}]}`)
	// A checkpoint naming a node is held to the root, as a proposal is.
	refused("checkpoint a node outside the root", "POST", "/api/v1/agent/checkpoint", `{"projectId":"`+w.project.ID.String()+`","nodeId":"`+w.shipping.ID.String()+`","summary":"x"}`)
	w.must(t, agent, "POST", "/api/v1/agent/checkpoint", `{"projectId":"`+w.project.ID.String()+`","nodeId":"`+w.card.ID.String()+`","summary":"cards done"}`, http.StatusCreated)
	refused("rebuild the whole tree", "POST", "/api/v1/prds/"+uuid.NewString()+"/build/confirm", `{"tree":{"title":"x"}}`)

	// The briefing defaults to the root: only its leaves.
	got := decodeAs[struct {
		Leaves []map[string]any `json:"leaves"`
	}](t, w.must(t, agent, "GET", "/api/v1/agent/context?project_id="+w.project.ID.String(), "", http.StatusOK))
	for _, l := range got.Leaves {
		if l["title"] == "Shipping" {
			t.Errorf("default-focus briefing lists Shipping, outside the root: %v", got.Leaves)
		}
	}
	if len(got.Leaves) == 0 {
		t.Error("default-focus briefing lists no leaves")
	}

	// An unscoped agent token keeps its reach.
	w.must(t, w.asAgent(t, w.owner), "PATCH", nodePath(w.shipping.ID), `{"title":"Shipping v2"}`, http.StatusOK)
}

func TestAnAgentTokenRootMustBeAVisibleNode(t *testing.T) {
	w := newTreeWorld(t)
	w.must(t, w.owner, "POST", "/api/v1/agent/token", `{"rootNodeId":"`+uuid.NewString()+`"}`, http.StatusNotFound)
	w.must(t, w.outsider, "POST", "/api/v1/agent/token", `{"rootNodeId":"`+w.payments.ID.String()+`"}`, http.StatusNotFound)
	w.must(t, w.owner, "POST", "/api/v1/agent/token", `{"rootNodeId":"nope"}`, http.StatusBadRequest)
	tok := decodeAs[map[string]any](t, w.must(t, w.owner, "POST", "/api/v1/agent/token", `{}`, http.StatusOK))["accessToken"].(string)
	if _, has := claimsOf(t, tok)["root"]; has {
		t.Error("a token minted without a root carries a root claim")
	}
}
