package api

import (
	"net/http"
	"testing"
	"time"
)

// rv5-COMP-02: a roadmap node with every optional field set (a parent, an
// owner, a deadline and its pressure, acceptance, a completion reason,
// canWrite, a rollup with an earliest deadline, a dependency) keeps the shape
// captured in testdata/roadmap_node_full.json from this same sequence.
// client/test/api/roadmap_node_fixture_test.dart fails when
// RoadmapNode.fromJson leaves a key of that fixture unread, so a field the
// server adds reaches the web model or a test says so. DATABASE_URL-gated.
func TestRoadmapNodeWithEveryFieldKeepsItsCapturedShape(t *testing.T) {
	w := newRouteWorld(t)
	w.pinClock(roadmapDeadline.Add(-48 * time.Hour))
	add := func(title, parentID string) string {
		body := w.must(t, w.owner, http.MethodPost, w.proj()+"/roadmap", `{"title":"`+title+`","parentId":"`+parentID+`"}`, http.StatusCreated)
		return decodeAs[roadmapNodeView](t, body).ID
	}
	patch := func(id, body string) {
		w.must(t, w.owner, http.MethodPatch, "/api/v1/roadmap/"+id, body, http.StatusOK)
	}

	pilot := add("Pilot", w.roadmapNode(t, "Programme"))
	patch(pilot, `{"ownerId":"`+w.owner.ID.String()+`","acceptance":"Ten teams use it",`+roadmapDeadlineJSON()+`}`)
	patch(add("First team", pilot), `{`+roadmapDeadlineJSON()+`}`)
	patch(pilot, `{"status":"in_progress"}`)
	patch(pilot, `{"status":"done","completionReason":"Shipped by hand."}`)
	// Reopened: the reason stays as history, and unfinished work has pressure
	// again, so every optional field is set at once.
	patch(pilot, `{"status":"in_progress"}`)
	// It waits on a node in another branch.
	w.must(t, w.owner, http.MethodPost, "/api/v1/roadmap/"+pilot+"/dependencies", `{"dependsOnId":"`+w.roadmapNode(t, "Hiring")+`"}`, http.StatusCreated)

	var got any
	for _, n := range decodeAs[[]map[string]any](t, w.must(t, w.owner, http.MethodGet, w.proj()+"/roadmap", "", http.StatusOK)) {
		if n["id"] == pilot {
			got = n
		}
	}
	_, want := fixtureJSON(t, "roadmap_node_full.json")
	if shapeOf(got) != shapeOf(want) {
		t.Fatalf("a fully set roadmap node changed shape\n got  %s\n want %s\n (testdata/roadmap_node_full.json is what the client is tested against)", shapeOf(got), shapeOf(want))
	}
}
