package store_test

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/roadmaptime"
	"github.com/frankji-groundcontrol/insideout/server/internal/sparkworld"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
	"github.com/google/uuid"
)

// People-in-charge mode (2026-11-22 plan, S2): each person's `mine` starts at
// what they are in charge of. An unowned project root counts for the project
// owner and the workspace admins; each root carries what it waits on and what
// it blocks. Real PostgreSQL, on the Spark demo's tree.

func sparkStore(t *testing.T) *store.Store {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	testlock.RequireLocalURL(t, url)
	st, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	return st
}

func mineByTitle(t *testing.T, st *store.Store, w *sparkworld.World, who *store.User) map[string]store.MyRoadmapRoot {
	t.Helper()
	roots, err := st.ListMyRoadmapRoots(context.Background(), who.ID, w.Workspace.ID)
	if err != nil {
		t.Fatalf("mine of %s: %v", who.Username, err)
	}
	out := map[string]store.MyRoadmapRoot{}
	for _, r := range roots {
		out[r.Node.Title] = r
	}
	return out
}

func keys(m map[string]store.MyRoadmapRoot) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func linkTitles(links []store.RoadmapLink) []string {
	out := []string{}
	for _, l := range links {
		out = append(out, l.Title)
	}
	return out
}

func TestMineCountsTheRootTheProjectOwnerAndAdminsAreInChargeOf(t *testing.T) {
	st := sparkStore(t)
	w := sparkworld.Build(t, st)
	cases := []struct {
		who  *store.User
		want []string
	}{
		{w.Maya, []string{"Spark MVP"}},
		{w.Jo, []string{"Spark MVP"}}, // her Safety & trust sits under it
		{w.Leo, []string{"Chat", "Matching"}},
		{w.Sam, []string{"Daily match limit", "Prompt-based matching engine", "Realtime messaging service"}},
		{w.Priya, []string{"Onboarding & profiles"}},
	}
	for _, c := range cases {
		got := mineByTitle(t, st, w, c.who)
		if !slices.Equal(keys(got), c.want) {
			t.Errorf("%s: mine = %v, want %v", c.who.Username, keys(got), c.want)
		}
	}
	root := mineByTitle(t, st, w, w.Jo)["Spark MVP"]
	if root.WorkspaceID != w.Workspace.ID || len(root.WaitsOn) != 0 || len(root.Blocking) != 0 {
		t.Errorf("the whole project waits on and blocks nothing outside: %+v", root)
	}
}

func TestEachMineRootCarriesItsNowWaitsAndBlocks(t *testing.T) {
	st := sparkStore(t)
	w := sparkworld.Build(t, st)
	sam := mineByTitle(t, st, w, w.Sam)
	engine := sam["Prompt-based matching engine"]
	now := roadmaptime.Assemble(engine.Subtree, time.Now()).Now
	if len(now) != 1 || now[0].Title != "Compatibility scoring" || now[0].OwnerName != "Sam Okafor" {
		t.Errorf("engine Now = %+v, want Compatibility scoring", now)
	}
	if got := linkTitles(engine.Blocking); !slices.Equal(got, []string{"Closed beta in two cities", "Daily drop UI"}) {
		t.Errorf("engine blocks %v", got)
	}
	if len(engine.WaitsOn) != 0 {
		t.Errorf("engine waits on %v, want nothing", linkTitles(engine.WaitsOn))
	}
	if b := sam["Realtime messaging service"].Blocking; len(b) != 1 || b[0].Status != "pending" || b[0].Title != "Closed beta in two cities" {
		t.Errorf("realtime blocks %+v", b)
	}
	leo := mineByTitle(t, st, w, w.Leo)
	// Conversation starters waits on the embeddings, which are done: no wait.
	if c := leo["Chat"]; len(c.WaitsOn) != 0 || !slices.Equal(linkTitles(c.Blocking), []string{"Closed beta in two cities"}) {
		t.Errorf("Chat waits on %v, blocks %v", linkTitles(c.WaitsOn), linkTitles(c.Blocking))
	}
	launch := w.Node(t, "Launch")
	if _, err := st.UpdateRoadmapNode(context.Background(), w.Maya.ID, launch.ID, store.RoadmapNodeFields{OwnerID: &w.Priya.ID}); err != nil {
		t.Fatalf("hand Launch to Priya: %v", err)
	}
	l := mineByTitle(t, st, w, w.Priya)["Launch"]
	want := []string{"Prompt-based matching engine", "Realtime messaging service", "Report a member", "Profile builder", "Selfie verification badge"}
	slices.Sort(want)
	got := linkTitles(l.WaitsOn)
	if !slices.Equal(got, want) {
		t.Errorf("Launch waits on %v, want %v", got, want)
	}
	for _, x := range l.WaitsOn {
		if x.Title == "Realtime messaging service" && (x.OwnerName != "Sam Okafor" || x.ID == uuid.Nil) {
			t.Errorf("waits-on entry %+v lacks its owner or id", x)
		}
	}
}
