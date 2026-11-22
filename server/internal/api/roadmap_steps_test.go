package api

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

// The ordered steps of one level (docs/architecture/roadmap-tree.md, "Steps")
// on the tree world, with Card waiting on Shipping: Root → {Payments
// (member's) → Card, Shipping}. Real PostgreSQL, the real router.

type stepView struct {
	Node         map[string]any               `json:"node"`
	Stage        int                          `json:"stage"`
	Loop         *int                         `json:"loop"`
	After        []string                     `json:"after"`
	Links        map[string]int               `json:"links"`
	ItemsInside  int                          `json:"itemsInside"`
	WaitsOutside []struct{ ID, Title string } `json:"waitsOutside"`
}

type stepsView struct {
	Root      map[string]any `json:"root"`
	Ancestors []struct {
		ID      string  `json:"id"`
		Title   string  `json:"title"`
		OwnerID *string `json:"ownerId"`
	} `json:"ancestors"`
	Steps []stepView `json:"steps"`
}

// stepTitles is each step as "title@stage".
func stepTitles(v stepsView) []string {
	out := []string{}
	for _, s := range v.Steps {
		out = append(out, s.Node["title"].(string)+"@"+string(rune('0'+s.Stage)))
	}
	return out
}

func (w *treeWorld) steps(t *testing.T, as routeUser, query string) stepsView {
	t.Helper()
	return decodeAs[stepsView](t, w.must(t, as, "GET", w.proj()+"/roadmap/steps"+query, "", http.StatusOK))
}

func TestTheStepsOfALevelComeInDependencyOrder(t *testing.T) {
	w := newTreeWorld(t)
	w.must(t, w.owner, "POST", nodePath(w.card.ID)+"/dependencies", depBody(w.shipping.ID), http.StatusCreated)

	// The project's one root opens itself: its children are the steps.
	top := w.steps(t, w.member, "")
	if top.Root["id"] != w.root.ID.String() || len(top.Ancestors) != 0 {
		t.Fatalf("top root = %v ancestors %v, want Root and none", top.Root["id"], top.Ancestors)
	}
	if got := stepTitles(top); !reflect.DeepEqual(got, []string{"Shipping@0", "Payments@1"}) {
		t.Fatalf("top steps = %v, want Shipping then Payments", got)
	}
	pay := top.Steps[1]
	if !reflect.DeepEqual(pay.After, []string{w.shipping.ID.String()}) || pay.Links[w.shipping.ID.String()] != 1 ||
		pay.ItemsInside != 1 || pay.Loop != nil || len(pay.WaitsOutside) != 0 || pay.Node["canWrite"] != true {
		t.Errorf("Payments step = %+v, want after Shipping by one link over one item, writable by its owner", pay)
	}

	// Opening Payments: its one step waits on Shipping, outside the level.
	in := w.steps(t, w.member, "?root="+w.payments.ID.String())
	if in.Root["id"] != w.payments.ID.String() || len(in.Ancestors) != 1 || in.Ancestors[0].Title != "Root" || in.Ancestors[0].OwnerID != nil {
		t.Fatalf("Payments root = %v ancestors %+v, want Payments under an unowned Root", in.Root["id"], in.Ancestors)
	}
	if len(in.Steps) != 1 || in.Steps[0].Node["id"] != w.card.ID.String() ||
		len(in.Steps[0].WaitsOutside) != 1 || in.Steps[0].WaitsOutside[0].Title != "Shipping" {
		t.Fatalf("Payments steps = %+v, want Card waiting outside on Shipping", in.Steps)
	}
	// A leaf is its own step.
	if leaf := w.steps(t, w.member, "?root="+w.card.ID.String()); len(leaf.Steps) != 1 || leaf.Steps[0].Node["id"] != w.card.ID.String() {
		t.Errorf("Card steps = %+v, want Card itself", leaf.Steps)
	}

	// Two roots: no root, the roots are the steps.
	if _, err := w.st.CreateRoadmapNode(t.Context(), w.owner.ID, w.project.ID, nil, "Later", ""); err != nil {
		t.Fatal(err)
	}
	if two := w.steps(t, w.member, ""); two.Root != nil || !reflect.DeepEqual(stepTitles(two), []string{"Root@0", "Later@0"}) {
		t.Errorf("two roots = %v %v, want no root and both roots", two.Root, stepTitles(two))
	}

	steps := w.proj() + "/roadmap/steps"
	w.must(t, w.member, "GET", steps+"?root=nope", "", http.StatusBadRequest)
	w.must(t, w.member, "GET", steps+"?root="+uuid.NewString(), "", http.StatusNotFound)
	w.must(t, w.outsider, "GET", steps, "", http.StatusNotFound)
}

// A root-scoped agent token opens its root by default and may not open a
// level outside it; its list and Progress default to its root too.
func TestARootScopedAgentTokenReadsItsOwnLevel(t *testing.T) {
	w := newTreeWorld(t)
	agent := w.rootAgent(t, w.member, w.payments.ID)

	if got := w.steps(t, agent, ""); got.Root["id"] != w.payments.ID.String() || len(got.Steps) != 1 {
		t.Errorf("agent default steps: root %v steps %d, want Payments with Card", got.Root["id"], len(got.Steps))
	}
	w.steps(t, agent, "?root="+w.card.ID.String())
	status, body := w.call(agent, "GET", w.proj()+"/roadmap/steps?root="+w.shipping.ID.String(), "")
	requireScopeRefusal(t, "steps outside the root", status, body)

	if list := canWriteByTitle(t, w.must(t, agent, "GET", w.proj()+"/roadmap", "", http.StatusOK)); len(list) != 2 || !list["Card"] {
		t.Errorf("agent default list = %v, want Payments and Card", list)
	}
	// Shipping, outside the root, is next in the project but not in the agent's Progress.
	w.must(t, w.owner, "PATCH", nodePath(w.shipping.ID), `{"deadline":"2099-01-01T00:00:00Z"}`, http.StatusOK)
	whole := w.must(t, w.member, "GET", w.proj()+"/roadmap/progress", "", http.StatusOK)
	sub := w.must(t, w.member, "GET", w.proj()+"/roadmap/progress?root="+w.payments.ID.String(), "", http.StatusOK)
	if got := w.must(t, agent, "GET", w.proj()+"/roadmap/progress", "", http.StatusOK); got != sub || got == whole {
		t.Errorf("agent default progress = %s, want the Payments subtree's %s, not the project's", got, sub)
	}
	// Reads stay workspace-wide with an explicit root, and a token without a
	// root keeps the whole project.
	w.must(t, agent, "GET", w.proj()+"/roadmap?root="+w.shipping.ID.String(), "", http.StatusOK)
	if all := canWriteByTitle(t, w.must(t, w.asAgent(t, w.member), "GET", w.proj()+"/roadmap", "", http.StatusOK)); len(all) != 4 {
		t.Errorf("unscoped agent list = %v, want all four nodes", all)
	}
}
