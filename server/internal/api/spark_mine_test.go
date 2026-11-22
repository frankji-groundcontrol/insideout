package api

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/sparkworld"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// People-in-charge mode (2026-11-22 plan, S2), per Spark role: GET
// /roadmap/mine answers each person's roots across their workspaces, each
// with the Now work below it, what it waits on and what it blocks; Progress
// Next ranks ready work first. Real PostgreSQL.

type sparkRoutes struct {
	*routeWorld
	spark                     *sparkworld.World
	maya, leo, sam, jo, priya routeUser
}

func newSparkRoutes(t *testing.T) *sparkRoutes {
	t.Helper()
	w := &sparkRoutes{routeWorld: newRouteWorld(t)}
	w.spark = sparkworld.Build(t, w.st)
	tokens := auth.NewTokenIssuer(routeTestSecret, time.Hour)
	as := func(u *store.User) routeUser {
		token, err := tokens.MintAccessToken(u.ID)
		if err != nil {
			t.Fatalf("mint: %v", err)
		}
		return routeUser{u, token}
	}
	w.maya, w.leo, w.sam, w.jo, w.priya = as(w.spark.Maya), as(w.spark.Leo), as(w.spark.Sam), as(w.spark.Jo), as(w.spark.Priya)
	return w
}

type glanceItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Pressure  string `json:"pressure"`
	OwnerName string `json:"ownerName"`
	WaitsOn   []struct {
		Title string `json:"title"`
	} `json:"waitsOn"`
}

type glanceRoot struct {
	Node        map[string]any `json:"node"`
	ProjectID   string         `json:"projectId"`
	WorkspaceID string         `json:"workspaceId"`
	Now         []glanceItem   `json:"now"`
	WaitsOn     []glanceItem   `json:"waitsOn"`
	Blocking    []glanceItem   `json:"blocking"`
}

func (w *sparkRoutes) mine(t *testing.T, as routeUser, path string) map[string]glanceRoot {
	t.Helper()
	out := map[string]glanceRoot{}
	for _, r := range decodeAs[[]glanceRoot](t, w.must(t, as, "GET", path, "", http.StatusOK)) {
		out[r.Node["title"].(string)] = r
	}
	return out
}

func titlesOf(items []glanceItem) []string {
	out := []string{}
	for _, it := range items {
		out = append(out, it.Title)
	}
	return out
}

func TestMineGivesEachSparkRoleWhatTheyAreInChargeOf(t *testing.T) {
	w := newSparkRoutes(t)
	for _, who := range []routeUser{w.maya, w.jo} {
		got := w.mine(t, who, "/api/v1/roadmap/mine")
		root, ok := got["Spark MVP"]
		if len(got) != 1 || !ok {
			t.Fatalf("%s: mine = %v, want Spark MVP", who.Username, got)
		}
		want := []string{"Realtime messaging service", "Moderator review queue", "Daily match limit",
			"Photo upload with crop", "Compatibility scoring"}
		if !slices.Equal(titlesOf(root.Now), want) || len(root.WaitsOn) != 0 || len(root.Blocking) != 0 ||
			root.WorkspaceID != w.spark.Workspace.ID.String() || root.ProjectID != w.spark.Project.ID.String() {
			t.Errorf("%s: Spark MVP now %v, waits %d, blocks %d", who.Username, titlesOf(root.Now), len(root.WaitsOn), len(root.Blocking))
		}
	}
	leo := w.mine(t, w.leo, "/api/v1/roadmap/mine")
	if _, ok := leo["Matching"]; !ok || len(leo) != 2 || leo["Chat"].Node == nil {
		t.Errorf("Leo: mine = %v, want Matching and Chat", leo)
	}
}

func TestSamSeesHisNowAndWhatHisWorkBlocks(t *testing.T) {
	w := newSparkRoutes(t)
	// Sam's side project in a workspace of his own: he owns the project, so
	// its unowned root is his too, and only the all-workspaces read shows it.
	ctx := context.Background()
	ws, err := w.st.CreateWorkspace(ctx, w.sam.ID, "Side", "")
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	side, err := w.st.CreateProject(ctx, w.sam.ID, ws.ID, "Side", "")
	if err != nil {
		t.Fatalf("project: %v", err)
	}
	if _, err := w.st.CreateRoadmapNode(ctx, w.sam.ID, side.ID, nil, "Side root", ""); err != nil {
		t.Fatalf("root: %v", err)
	}
	all := w.mine(t, w.sam, "/api/v1/roadmap/mine")
	if len(all) != 4 || all["Side root"].WorkspaceID != ws.ID.String() {
		t.Fatalf("Sam across workspaces: %v", all)
	}
	spark := w.mine(t, w.sam, "/api/v1/workspaces/"+w.spark.Workspace.ID.String()+"/roadmap/mine")
	if _, ok := spark["Side root"]; ok || len(spark) != 3 {
		t.Errorf("Sam in Spark only: %v", spark)
	}
	engine := all["Prompt-based matching engine"]
	if n := engine.Now; len(n) != 1 || n[0].Title != "Compatibility scoring" || n[0].Pressure != "near" || n[0].OwnerName != "Sam Okafor" {
		t.Errorf("engine now = %+v", n)
	}
	if b := titlesOf(engine.Blocking); !slices.Equal(b, []string{"Closed beta in two cities", "Daily drop UI"}) {
		t.Errorf("engine blocks %v", b)
	}
	rt := all["Realtime messaging service"]
	if len(rt.Now) != 1 || rt.Now[0].Pressure != "overdue" || !slices.Equal(titlesOf(rt.Blocking), []string{"Closed beta in two cities"}) {
		t.Errorf("realtime: now %+v, blocks %v", rt.Now, titlesOf(rt.Blocking))
	}
	if limit := all["Daily match limit"]; len(limit.Now) != 1 || limit.Now[0].Title != "Daily match limit" {
		t.Errorf("a leaf root in progress is its own Now: %+v", limit.Now)
	}
	w.must(t, w.asAgent(t, w.sam), "GET", "/api/v1/roadmap/mine", "", http.StatusOK)
	if got := w.must(t, w.outsider, "GET", "/api/v1/roadmap/mine", "", http.StatusOK); got != "[]\n" && got != "[]" {
		t.Errorf("outsider: %s, want []", got)
	}
}

func TestProgressNextRanksReadyWorkBeforeWaitingWork(t *testing.T) {
	w := newSparkRoutes(t)
	path := "/api/v1/projects/" + w.spark.Project.ID.String() + "/roadmap/progress?root=" + w.spark.Node(t, "Matching").ID.String()
	p := decodeAs[struct {
		Next []glanceItem `json:"next"`
	}](t, w.must(t, w.leo, "GET", path, "", http.StatusOK))
	want := []string{"Match explanation copy", "Daily drop UI", "Matching quality dashboard"}
	if !slices.Equal(titlesOf(p.Next), want) {
		t.Fatalf("Next = %v, want %v", titlesOf(p.Next), want)
	}
	if len(p.Next[0].WaitsOn) != 0 || len(p.Next[1].WaitsOn) != 1 || p.Next[1].WaitsOn[0].Title != "Compatibility scoring" {
		t.Errorf("waitsOn: %+v", p.Next)
	}
}
