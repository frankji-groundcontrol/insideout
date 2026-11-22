package api

import (
	"net/http"
	"slices"
	"testing"

	"github.com/google/uuid"
)

// A node opened as its person in charge sees it (2026-11-22 view-as plan,
// V1): GET /roadmap/{nid}/glance on the Spark world, real PostgreSQL.

type glanceView struct {
	Node     map[string]any `json:"node"`
	InCharge *struct {
		UserID string `json:"userId"`
		Name   string `json:"name"`
		Via    *struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"via"`
		ProjectOwner bool `json:"projectOwner"`
	} `json:"inCharge"`
	Now      []glanceItem `json:"now"`
	WaitsOn  []glanceItem `json:"waitsOn"`
	Blocking []glanceItem `json:"blocking"`
}

func (w *sparkRoutes) glance(t *testing.T, as routeUser, title string) glanceView {
	t.Helper()
	path := "/api/v1/roadmap/" + w.spark.Node(t, title).ID.String() + "/glance"
	return decodeAs[glanceView](t, w.must(t, as, "GET", path, "", http.StatusOK))
}

func TestGlanceOpensANodeAsItsPersonInChargeSeesIt(t *testing.T) {
	w := newSparkRoutes(t)
	on := w.glance(t, w.maya, "Onboarding & profiles")
	if c := on.InCharge; c == nil || c.UserID != w.priya.ID.String() || c.Name != "Priya Nair" || c.Via != nil || c.ProjectOwner {
		t.Errorf("Onboarding in charge = %+v, want Priya", c)
	}
	if !slices.Contains(titlesOf(on.Now), "Photo upload with crop") || !slices.Contains(titlesOf(on.Blocking), "Closed beta in two cities") ||
		on.WaitsOn == nil || on.Node["title"] != "Onboarding & profiles" {
		t.Errorf("Onboarding: node %v, now %v, waits %v, blocks %v", on.Node["title"], titlesOf(on.Now), on.WaitsOn, titlesOf(on.Blocking))
	}
	pb := w.glance(t, w.leo, "Profile builder")
	if c := pb.InCharge; c == nil || c.UserID != w.priya.ID.String() || c.Via == nil ||
		c.Via.ID != w.spark.Node(t, "Onboarding & profiles").ID.String() || c.Via.Title != "Onboarding & profiles" || c.ProjectOwner {
		t.Errorf("Profile builder in charge = %+v, want Priya via Onboarding", c)
	}
	root := w.glance(t, w.asAgent(t, w.sam), "Spark MVP")
	if c := root.InCharge; c == nil || c.UserID != w.maya.ID.String() || c.Name != "Maya Chen" || c.Via != nil || !c.ProjectOwner {
		t.Errorf("Spark MVP in charge = %+v, want Maya as project owner", c)
	}
}

func TestGlanceRefusesABadIdAndHidesWhatTheCallerCannotSee(t *testing.T) {
	w := newSparkRoutes(t)
	w.must(t, w.maya, "GET", "/api/v1/roadmap/not-a-uuid/glance", "", http.StatusBadRequest)
	w.must(t, w.maya, "GET", "/api/v1/roadmap/"+uuid.NewString()+"/glance", "", http.StatusNotFound)
	node := "/api/v1/roadmap/" + w.spark.Node(t, "Onboarding & profiles").ID.String() + "/glance"
	w.must(t, w.outsider, "GET", node, "", http.StatusNotFound)
}
