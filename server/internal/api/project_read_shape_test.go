package api

import (
	"net/http"
	"testing"
)

// rv5-STRC-08: the web reads GET /projects/{id} and GET /roadmap/{id}/evidence
// into typed models; the client's typed_reads_test.dart parses the two
// captured responses in testdata/; this pins that the server still answers in
// their shapes. Real PostgreSQL through the whole handler chain.
func TestProjectAndEvidenceReadsKeepTheirCapturedShapes(t *testing.T) {
	w := newRouteWorld(t)
	w.note(t, w.owner, "Shipped the first screen")

	body := w.must(t, w.owner, http.MethodGet, w.proj(), "", http.StatusOK)
	_, want := fixtureJSON(t, "project_get.json")
	if got, want := shapeOf(decodeAs[any](t, body)), shapeOf(want); got != want {
		t.Fatalf("GET /projects/{id} changed shape\n got  %s\n want %s\n body %s\n (testdata/project_get.json is what the client is tested against)", got, want, body)
	}

	node := w.roadmapNode(t, "Pilot the tour")
	evidence := "/api/v1/roadmap/" + node + "/evidence"
	w.must(t, w.owner, http.MethodPost, evidence,
		`{"detail":"4 of 5 pilot users finished the tour","sourceUrl":"","observedAt":"2026-11-01"}`, http.StatusCreated)
	body = w.must(t, w.owner, http.MethodGet, evidence, "", http.StatusOK)
	_, want = fixtureJSON(t, "roadmap_evidence_list.json")
	if got, want := shapeOf(decodeAs[any](t, body)), shapeOf(want); got != want {
		t.Fatalf("GET /roadmap/{id}/evidence changed shape\n got  %s\n want %s\n body %s\n (testdata/roadmap_evidence_list.json is what the client is tested against)", got, want, body)
	}
}
