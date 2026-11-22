package store_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/roadmaptime"
	"github.com/frankji-groundcontrol/insideout/server/internal/sparkworld"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// A node opened as its person in charge sees it (2026-11-22 view-as plan,
// V1): the node's owner, else the nearest owned ancestor, else the project
// owner, with the node's own first glance. Real PostgreSQL, on Spark.

func glanceOf(t *testing.T, st *store.Store, w *sparkworld.World, title string) *store.RoadmapGlance {
	t.Helper()
	g, err := st.ReadRoadmapGlance(context.Background(), w.Leo.ID, w.Node(t, title).ID)
	if err != nil {
		t.Fatalf("glance of %s: %v", title, err)
	}
	return g
}

func TestGlanceNamesWhoIsInChargeOfANode(t *testing.T) {
	st := sparkStore(t)
	w := sparkworld.Build(t, st)
	onboarding := glanceOf(t, st, w, "Onboarding & profiles")
	if c := onboarding.InCharge; c == nil || c.UserID != w.Priya.ID || c.Name != "Priya Nair" || c.Via != nil || c.ProjectOwner {
		t.Errorf("Onboarding in charge = %+v, want Priya directly", c)
	}
	now := roadmaptime.Assemble(onboarding.Subtree, time.Now()).Now
	if len(now) != 1 || now[0].Title != "Photo upload with crop" {
		t.Errorf("Onboarding now = %+v", now)
	}
	if got := linkTitles(onboarding.Blocking); !slices.Contains(got, "Closed beta in two cities") {
		t.Errorf("Onboarding blocks %v", got)
	}
	if onboarding.Node.Title != "Onboarding & profiles" {
		t.Errorf("node = %q", onboarding.Node.Title)
	}
	builder := glanceOf(t, st, w, "Profile builder")
	if c := builder.InCharge; c == nil || c.UserID != w.Priya.ID || c.Via == nil ||
		c.Via.ID != w.Node(t, "Onboarding & profiles").ID || c.Via.Title != "Onboarding & profiles" || c.ProjectOwner {
		t.Errorf("Profile builder in charge = %+v, want Priya via Onboarding", c)
	}
	root := glanceOf(t, st, w, "Spark MVP")
	if c := root.InCharge; c == nil || c.UserID != w.Maya.ID || c.Name != "Maya Chen" || c.Via != nil || !c.ProjectOwner {
		t.Errorf("Spark MVP in charge = %+v, want Maya as project owner", c)
	}
	if _, err := st.UpdateProject(context.Background(), w.Maya.ID, w.Project.ID, store.ProjectUpdateFields{
		Title: w.Project.Title, Status: w.Project.Status, ClearOwner: true}); err != nil {
		t.Fatalf("clear the project owner: %v", err)
	}
	if c := glanceOf(t, st, w, "Spark MVP").InCharge; c != nil {
		t.Errorf("an unowned root of an ownerless project: in charge = %+v, want nil", c)
	}
}

func TestGlanceOfANodeTheActorCannotSeeIsNotFound(t *testing.T) {
	st := sparkStore(t)
	w := sparkworld.Build(t, st)
	ctx := context.Background()
	outsider, err := st.CreateUser(ctx, "glance-"+uuid.NewString()+"@test.local", "x", "Outsider")
	if err != nil {
		t.Fatalf("outsider: %v", err)
	}
	_, err = st.ReadRoadmapGlance(ctx, outsider.ID, w.Node(t, "Onboarding & profiles").ID)
	if !errors.Is(err, store.ErrNotFound) && !errors.Is(err, store.ErrForbidden) {
		t.Errorf("outsider: err = %v, want not found or forbidden", err)
	}
	if _, err := st.ReadRoadmapGlance(ctx, w.Leo.ID, uuid.New()); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("unknown node: err = %v, want ErrNotFound", err)
	}
}
