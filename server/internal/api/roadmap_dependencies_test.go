package api

import (
	"context"
	"net/http"
	"slices"
	"testing"

	"github.com/google/uuid"
)

// Node dependencies on the wire (2026-11-22 plan, "Dependency contract"), on
// the tree world: Root → {Payments (member's) → Card, Shipping}. Real
// PostgreSQL, the real router.

func depPath(nid, did uuid.UUID) string {
	return nodePath(nid) + "/dependencies/" + did.String()
}

func depBody(id uuid.UUID) string { return `{"dependsOnId":"` + id.String() + `"}` }

func TestADependencyIsAddedAndRemovedOverTheAPI(t *testing.T) {
	w := newTreeWorld(t)
	deps := nodePath(w.card.ID) + "/dependencies"

	for range 2 { // the second add is no change
		got := decodeAs[map[string]any](t, w.must(t, w.owner, "POST", deps, depBody(w.shipping.ID), http.StatusCreated))
		if got["id"] != w.card.ID.String() || !slices.Equal(anyStrings(got["dependsOn"]), []string{w.shipping.ID.String()}) {
			t.Fatalf("POST answered %v, want Card's view waiting on Shipping", got)
		}
	}
	// Every node view carries dependsOn, an empty list when it waits on nothing.
	for _, n := range decodeAs[[]map[string]any](t, w.must(t, w.member, "GET", w.proj()+"/roadmap", "", http.StatusOK)) {
		ds, ok := n["dependsOn"].([]any)
		if !ok || (n["id"] == w.card.ID.String()) != (len(ds) == 1) {
			t.Errorf("list: %v dependsOn = %v", n["title"], n["dependsOn"])
		}
	}
	read := decodeAs[struct{ Node map[string]any }](t, w.must(t, w.member, "GET", nodePath(w.card.ID), "", http.StatusOK))
	if len(anyStrings(read.Node["dependsOn"])) != 1 {
		t.Errorf("read: dependsOn = %v", read.Node["dependsOn"])
	}
	patched := decodeAs[map[string]any](t, w.must(t, w.owner, "PATCH", nodePath(w.shipping.ID), `{"description":"x"}`, http.StatusOK))
	if ds, ok := patched["dependsOn"].([]any); !ok || len(ds) != 0 {
		t.Errorf("patch: dependsOn = %v, want []", patched["dependsOn"])
	}
	moved := decodeAs[map[string]any](t, w.must(t, w.owner, "POST", nodePath(w.card.ID)+"/move", `{"position":0}`, http.StatusOK))
	if len(anyStrings(moved["dependsOn"])) != 1 {
		t.Errorf("move: dependsOn = %v", moved["dependsOn"])
	}

	if code, body := w.call(w.owner, "DELETE", depPath(w.card.ID, w.shipping.ID), ""); code != http.StatusNoContent || body != "" {
		t.Fatalf("DELETE: %d %q, want 204 and no body", code, body)
	}
	w.expectErrors(t, []errCase{{"remove a link that is gone", w.owner, "DELETE", depPath(w.card.ID, w.shipping.ID), "",
		http.StatusNotFound, "not_found", "dependency not found"}})
}

func TestADependencyAnswersEachRefusal(t *testing.T) {
	w := newTreeWorld(t)
	ctx := context.Background()
	other, err := w.st.CreateProject(ctx, w.owner.ID, w.project.WorkspaceID, "Elsewhere", "")
	if err != nil {
		t.Fatalf("second project: %v", err)
	}
	far, err := w.st.CreateRoadmapNode(ctx, w.owner.ID, other.ID, nil, "Far", "")
	if err != nil {
		t.Fatalf("far node: %v", err)
	}
	w.must(t, w.owner, "POST", nodePath(w.card.ID)+"/dependencies", depBody(w.shipping.ID), http.StatusCreated)
	deps := func(id uuid.UUID) string { return nodePath(id) + "/dependencies" }
	w.expectErrors(t, []errCase{
		{"bad node id", w.owner, "POST", "/api/v1/roadmap/nope/dependencies", depBody(w.shipping.ID), http.StatusBadRequest, "bad_request", "invalid nid"},
		{"bad dependsOnId", w.owner, "POST", deps(w.card.ID), `{"dependsOnId":"nope"}`, http.StatusBadRequest, "bad_request", "invalid dependsOnId"},
		{"self", w.owner, "POST", deps(w.card.ID), depBody(w.card.ID), http.StatusBadRequest, "dependency_self", "a node cannot depend on itself"},
		{"on its parent", w.owner, "POST", deps(w.card.ID), depBody(w.payments.ID), http.StatusConflict, "dependency_in_tree", "a node cannot depend on its own ancestor or descendant"},
		{"on its child", w.owner, "POST", deps(w.root.ID), depBody(w.card.ID), http.StatusConflict, "dependency_in_tree", "a node cannot depend on its own ancestor or descendant"},
		{"a cycle", w.owner, "POST", deps(w.shipping.ID), depBody(w.card.ID), http.StatusConflict, "dependency_cycle", "this dependency would close a cycle"},
		{"not a writer", w.member, "POST", deps(w.shipping.ID), depBody(w.card.ID), http.StatusForbidden, "forbidden", nodeWriteDenied},
		{"not a writer removes", w.member, "DELETE", depPath(w.root.ID, w.card.ID), "", http.StatusForbidden, "forbidden", nodeWriteDenied},
		{"another project", w.owner, "POST", deps(w.card.ID), depBody(far.ID), http.StatusNotFound, "not_found", "node not found"},
		{"unknown node", w.owner, "POST", deps(uuid.New()), depBody(w.shipping.ID), http.StatusNotFound, "not_found", "node not found"},
		{"not a member", w.outsider, "POST", deps(w.card.ID), depBody(w.root.ID), http.StatusNotFound, "not_found", "node not found"},
	})
}

func TestAgentTokensAddDependenciesInsideTheirRoot(t *testing.T) {
	w := newTreeWorld(t)
	agent := w.asAgent(t, w.owner)
	w.must(t, agent, "POST", nodePath(w.shipping.ID)+"/dependencies", depBody(w.card.ID), http.StatusCreated)
	w.must(t, agent, "DELETE", depPath(w.shipping.ID, w.card.ID), "", http.StatusNoContent)

	rooted := w.rootAgent(t, w.owner, w.payments.ID)
	// Inside its root it may wait on any node of the project.
	w.must(t, rooted, "POST", nodePath(w.card.ID)+"/dependencies", depBody(w.shipping.ID), http.StatusCreated)
	// Its briefing says what Card waits on.
	brief := decodeAs[struct {
		Leaves []map[string]any `json:"leaves"`
	}](t, w.must(t, rooted, "GET", "/api/v1/agent/context?project_id="+w.project.ID.String(), "", http.StatusOK))
	if len(brief.Leaves) != 1 || !slices.Equal(anyStrings(brief.Leaves[0]["waitsOn"]), []string{"Shipping"}) {
		t.Errorf("briefing leaves = %v, want Card waiting on Shipping", brief.Leaves)
	}
	w.must(t, rooted, "DELETE", depPath(w.card.ID, w.shipping.ID), "", http.StatusNoContent)
	status, body := w.call(rooted, "POST", nodePath(w.shipping.ID)+"/dependencies", depBody(w.card.ID))
	requireScopeRefusal(t, "add outside the root", status, body)
	status, body = w.call(rooted, "DELETE", depPath(w.shipping.ID, w.card.ID), "")
	requireScopeRefusal(t, "remove outside the root", status, body)
}

// anyStrings reads a JSON array of strings; nil when v is not one.
func anyStrings(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		s, _ := e.(string)
		out = append(out, s)
	}
	return out
}
