// Package sparkworld builds the Spark demo's people, tree and dependencies
// (scripts/demo/spark_story.py TREE and DEPENDENCIES, the branches the
// people-in-charge tests need) on a real store, for tests in any package.
package sparkworld

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// World is the Spark workspace: Maya created it and owns the project, Jo is
// a promoted admin, Leo, Priya and Sam are members.
type World struct {
	Maya, Leo, Priya, Sam, Jo *store.User
	Workspace                 *store.Workspace
	Project                   *store.Project
	nodes                     map[string]*store.RoadmapNode
}

// Node is the node titled title; the test fails when there is none.
func (w *World) Node(t testing.TB, title string) *store.RoadmapNode {
	t.Helper()
	n, ok := w.nodes[title]
	if !ok {
		t.Fatalf("spark world has no node %q", title)
	}
	return n
}

// node: title, owner key ("" none), status ("" pending), due in days (nil:
// none), children.
type node struct {
	title, owner, status string
	due                  *int
	children             []node
}

func days(d int) *int { return &d }

// tree is spark_story.py TREE, the branches and leaves the tests read.
var tree = node{title: "Spark MVP", children: []node{
	{title: "Onboarding & profiles", owner: "priya", children: []node{
		{title: "Profile builder", children: []node{
			{title: "Photo upload with crop", owner: "priya", status: "in_progress", due: days(3)},
			{title: "Prompt picker", due: days(12)},
		}},
		{title: "Selfie verification badge", due: days(30)},
	}},
	{title: "Matching", owner: "leo", children: []node{
		{title: "Prompt-based matching engine", owner: "sam", children: []node{
			{title: "Prompt answer embeddings", owner: "sam", status: "done", due: days(-4)},
			{title: "Compatibility scoring", owner: "sam", status: "in_progress", due: days(4)},
			{title: "Match explanation copy", due: days(18)},
		}},
		{title: "Daily match limit", owner: "sam", status: "in_progress", due: days(2)},
		{title: "Daily drop UI", due: days(9)},
		{title: "Matching quality dashboard", due: days(35)},
	}},
	{title: "Chat", owner: "leo", children: []node{
		{title: "Realtime messaging service", owner: "sam", status: "in_progress", due: days(-3)},
		{title: "Conversation starters from prompts", due: days(10)},
	}},
	{title: "Safety & trust", owner: "jo", children: []node{
		{title: "Report a member", children: []node{
			{title: "Moderator review queue", owner: "jo", status: "in_progress", due: days(-1)},
		}},
	}},
	{title: "Launch", children: []node{
		{title: "Closed beta in two cities", due: days(40)},
		{title: "App Store listing and screenshots", due: days(25)},
	}},
}}

// Dependencies is spark_story.py DEPENDENCIES: (waiting node, node it waits on).
var Dependencies = [][2]string{
	{"Closed beta in two cities", "Prompt-based matching engine"},
	{"Closed beta in two cities", "Realtime messaging service"},
	{"Closed beta in two cities", "Report a member"},
	{"Closed beta in two cities", "Profile builder"},
	{"App Store listing and screenshots", "Selfie verification badge"},
	{"Daily drop UI", "Compatibility scoring"},
	{"Conversation starters from prompts", "Prompt answer embeddings"},
	{"Matching quality dashboard", "Daily drop UI"},
}

// Build creates the Spark world on st. Maya writes every node and link, as
// the project owner; owners are set with each status so no leaf is claimed
// by her.
func Build(t testing.TB, st *store.Store) *World {
	t.Helper()
	ctx := context.Background()
	mk := func(name string) *store.User {
		u, err := st.CreateUser(ctx, "spark-"+uuid.NewString()+"@test.local", "x", name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		return u
	}
	w := &World{Maya: mk("Maya Chen"), Leo: mk("Leo Park"), Priya: mk("Priya Nair"), Sam: mk("Sam Okafor"),
		Jo: mk("Jo Rivera"), nodes: map[string]*store.RoadmapNode{}}
	var err error
	if w.Workspace, err = st.CreateWorkspace(ctx, w.Maya.ID, "Spark Labs", ""); err != nil {
		t.Fatalf("workspace: %v", err)
	}
	for _, u := range []*store.User{w.Leo, w.Priya, w.Sam, w.Jo} {
		if _, err := st.JoinWorkspace(ctx, u.ID, w.Workspace.Code); err != nil {
			t.Fatalf("join: %v", err)
		}
	}
	if err := st.UpdateMemberRole(ctx, w.Maya.ID, w.Workspace.ID, w.Jo.ID, "admin"); err != nil {
		t.Fatalf("promote Jo: %v", err)
	}
	if w.Project, err = st.CreateProject(ctx, w.Maya.ID, w.Workspace.ID, "Spark MVP", ""); err != nil {
		t.Fatalf("project: %v", err)
	}
	w.add(t, st, nil, tree)
	for _, d := range Dependencies {
		if _, err := st.AddRoadmapDependency(ctx, w.Maya.ID, w.Node(t, d[0]).ID, w.Node(t, d[1]).ID); err != nil {
			t.Fatalf("%s waits on %s: %v", d[0], d[1], err)
		}
	}
	return w
}

func (w *World) add(t testing.TB, st *store.Store, parent *uuid.UUID, n node) {
	t.Helper()
	ctx := context.Background()
	created, err := st.CreateRoadmapNode(ctx, w.Maya.ID, w.Project.ID, parent, n.title, "")
	if err != nil {
		t.Fatalf("create %s: %v", n.title, err)
	}
	w.nodes[n.title] = created
	for _, c := range n.children {
		w.add(t, st, &created.ID, c)
	}
	f := store.RoadmapNodeFields{}
	if n.owner != "" {
		owner := map[string]*store.User{"priya": w.Priya, "leo": w.Leo, "sam": w.Sam, "jo": w.Jo}[n.owner]
		f.OwnerID = &owner.ID
	}
	if n.status != "" {
		f.Status = &n.status
	}
	if n.status == "done" { // done is reached from in_progress, with a reason
		started, reason := "in_progress", "The work shipped on staging."
		w.set(t, st, created.ID, store.RoadmapNodeFields{Status: &started, OwnerID: f.OwnerID})
		f.CompletionReason = &reason
	}
	if n.due != nil {
		d := time.Now().Add(time.Duration(*n.due) * 24 * time.Hour)
		f.Deadline = &d
	}
	if f == (store.RoadmapNodeFields{}) {
		return
	}
	w.nodes[n.title] = w.set(t, st, created.ID, f)
}

func (w *World) set(t testing.TB, st *store.Store, id uuid.UUID, f store.RoadmapNodeFields) *store.RoadmapNode {
	t.Helper()
	n, err := st.UpdateRoadmapNode(context.Background(), w.Maya.ID, id, f)
	if err != nil {
		t.Fatalf("set node: %v", err)
	}
	return n
}
