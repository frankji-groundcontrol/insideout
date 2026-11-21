package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// Owner decision 7 (2026-11-21): anyone who may write a node files a proposal
// about that node or its descendants; only a plan writer decides it. Real
// PostgreSQL. The tree world hands Payments (→ Card) to the plain member.

// proposeOn files a proposal as who, naming node (uuid.Nil: no node).
func (w *treeWorld) proposeOn(as routeUser, node uuid.UUID) (int, string) {
	body := `{"projectId":"` + w.project.ID.String() + `","kind":"scope","summary":"narrow it"`
	if node != uuid.Nil {
		body += `,"nodeId":"` + node.String() + `"`
	}
	return w.call(as, http.MethodPost, "/api/v1/agent/propose", body+`}`)
}

// mustProposeOn files a proposal that must land, and checks the row names node.
func (w *treeWorld) mustProposeOn(t *testing.T, as routeUser, node uuid.UUID) string {
	t.Helper()
	status, body := w.proposeOn(as, node)
	expect(t, "propose", status, body, http.StatusCreated)
	id := decodeAs[map[string]any](t, body)["id"].(string)
	ups, err := w.st.ListProjectUpdatesBefore(context.Background(), w.owner.ID, w.project.ID, 0, nil)
	if err != nil {
		t.Fatalf("list updates: %v", err)
	}
	for _, u := range ups {
		if u.ID.String() != id {
			continue
		}
		got := uuid.Nil
		if u.NodeID != nil {
			got = *u.NodeID
		}
		if got != node {
			t.Fatalf("proposal %s stores node %v, want %v", id, got, node)
		}
		return id
	}
	t.Fatalf("proposal %s is not in the timeline", id)
	return ""
}

func TestADelegateProposesInsideTheirSubtree(t *testing.T) {
	w := newTreeWorld(t)
	w.mustProposeOn(t, w.member, w.payments.ID)
	w.mustProposeOn(t, w.member, w.card.ID)
	for what, node := range map[string]uuid.UUID{"sibling": w.shipping.ID, "parent": w.root.ID, "no node": uuid.Nil} {
		status, body := w.proposeOn(w.member, node)
		expect(t, "delegate proposes on the "+what, status, body, http.StatusForbidden)
	}
	plain := w.newMember(t, "plain")
	status, body := w.proposeOn(plain, w.card.ID)
	expect(t, "member with no node proposes", status, body, http.StatusForbidden)
}

func TestAPlanWriterProposesWithOrWithoutANode(t *testing.T) {
	w := newTreeWorld(t)
	w.mustProposeOn(t, w.owner, uuid.Nil)
	w.mustProposeOn(t, w.owner, w.shipping.ID)
}

func TestAProposalNodeMustBeInTheProject(t *testing.T) {
	w := newTreeWorld(t)
	ctx := context.Background()
	other, err := w.st.CreateProject(ctx, w.owner.ID, w.project.WorkspaceID, "Other", "")
	if err != nil {
		t.Fatalf("create other project: %v", err)
	}
	n, err := w.st.CreateRoadmapNode(ctx, w.owner.ID, other.ID, nil, "Elsewhere", "")
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	status, body := w.proposeOn(w.owner, n.ID)
	expect(t, "node of another project", status, body, http.StatusNotFound)
	status, body = w.proposeOn(w.owner, uuid.New())
	expect(t, "no such node", status, body, http.StatusNotFound)
	status, body = w.proposeOn(w.outsider, w.card.ID)
	expect(t, "a node the caller cannot see", status, body, http.StatusNotFound)
	status, body = w.call(w.owner, http.MethodPost, "/api/v1/agent/propose",
		`{"projectId":"`+w.project.ID.String()+`","kind":"scope","summary":"x","nodeId":"nope"}`)
	expect(t, "malformed node id", status, body, http.StatusBadRequest)
}

func TestARootScopedAgentProposesOnlyInsideItsRoot(t *testing.T) {
	w := newTreeWorld(t)
	delegateAgent := w.rootAgent(t, w.member, w.payments.ID)
	w.mustProposeOn(t, delegateAgent, w.card.ID)
	ownerAgent := w.rootAgent(t, w.owner, w.payments.ID)
	w.mustProposeOn(t, ownerAgent, w.payments.ID)
	for what, node := range map[string]uuid.UUID{"outside the root": w.shipping.ID, "no node": uuid.Nil} {
		status, body := w.proposeOn(ownerAgent, node)
		requireScopeRefusal(t, "root agent proposes "+what, status, body)
	}
	// An unscoped agent of the delegate keeps the delegate's reach.
	w.mustProposeOn(t, w.asAgent(t, w.member), w.card.ID)
}

func TestADelegateCannotDecideTheirOwnProposal(t *testing.T) {
	w := newTreeWorld(t)
	id := w.mustProposeOn(t, w.member, w.payments.ID)
	status, body := w.decide(w.member, id, `{"decision":"accepted"}`)
	expect(t, "delegate decides", status, body, http.StatusForbidden)
	status, body = w.decide(w.owner, id, `{"decision":"accepted"}`)
	expect(t, "owner decides", status, body, http.StatusOK)
}
