package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/agent"
	"github.com/google/uuid"
)

// rv5 rows 1.2-1.4 (FEAT-01, PARB-02, PARB-01, FEAT-02): the build path checks
// before it writes. The draft route is read-only, only the confirm route
// creates and links a project (for the PRD author or a workspace admin), an
// unapproved PRD is refused on both, and an agent token can neither rebuild a
// populated roadmap nor write the completion override. Real PostgreSQL.

// newBuildWorld is the route world with the template planner switched on.
func newBuildWorld(t *testing.T) *routeWorld {
	t.Helper()
	w := newRouteWorld(t)
	w.srv.planner = agent.NewTemplateRoadmapPlanner()
	return w
}

// ownerPrd is a PRD the owner (the workspace admin) authored, approved when
// asked to.
func (w *routeWorld) ownerPrd(t *testing.T, approve bool) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	idea, err := w.st.CreateIdea(ctx, w.owner.ID, w.project.WorkspaceID, "Build path "+uuid.NewString(), "what it is")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	prd, _, err := w.st.ConvertIdea(ctx, w.owner.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}
	if approve {
		for _, s := range []string{"reviewing", "approved"} {
			if _, err := w.st.UpdatePrdStatus(ctx, w.owner.ID, prd.ID, s); err != nil {
				t.Fatalf("status %s: %v", s, err)
			}
		}
	}
	return prd.ID
}

// projectCount is how many projects the world's workspace holds.
func (w *routeWorld) projectCount(t *testing.T) int {
	t.Helper()
	ps, err := w.st.ListProjectsForWorkspace(context.Background(), w.owner.ID, w.project.WorkspaceID)
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	return len(ps)
}

// linkedProject is the project a PRD is linked to, or "" when none.
func (w *routeWorld) linkedProject(t *testing.T, prdID uuid.UUID) string {
	t.Helper()
	prd, err := w.st.GetPrdForMember(context.Background(), prdID, w.owner.ID)
	if err != nil {
		t.Fatalf("read prd: %v", err)
	}
	if prd.ProjectID == nil {
		return ""
	}
	return prd.ProjectID.String()
}

type buildDraft struct {
	ProjectID string          `json:"projectId"`
	Tree      json.RawMessage `json:"tree"`
}

func buildPath(prdID uuid.UUID) string { return "/api/v1/prds/" + prdID.String() + "/build" }

// confirmBody is a confirm request carrying the drafted tree.
func confirmBody(tree json.RawMessage, extra string) string {
	return `{"tree":` + string(tree) + extra + `}`
}

func requireCode(t *testing.T, what, body, code string) {
	t.Helper()
	if !strings.Contains(body, `"code":"`+code+`"`) {
		t.Fatalf("%s: body %s, want code %s", what, body, code)
	}
}

func TestADraftWritesNoProjectAndOnlyTheAuthorConfirms(t *testing.T) {
	w := newBuildWorld(t)
	prdID := w.ownerPrd(t, true)
	before := w.projectCount(t)

	// A plain member who is not the author may draft (it writes nothing).
	status, body := w.call(w.member, "POST", buildPath(prdID), "")
	expect(t, "member draft", status, body, http.StatusOK)
	draft := decodeAs[buildDraft](t, body)
	if draft.ProjectID != "" || len(draft.Tree) == 0 {
		t.Fatalf("member draft of an unlinked PRD: body %s, want a tree and no projectId", body)
	}
	status, body = w.call(w.member, "POST", buildPath(prdID)+"/confirm", confirmBody(draft.Tree, ""))
	expect(t, "member confirm", status, body, http.StatusForbidden)

	// The author's draft writes nothing either.
	status, body = w.call(w.owner, "POST", buildPath(prdID), "")
	expect(t, "author draft", status, body, http.StatusOK)
	if got := w.projectCount(t); got != before {
		t.Fatalf("drafts and a refused confirm made %d projects, want %d", got, before)
	}
	if got := w.linkedProject(t, prdID); got != "" {
		t.Fatalf("drafts linked the PRD to project %s", got)
	}

	// The author's confirm creates and links one project.
	status, body = w.call(w.owner, "POST", buildPath(prdID)+"/confirm", confirmBody(draft.Tree, ""))
	expect(t, "author confirm", status, body, http.StatusCreated)
	built := decodeAs[buildDraft](t, body)
	if got := w.linkedProject(t, prdID); got == "" || got != built.ProjectID {
		t.Fatalf("confirm answered project %s, PRD linked to %q", built.ProjectID, got)
	}
	if got := w.projectCount(t); got != before+1 {
		t.Fatalf("after confirm: %d projects, want %d", got, before+1)
	}

	// A draft of a linked PRD names the project.
	status, body = w.call(w.member, "POST", buildPath(prdID), "")
	expect(t, "draft of a linked PRD", status, body, http.StatusOK)
	if got := decodeAs[buildDraft](t, body).ProjectID; got != built.ProjectID {
		t.Fatalf("draft of a linked PRD: projectId %q, want %s", got, built.ProjectID)
	}
}

func TestAnUnapprovedPrdIsNotBuilt(t *testing.T) {
	w := newBuildWorld(t)
	prdID := w.ownerPrd(t, false)
	before := w.projectCount(t)

	status, body := w.call(w.owner, "POST", buildPath(prdID), "")
	expect(t, "draft of a draft PRD", status, body, http.StatusConflict)
	requireCode(t, "draft of a draft PRD", body, "prd_not_approved")

	status, body = w.call(w.owner, "POST", buildPath(prdID)+"/confirm", `{"tree":{"title":"Anything"}}`)
	expect(t, "confirm of a draft PRD", status, body, http.StatusConflict)
	requireCode(t, "confirm of a draft PRD", body, "prd_not_approved")

	if got := w.projectCount(t); got != before {
		t.Fatalf("refused builds made %d projects, want %d", got, before)
	}
}

func TestAnAgentTokenBuildsOnlyOntoAnEmptyRoadmap(t *testing.T) {
	w := newBuildWorld(t)
	agentOwner := w.asAgent(t, w.owner)

	// An empty roadmap: the agent may build it.
	prdID := w.ownerPrd(t, true)
	status, body := w.call(agentOwner, "POST", buildPath(prdID), "")
	expect(t, "agent draft", status, body, http.StatusOK)
	draft := decodeAs[buildDraft](t, body)
	status, body = w.call(agentOwner, "POST", buildPath(prdID)+"/confirm", confirmBody(draft.Tree, ""))
	expect(t, "agent build onto an empty roadmap", status, body, http.StatusCreated)
	projectID := uuid.MustParse(decodeAs[buildDraft](t, body).ProjectID)

	// A populated roadmap: the agent may not replace it, even confirming the count.
	nodes, err := w.st.ListRoadmap(context.Background(), w.owner.ID, projectID)
	if err != nil || len(nodes) == 0 {
		t.Fatalf("list roadmap: %d nodes, err %v", len(nodes), err)
	}
	extra := `,"expectedCount":` + strconv.Itoa(len(nodes)) + `,"force":true`
	status, body = w.call(agentOwner, "POST", buildPath(prdID)+"/confirm", confirmBody(draft.Tree, extra))
	requireScopeRefusal(t, "agent rebuild over a populated roadmap", status, body)
	after, _ := w.st.ListRoadmap(context.Background(), w.owner.ID, projectID)
	if len(after) != len(nodes) || after[0].ID != nodes[0].ID {
		t.Fatalf("a refused agent rebuild changed the roadmap")
	}

	// The human may.
	status, body = w.call(w.owner, "POST", buildPath(prdID)+"/confirm", confirmBody(draft.Tree, extra))
	expect(t, "human rebuild", status, body, http.StatusCreated)
}

func TestAnAgentTokenCannotWriteTheCompletionOverride(t *testing.T) {
	w := newBuildWorld(t)
	ctx := context.Background()
	agentOwner := w.asAgent(t, w.owner)
	node, err := w.st.CreateRoadmapNode(ctx, w.owner.ID, w.project.ID, nil, "Ship it", "")
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	path := "/api/v1/roadmap/" + node.ID.String()
	w.must(t, w.owner, "PATCH", path, `{"status":"in_progress"}`, http.StatusOK)

	status, body := w.call(agentOwner, "PATCH", path, `{"status":"done","completionReason":"It shipped to every user."}`)
	requireScopeRefusal(t, "agent completion reason", status, body)

	// With evidence the agent may still move the node to done.
	if err := w.st.AddRoadmapEvidence(ctx, w.owner.ID, node.ID, "commit", "abc1234 shipped it", ""); err != nil {
		t.Fatalf("add evidence: %v", err)
	}
	w.must(t, agentOwner, "PATCH", path, `{"status":"done"}`, http.StatusOK)
}
