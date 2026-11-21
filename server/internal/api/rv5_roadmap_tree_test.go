package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/agent"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// rv5 rows 2.1 and 2.2 on the wire (TREEB-03, TREEX-06, TREEX-07, FE-05): a
// node view says whether the caller may write it, a subtree reads on its own,
// and one node reads with its ancestors. Real PostgreSQL.

// treeWorld is a routeWorld whose project holds Root → {Payments → Card,
// Shipping}, with Payments handed to the plain member.
type treeWorld struct {
	*routeWorld
	root, payments, card, shipping *store.RoadmapNode
}

func newTreeWorld(t *testing.T) *treeWorld {
	t.Helper()
	w := &treeWorld{routeWorld: newRouteWorld(t)}
	ctx := context.Background()
	add := func(parent *store.RoadmapNode, title string) *store.RoadmapNode {
		t.Helper()
		var pid *uuid.UUID
		if parent != nil {
			pid = &parent.ID
		}
		n, err := w.st.CreateRoadmapNode(ctx, w.owner.ID, w.project.ID, pid, title, "")
		if err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
		return n
	}
	w.root = add(nil, "Root")
	w.payments = add(w.root, "Payments")
	w.shipping = add(w.root, "Shipping")
	w.card = add(w.payments, "Card")
	if _, err := w.st.UpdateRoadmapNode(ctx, w.owner.ID, w.payments.ID, store.RoadmapNodeFields{OwnerID: &w.member.ID}); err != nil {
		t.Fatalf("assign Payments: %v", err)
	}
	return w
}

// canWriteByTitle reads a node list into title → canWrite, failing when a
// node carries no canWrite at all.
func canWriteByTitle(t *testing.T, body string) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, n := range decodeAs[[]map[string]any](t, body) {
		cw, ok := n["canWrite"].(bool)
		if !ok {
			t.Fatalf("node %v has no boolean canWrite", n["title"])
		}
		out[n["title"].(string)] = cw
	}
	return out
}

func TestTheRoadmapListSaysWhoMayWriteEachNode(t *testing.T) {
	w := newTreeWorld(t)
	list := w.proj() + "/roadmap"

	got := canWriteByTitle(t, w.must(t, w.member, "GET", list, "", http.StatusOK))
	want := map[string]bool{"Root": false, "Payments": true, "Card": true, "Shipping": false}
	for title, cw := range want {
		if got[title] != cw {
			t.Errorf("member: %s canWrite = %v, want %v", title, got[title], cw)
		}
	}
	for title, cw := range canWriteByTitle(t, w.must(t, w.owner, "GET", list, "", http.StatusOK)) {
		if !cw {
			t.Errorf("owner: %s canWrite = false", title)
		}
	}
}

func TestTheRoadmapListReadsOneSubtree(t *testing.T) {
	w := newTreeWorld(t)
	body := w.must(t, w.member, "GET", w.proj()+"/roadmap?root="+w.payments.ID.String(), "", http.StatusOK)
	got := canWriteByTitle(t, body)
	if len(got) != 2 || !got["Payments"] || !got["Card"] {
		t.Fatalf("subtree of Payments: %s, want exactly Payments and Card", body)
	}
	w.must(t, w.member, "GET", w.proj()+"/roadmap?root="+uuid.NewString(), "", http.StatusNotFound)
	w.must(t, w.member, "GET", w.proj()+"/roadmap?root=not-a-uuid", "", http.StatusBadRequest)
}

type nodeReadView struct {
	Node      map[string]any `json:"node"`
	Ancestors []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"ancestors"`
	ProjectID   string `json:"projectId"`
	ProjectName string `json:"projectName"`
}

func TestOneNodeReadsWithItsAncestors(t *testing.T) {
	w := newTreeWorld(t)
	path := "/api/v1/roadmap/" + w.card.ID.String()
	got := decodeAs[nodeReadView](t, w.must(t, w.member, "GET", path, "", http.StatusOK))
	if got.Node["id"] != w.card.ID.String() || got.Node["canWrite"] != true {
		t.Errorf("node: %v, want Card with canWrite true", got.Node)
	}
	if len(got.Ancestors) != 2 || got.Ancestors[0].Title != "Root" || got.Ancestors[1].Title != "Payments" ||
		got.Ancestors[1].ID != w.payments.ID.String() {
		t.Errorf("ancestors: %+v, want Root then Payments", got.Ancestors)
	}
	if got.ProjectID != w.project.ID.String() || got.ProjectName != w.project.Title {
		t.Errorf("project: %s %q, want %s %q", got.ProjectID, got.ProjectName, w.project.ID, w.project.Title)
	}
	root := decodeAs[nodeReadView](t, w.must(t, w.member, "GET", "/api/v1/roadmap/"+w.root.ID.String(), "", http.StatusOK))
	if len(root.Ancestors) != 0 || root.Node["canWrite"] != false {
		t.Errorf("root read: ancestors %+v canWrite %v, want none and false", root.Ancestors, root.Node["canWrite"])
	}
	w.must(t, w.outsider, "GET", path, "", http.StatusNotFound)
	w.must(t, w.member, "GET", "/api/v1/roadmap/"+uuid.NewString(), "", http.StatusNotFound)
}

// A member who may not write a node is refused before the planner runs, not
// after a model call whose draft they could never confirm (TREEX-07).
func TestBreakingDownANodeYouCannotWriteIsRefusedUpFront(t *testing.T) {
	w := newTreeWorld(t)
	w.srv = NewServer(w.st, w.srv.tokens, &config.Config{PublicURL: "http://app.test"},
		slog.New(slog.NewTextHandler(io.Discard, nil)), nil, agent.NewTemplateRoadmapPlanner())
	w.h = w.srv.Handler()
	w.must(t, w.member, "POST", "/api/v1/roadmap/"+w.shipping.ID.String()+"/expand", "", http.StatusForbidden)
	w.must(t, w.member, "POST", "/api/v1/roadmap/"+w.payments.ID.String()+"/expand", "", http.StatusOK)
}

// The agent briefing for a focus lists only the focus subtree's leaves, and an
// unknown focus is refused instead of ignored (rv5-TREEX-05, TREEB-06).
func TestAgentContextKeepsToTheFocusSubtree(t *testing.T) {
	w := newTreeWorld(t)
	base := "/api/v1/agent/context?project_id=" + w.project.ID.String() + "&focus="
	body := w.must(t, w.member, "GET", base+w.payments.ID.String(), "", http.StatusOK)
	got := decodeAs[struct {
		Focus  map[string]any   `json:"focus"`
		Leaves []map[string]any `json:"leaves"`
	}](t, body)
	if len(got.Leaves) != 1 || got.Leaves[0]["id"] != w.card.ID.String() {
		t.Errorf("leaves under Payments: %v, want Card only", got.Leaves)
	}
	if path, _ := got.Focus["path"].([]any); len(path) != 1 || path[0] != "Root" {
		t.Errorf("focus path: %v, want [Root]", got.Focus["path"])
	}
	w.must(t, w.member, "GET", base+uuid.NewString(), "", http.StatusNotFound)
	w.must(t, w.member, "GET", base+"not-a-node", "", http.StatusNotFound)
}

// An agent token may not change who owns a node: ownership grants write rights,
// so it stays a human decision (owner decision 4, 2026-11-21). The agent
// proposes instead; its other edits still go through.
func TestAnAgentTokenCannotChangeANodesOwner(t *testing.T) {
	w := newTreeWorld(t)
	agentTok := w.asAgent(t, w.owner)
	patch := "/api/v1/roadmap/" + w.card.ID.String()
	for _, body := range []string{`{"ownerId":"` + w.member.ID.String() + `"}`, `{"ownerId":""}`, `{"title":"Card v2","ownerId":""}`} {
		status, resp := w.call(agentTok, "PATCH", patch, body)
		requireScopeRefusal(t, "agent PATCH "+body, status, resp)
	}
	n, err := w.st.GetRoadmapNode(context.Background(), w.owner.ID, w.card.ID)
	if err != nil || n.Title != "Card" || n.OwnerID != nil {
		t.Fatalf("after refused agent patches: %+v, err %v; want Card, no owner", n, err)
	}
	w.must(t, agentTok, "PATCH", patch, `{"title":"Card v2"}`, http.StatusOK)
	w.must(t, w.owner, "PATCH", patch, `{"ownerId":"`+w.member.ID.String()+`"}`, http.StatusOK)
}
