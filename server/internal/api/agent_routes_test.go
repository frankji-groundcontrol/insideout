package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TEST-13: an agent proposes, a human decides. Only the project's owner or a
// workspace admin may do either, and only agent_proposal records can be
// decided. DB-gated: the verdicts come from real row-level security.

func (w *routeWorld) propose(as routeUser, title string) (int, string) {
	return w.call(as, http.MethodPost, "/api/v1/agent/propose",
		`{"projectId":"`+w.project.ID.String()+`","kind":"structure","summary":"`+title+`","items":[{"title":"`+title+`"}]}`)
}

func (w *routeWorld) decide(as routeUser, updateID, body string) (int, string) {
	return w.call(as, http.MethodPost, "/api/v1/agent/proposals/"+updateID+"/decision", body)
}

// proposeAsOwner files a proposal and returns its update id.
func (w *routeWorld) proposeAsOwner(t *testing.T, title string) string {
	t.Helper()
	status, body := w.propose(w.owner, title)
	expect(t, "owner proposes", status, body, http.StatusCreated)
	got := decodeAs[struct {
		ID       string
		Proposed bool
		Items    int
	}](t, body)
	if got.ID == "" || !got.Proposed || got.Items != 1 {
		t.Fatalf("propose answered %s, want an id, proposed:true and one item", body)
	}
	return got.ID
}

func (w *routeWorld) nodeTitled(t *testing.T, title string) bool {
	t.Helper()
	nodes, err := w.st.ListRoadmap(context.Background(), w.owner.ID, w.project.ID)
	if err != nil {
		t.Fatalf("list roadmap: %v", err)
	}
	for _, n := range nodes {
		if n.Title == title {
			return true
		}
	}
	return false
}

func TestOnlyThePlanOwnerCanPropose(t *testing.T) {
	w := newRouteWorld(t)

	status, body := w.propose(w.member, "member proposal")
	expect(t, "plain member proposes", status, body, http.StatusForbidden)
	status, body = w.propose(w.outsider, "outsider proposal")
	expect(t, "outsider proposes", status, body, http.StatusNotFound)
	w.proposeAsOwner(t, "owner proposal")
}

// refused fails the test unless a call was turned away. A plain member gets a
// 404 here, not a 403: DecideProposal locks the row FOR UPDATE and RLS only
// lets the proposal's author or a workspace admin lock it, so the member never
// reaches the owner check. Both are a refusal; nothing is written either way.
func refused(t *testing.T, what string, status int, body string) {
	t.Helper()
	if status != http.StatusForbidden && status != http.StatusNotFound {
		t.Fatalf("%s: status %d, want a refusal (403 or 404); body %s", what, status, body)
	}
}

func TestProposalDecisionIsForTheOwnerOnly(t *testing.T) {
	w := newRouteWorld(t)
	id := w.proposeAsOwner(t, "Decide me "+uuid.NewString()[:8])
	accept := `{"decision":"accepted","apply":true}`

	status, body := w.decide(w.member, id, accept)
	refused(t, "member decides", status, body)
	status, body = w.decide(w.outsider, id, accept)
	refused(t, "outsider decides", status, body)
	status, body = w.decide(w.owner, id, `{"decision":"maybe"}`)
	expect(t, "owner sends an unknown decision", status, body, http.StatusBadRequest)
	status, body = w.decide(w.owner, uuid.NewString(), accept)
	expect(t, "owner decides an unknown proposal", status, body, http.StatusNotFound)

	// A checkpoint is a timeline record, not a proposal.
	status, body = w.call(w.owner, http.MethodPost, "/api/v1/agent/checkpoint",
		`{"projectId":"`+w.project.ID.String()+`","summary":"checkpoint, not a proposal"}`)
	expect(t, "owner checkpoints", status, body, http.StatusCreated)
	checkpoint := decodeAs[struct{ ID string }](t, body).ID
	status, body = w.decide(w.owner, checkpoint, accept)
	expect(t, "owner decides a checkpoint", status, body, http.StatusForbidden)
}

// The one case RLS does not already settle: a proposal's author can lock its
// row, so when that author has since lost the project (and is no one's admin)
// only the owner check in DecideProposal stands between them and a verdict.
func TestAFormerOwnerCannotDecideTheirOwnProposal(t *testing.T) {
	w := newRouteWorld(t)
	setOwner := func(who routeUser) {
		t.Helper()
		status, body := w.call(w.owner, http.MethodPatch, "/api/v1/projects/"+w.project.ID.String(),
			`{"title":"Route tests","description":"","status":"active","ownerId":"`+who.ID.String()+`"}`)
		expect(t, "reassign the project", status, body, http.StatusOK)
	}

	setOwner(w.member)
	status, body := w.propose(w.member, "From a former owner "+uuid.NewString()[:8])
	expect(t, "the new owner proposes", status, body, http.StatusCreated)
	id := decodeAs[struct{ ID string }](t, body).ID
	setOwner(w.owner)

	status, body = w.decide(w.member, id, `{"decision":"accepted","apply":true}`)
	expect(t, "former owner decides their own proposal", status, body, http.StatusForbidden)
	status, body = w.decide(w.owner, id, `{"decision":"rejected"}`)
	expect(t, "current owner decides it", status, body, http.StatusOK)
}

func TestOwnerAcceptsAndAppliesAProposal(t *testing.T) {
	w := newRouteWorld(t)
	title := "Accepted node " + uuid.NewString()[:8]
	id := w.proposeAsOwner(t, title)

	status, body := w.decide(w.owner, id, `{"decision":"accepted","reason":"fits the plan","apply":true}`)
	expect(t, "owner accepts", status, body, http.StatusOK)
	got := decodeAs[struct {
		Decision     string
		DecidedBy    string
		AppliedNodes []string
	}](t, body)
	if got.Decision != "accepted" || got.DecidedBy != w.owner.ID.String() || len(got.AppliedNodes) != 1 {
		t.Fatalf("accept answered %s, want accepted by the owner with one applied node", body)
	}
	if !w.nodeTitled(t, title) {
		t.Fatalf("accepting with apply:true left no roadmap node titled %q", title)
	}
}

func TestOwnerRejectsAProposalWithoutApplyingIt(t *testing.T) {
	w := newRouteWorld(t)
	title := "Rejected node " + uuid.NewString()[:8]
	id := w.proposeAsOwner(t, title)

	// apply:true on a rejection must be ignored.
	status, body := w.decide(w.owner, id, `{"decision":"rejected","reason":"not now","apply":true}`)
	expect(t, "owner rejects", status, body, http.StatusOK)
	got := decodeAs[struct {
		Decision     string
		Reason       string
		AppliedNodes []string
	}](t, body)
	if got.Decision != "rejected" || got.Reason != "not now" || len(got.AppliedNodes) != 0 {
		t.Fatalf("reject answered %s, want rejected with the reason and nothing applied", body)
	}
	if strings.Contains(body, `"appliedNodes":null`) {
		t.Fatalf("reject answered %s, want appliedNodes to be an empty list, not null", body)
	}
	if w.nodeTitled(t, title) {
		t.Fatalf("rejecting left a roadmap node titled %q", title)
	}
}
