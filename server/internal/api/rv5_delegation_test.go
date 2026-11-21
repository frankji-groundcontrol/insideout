package api

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// rv5 row 2.6 (TREEB-01, TREEB-09, TREEX-08): delegating a node is a recorded
// act. Each owner change appends to the node's assignment history and to the
// project feed, a delegate passes on what lies below the node they were
// handed but not that node, and a hand-back returns the node to whoever
// delegated it. Real PostgreSQL.

type assignmentView struct {
	ID             string  `json:"id"`
	Kind           string  `json:"kind"`
	FromUserID     *string `json:"fromUserId"`
	FromName       *string `json:"fromName"`
	ToUserID       *string `json:"toUserId"`
	ToName         *string `json:"toName"`
	AssignedBy     *string `json:"assignedBy"`
	AssignedByName *string `json:"assignedByName"`
	Note           string  `json:"note"`
	CreatedAt      string  `json:"createdAt"`
}

func nodePath(id uuid.UUID) string { return "/api/v1/roadmap/" + id.String() }

func (w *treeWorld) delegate(t *testing.T, as routeUser, node uuid.UUID, to uuid.UUID, note string, want int) map[string]any {
	t.Helper()
	body := w.must(t, as, "POST", nodePath(node)+"/delegate", `{"toUserId":"`+to.String()+`","note":`+js(note)+`}`, want)
	if want != http.StatusOK {
		return nil
	}
	return decodeAs[map[string]any](t, body)
}

func (w *treeWorld) assignments(t *testing.T, as routeUser, node uuid.UUID) []assignmentView {
	t.Helper()
	return decodeAs[[]assignmentView](t, w.must(t, as, "GET", nodePath(node)+"/assignments", "", http.StatusOK))
}

// feedKinds returns kind → node ids of the project's feed rows.
func (w *treeWorld) feedKinds(t *testing.T, as routeUser) map[string][]string {
	t.Helper()
	got := decodeAs[struct {
		Updates []map[string]any `json:"updates"`
	}](t, w.must(t, as, "GET", w.proj(), "", http.StatusOK))
	out := map[string][]string{}
	for _, u := range got.Updates {
		id, _ := u["nodeId"].(string)
		out[u["kind"].(string)] = append(out[u["kind"].(string)], id)
	}
	return out
}

func is(p *string, want string) bool { return p != nil && *p == want }

func TestAPlanWriterDelegatesANodeOnTheRecord(t *testing.T) {
	w := newTreeWorld(t)
	got := w.delegate(t, w.owner, w.shipping.ID, w.member.ID, "yours from Monday", http.StatusOK)
	if got["ownerId"] != w.member.ID.String() || got["canWrite"] != true {
		t.Fatalf("delegate response: owner %v canWrite %v, want the member and true", got["ownerId"], got["canWrite"])
	}
	hist := w.assignments(t, w.member, w.shipping.ID)
	if len(hist) != 1 {
		t.Fatalf("history of Shipping: %+v, want one row", hist)
	}
	h := hist[0]
	if h.Kind != "delegate" || h.FromUserID != nil || !is(h.ToUserID, w.member.ID.String()) || !is(h.ToName, w.member.Username) ||
		!is(h.AssignedBy, w.owner.ID.String()) || !is(h.AssignedByName, w.owner.Username) || h.Note != "yours from Monday" || h.CreatedAt == "" {
		t.Errorf("assignment row: %+v", h)
	}
	if ids := w.feedKinds(t, w.owner)["node_delegated"]; len(ids) == 0 || ids[0] != w.shipping.ID.String() {
		t.Errorf("feed node_delegated rows: %v, want one for Shipping", ids)
	}
}

func TestADelegatePassesOnADescendantButNotTheirOwnNode(t *testing.T) {
	w := newTreeWorld(t)
	third := w.newMember(t, "third")
	w.delegate(t, w.member, w.card.ID, third.ID, "", http.StatusOK)
	w.delegate(t, w.member, w.payments.ID, third.ID, "", http.StatusForbidden)
	w.delegate(t, w.member, w.shipping.ID, third.ID, "", http.StatusForbidden)
	w.delegate(t, w.owner, w.shipping.ID, w.outsider.ID, "", http.StatusBadRequest)
	w.must(t, w.owner, "POST", nodePath(w.shipping.ID)+"/delegate", `{"toUserId":"nope"}`, http.StatusBadRequest)
	w.must(t, w.owner, "POST", nodePath(uuid.New())+"/delegate", `{"toUserId":"`+w.member.ID.String()+`"}`, http.StatusNotFound)
	if hist := w.assignments(t, w.owner, w.card.ID); len(hist) != 1 || !is(hist[0].AssignedBy, w.member.ID.String()) {
		t.Errorf("Card history: %+v, want one row assigned by the member", hist)
	}
}

func TestAnAgentTokenCannotDelegateOrHandBack(t *testing.T) {
	w := newTreeWorld(t)
	for _, who := range []routeUser{w.owner, w.member} {
		agentTok := w.asAgent(t, who)
		status, body := w.call(agentTok, "POST", nodePath(w.card.ID)+"/delegate", `{"toUserId":"`+w.owner.ID.String()+`"}`)
		requireScopeRefusal(t, "agent delegate", status, body)
		status, body = w.call(agentTok, "POST", nodePath(w.payments.ID)+"/handback", `{}`)
		requireScopeRefusal(t, "agent handback", status, body)
	}
	w.must(t, w.asAgent(t, w.member), "GET", nodePath(w.payments.ID)+"/assignments", "", http.StatusOK)
}

func TestAHandBackReturnsTheNodeToItsDelegator(t *testing.T) {
	w := newTreeWorld(t)
	third := w.newMember(t, "third")
	w.delegate(t, w.member, w.card.ID, third.ID, "", http.StatusOK)
	w.must(t, w.member, "POST", nodePath(w.card.ID)+"/handback", `{}`, http.StatusForbidden)

	back := decodeAs[map[string]any](t, w.must(t, third, "POST", nodePath(w.card.ID)+"/handback", `{"note":"blocked on legal"}`, http.StatusOK))
	if back["ownerId"] != w.member.ID.String() || back["canWrite"] != false {
		t.Errorf("hand-back response: owner %v canWrite %v, want the member and false", back["ownerId"], back["canWrite"])
	}
	// The member hands their own delegated root back to the project owner.
	root := decodeAs[map[string]any](t, w.must(t, w.member, "POST", nodePath(w.payments.ID)+"/handback", `{}`, http.StatusOK))
	if root["ownerId"] != w.owner.ID.String() || root["canWrite"] != false {
		t.Errorf("hand-back of Payments: owner %v canWrite %v, want the project owner and false", root["ownerId"], root["canWrite"])
	}
	hist := w.assignments(t, w.owner, w.card.ID)
	if len(hist) != 2 || hist[0].Kind != "handback" || hist[0].Note != "blocked on legal" ||
		!is(hist[0].FromUserID, third.ID.String()) || !is(hist[0].ToUserID, w.member.ID.String()) {
		t.Errorf("Card history newest first: %+v", hist)
	}
	if ids := w.feedKinds(t, w.owner)["node_handed_back"]; len(ids) != 2 {
		t.Errorf("feed node_handed_back rows: %v, want two", ids)
	}
}

func TestHistorySurvivesAReDelegation(t *testing.T) {
	w := newTreeWorld(t)
	third := w.newMember(t, "third")
	w.delegate(t, w.owner, w.shipping.ID, w.member.ID, "first", http.StatusOK)
	w.delegate(t, w.owner, w.shipping.ID, third.ID, "second", http.StatusOK)
	// The web form re-sends the shown owner: not a change, not a row.
	w.must(t, w.owner, "PATCH", nodePath(w.shipping.ID), `{"title":"Shipping","ownerId":"`+third.ID.String()+`"}`, http.StatusOK)
	w.must(t, w.owner, "PATCH", nodePath(w.shipping.ID), `{"ownerId":""}`, http.StatusOK)
	hist := w.assignments(t, w.owner, w.shipping.ID)
	if len(hist) != 3 {
		t.Fatalf("Shipping history: %+v, want three rows", hist)
	}
	if hist[0].Kind != "cleared" || !is(hist[0].FromUserID, third.ID.String()) || hist[0].ToUserID != nil {
		t.Errorf("newest row: %+v, want cleared from third", hist[0])
	}
	if hist[1].Note != "second" || !is(hist[1].FromUserID, w.member.ID.String()) || !is(hist[1].ToName, third.Username) {
		t.Errorf("middle row: %+v, want member to third", hist[1])
	}
	if hist[2].Note != "first" || hist[2].FromUserID != nil {
		t.Errorf("oldest row: %+v, want nobody to member", hist[2])
	}
}

func TestFinishingANodeLandsInTheFeed(t *testing.T) {
	w := newTreeWorld(t)
	card := nodePath(w.card.ID)
	w.must(t, w.member, "PATCH", card, `{"status":"pending"}`, http.StatusOK)
	w.must(t, w.member, "PATCH", card, `{"status":"in_progress"}`, http.StatusOK)
	w.must(t, w.member, "PATCH", card, `{"status":"done","completionReason":"Customers can pay by card now."}`, http.StatusOK)
	if ids := w.feedKinds(t, w.owner)["node_done"]; len(ids) != 1 || ids[0] != w.card.ID.String() {
		t.Errorf("feed node_done rows: %v, want one for Card", ids)
	}
}

func TestLeavingTheWorkspaceIsRecordedAsAHandBack(t *testing.T) {
	w := newTreeWorld(t)
	w.must(t, w.owner, "DELETE", w.ws()+"/members/"+w.member.ID.String(), "", http.StatusOK)
	hist := w.assignments(t, w.owner, w.payments.ID)
	if len(hist) == 0 || hist[0].Kind != "cleared" || !is(hist[0].FromUserID, w.member.ID.String()) ||
		!is(hist[0].AssignedBy, w.owner.ID.String()) {
		t.Fatalf("Payments history after removal: %+v, want a cleared row from the member by the owner", hist)
	}
	if ids := w.feedKinds(t, w.owner)["node_handed_back"]; len(ids) != 1 || ids[0] != w.payments.ID.String() {
		t.Errorf("feed node_handed_back rows: %v, want one for Payments", ids)
	}
}

func TestErasingAnAccountRecordsItsNodesAsCleared(t *testing.T) {
	w := newTreeWorld(t)
	if err := w.st.EraseUser(t.Context(), w.member.ID); err != nil {
		t.Fatalf("erase: %v", err)
	}
	hist := w.assignments(t, w.owner, w.payments.ID)
	if len(hist) == 0 || hist[0].Kind != "cleared" || !is(hist[0].FromUserID, w.member.ID.String()) {
		t.Fatalf("Payments history after erasure: %+v, want a cleared row from the erased member", hist)
	}
}
