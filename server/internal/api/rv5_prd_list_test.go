package api

import (
	"context"
	"net/http"
	"testing"
)

// rv5 row 3.3 (FE-02, PARB-03): a member finds every PRD of the workspace,
// not only the ones they wrote, and a project names the PRD it was built
// from. Real PostgreSQL.

type prdListItem struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	AuthorID   string  `json:"authorId"`
	AuthorName string  `json:"authorName"`
	ProjectID  *string `json:"projectId"`
	UpdatedAt  string  `json:"updatedAt"`
}

func TestAMemberListsEveryPrdOfTheWorkspace(t *testing.T) {
	w := newCoverageWorld(t)
	ctx := context.Background()
	first := w.ownerPrd(t, false)
	built := w.ownerPrd(t, true)
	project, err := w.st.EnsureProjectForPrd(ctx, w.owner.ID, built)
	if err != nil {
		t.Fatalf("build project: %v", err)
	}

	list := w.ws() + "/prds"
	got := decodeAs[[]prdListItem](t, w.must(t, w.member, "GET", list, "", http.StatusOK))
	if len(got) != 2 || got[0].ID != built.String() || got[1].ID != first.String() {
		t.Fatalf("member's list: %+v, want the built PRD then the first, newest first", got)
	}
	b := got[0]
	if b.Status != "approved" || b.AuthorID != w.owner.ID.String() || b.AuthorName != w.owner.Username ||
		b.ProjectID == nil || *b.ProjectID != project.ID.String() || b.UpdatedAt == "" || b.Title == "" {
		t.Errorf("built PRD entry: %+v", b)
	}
	if got[1].ProjectID != nil || got[1].Status != "draft" {
		t.Errorf("unbuilt PRD entry: %+v, want projectId null and draft", got[1])
	}

	w.must(t, w.asAgent(t, w.member), "GET", list, "", http.StatusOK)
	w.must(t, w.outsider, "GET", list, "", http.StatusNotFound)
}

func TestAProjectNamesThePrdItWasBuiltFrom(t *testing.T) {
	w := newCoverageWorld(t)
	prd := w.ownerPrd(t, true)
	project, err := w.st.EnsureProjectForPrd(context.Background(), w.owner.ID, prd)
	if err != nil {
		t.Fatalf("build project: %v", err)
	}

	view := decodeAs[map[string]any](t, w.must(t, w.member, "GET", "/api/v1/projects/"+project.ID.String(), "", http.StatusOK))
	if view["prdId"] != prd.String() {
		t.Errorf("built project prdId = %v, want %s", view["prdId"], prd)
	}
	plain := decodeAs[map[string]any](t, w.must(t, w.member, "GET", w.proj(), "", http.StatusOK))
	if v, ok := plain["prdId"]; !ok || v != nil {
		t.Errorf("hand-made project prdId = %v (present %v), want null", v, ok)
	}
	for _, p := range decodeAs[[]map[string]any](t, w.must(t, w.member, "GET", w.ws()+"/projects", "", http.StatusOK)) {
		want := any(nil)
		if p["id"] == project.ID.String() {
			want = prd.String()
		}
		if v, ok := p["prdId"]; !ok || v != want {
			t.Errorf("board entry %v prdId = %v, want %v", p["title"], v, want)
		}
	}
}
