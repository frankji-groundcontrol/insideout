package api

import (
	"context"
	"net/http"
	"testing"
)

// rv5 row 2.5 (TREEB-04): a person's work tree is derived from the nodes they
// own (owner decision 1). GET /workspaces/{id}/roadmap/mine lists the
// caller's topmost owned nodes in one workspace, each a root with its
// project and path. Real PostgreSQL.

type mineEntry struct {
	Node        map[string]any `json:"node"`
	ProjectID   string         `json:"projectId"`
	ProjectName string         `json:"projectName"`
	Ancestors   []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"ancestors"`
}

func TestMineListsTheTopmostOwnedNodesOfOneWorkspace(t *testing.T) {
	w := newTreeWorld(t)
	ctx := context.Background()
	// Card sits under Payments, which the member already owns: not a root of
	// their tree. Shipping is a second root.
	w.delegate(t, w.owner, w.card.ID, w.member.ID, "", http.StatusOK)
	w.delegate(t, w.owner, w.shipping.ID, w.member.ID, "", http.StatusOK)
	// Another project in the same workspace.
	other, err := w.st.CreateProject(ctx, w.owner.ID, w.project.WorkspaceID, "Another", "")
	if err != nil {
		t.Fatalf("second project: %v", err)
	}
	ops, err := w.st.CreateRoadmapNode(ctx, w.owner.ID, other.ID, nil, "Ops", "")
	if err != nil {
		t.Fatalf("ops node: %v", err)
	}
	w.delegate(t, w.owner, ops.ID, w.member.ID, "", http.StatusOK)
	// A node the member owns in another workspace stays out.
	elsewhere, err := w.st.CreateWorkspace(ctx, w.outsider.ID, "Elsewhere", "")
	if err != nil {
		t.Fatalf("other workspace: %v", err)
	}
	if _, err := w.st.JoinWorkspace(ctx, w.member.ID, elsewhere.Code); err != nil {
		t.Fatalf("member joins elsewhere: %v", err)
	}
	far, err := w.st.CreateProject(ctx, w.outsider.ID, elsewhere.ID, "Far", "")
	if err != nil {
		t.Fatalf("far project: %v", err)
	}
	farNode, err := w.st.CreateRoadmapNode(ctx, w.outsider.ID, far.ID, nil, "Far node", "")
	if err != nil {
		t.Fatalf("far node: %v", err)
	}
	if _, err := w.st.DelegateRoadmapNode(ctx, w.outsider.ID, farNode.ID, w.member.ID, ""); err != nil {
		t.Fatalf("far delegate: %v", err)
	}

	got := decodeAs[[]mineEntry](t, w.must(t, w.member, "GET", w.ws()+"/roadmap/mine", "", http.StatusOK))
	titles := []string{}
	for _, e := range got {
		titles = append(titles, e.Node["title"].(string))
	}
	if len(got) != 3 || titles[0] != "Ops" || titles[1] != "Payments" || titles[2] != "Shipping" {
		t.Fatalf("mine: %v, want Ops, Payments, Shipping (project, then position)", titles)
	}
	pay := got[1]
	if pay.ProjectID != w.project.ID.String() || pay.ProjectName != w.project.Title ||
		len(pay.Ancestors) != 1 || pay.Ancestors[0].Title != "Root" || pay.Node["canWrite"] != true {
		t.Errorf("Payments entry: %+v", pay)
	}
	if r := rollupOf(t, pay.Node); r.Leaves != 1 {
		t.Errorf("Payments rollup: %+v, want Card as its one leaf", r)
	}
	if got[0].ProjectName != "Another" || len(got[0].Ancestors) != 0 {
		t.Errorf("Ops entry: %+v, want project Another and no ancestors", got[0])
	}

	// The project owner owns no node but is in charge of the unowned Root
	// (2026-11-22 owner decision); Ops, a root they handed on, is not theirs.
	owner := decodeAs[[]mineEntry](t, w.must(t, w.owner, "GET", w.ws()+"/roadmap/mine", "", http.StatusOK))
	if len(owner) != 1 || owner[0].Node["title"] != "Root" {
		t.Errorf("owner's mine: %+v, want the unowned Root", owner)
	}
	w.must(t, w.asAgent(t, w.member), "GET", w.ws()+"/roadmap/mine", "", http.StatusOK)
	w.must(t, w.outsider, "GET", w.ws()+"/roadmap/mine", "", http.StatusNotFound)
}
