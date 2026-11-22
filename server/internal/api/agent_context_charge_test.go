package api

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// People-in-charge mode (2026-11-22): with no focus, a briefing opens on what
// its human is in charge of. On the tree world with Card waiting on Shipping,
// the project owner (Maya in the Spark story) is in charge of the unowned
// Root, the member (Sam) of Payments. Real PostgreSQL, the real router.

type briefing struct {
	Focus    map[string]any      `json:"focus"`
	InCharge []map[string]string `json:"inCharge"`
	Leaves   []map[string]any    `json:"leaves"`
}

func (w *treeWorld) brief(t *testing.T, as routeUser, focus string) briefing {
	t.Helper()
	path := "/api/v1/agent/context?project_id=" + w.project.ID.String()
	if focus != "" {
		path += "&focus=" + focus
	}
	return decodeAs[briefing](t, w.must(t, as, "GET", path, "", http.StatusOK))
}

func leafTitles(b briefing) []string {
	out := []string{}
	for _, l := range b.Leaves {
		out = append(out, l["title"].(string))
	}
	return out
}

func TestABriefingOpensOnWhatItsHumanIsInChargeOf(t *testing.T) {
	w := newTreeWorld(t)
	w.must(t, w.owner, "POST", nodePath(w.card.ID)+"/dependencies", depBody(w.shipping.ID), http.StatusCreated)

	// The project owner: the unowned Root, with its steps in order.
	maya := w.brief(t, w.owner, "")
	if maya.Focus["id"] != w.root.ID.String() || maya.InCharge != nil {
		t.Fatalf("owner focus = %v inCharge %v, want Root", maya.Focus["id"], maya.InCharge)
	}
	steps, _ := maya.Focus["steps"].([]any)
	if len(steps) != 2 || steps[0].(map[string]any)["title"] != "Shipping" || steps[1].(map[string]any)["stage"] != float64(1) {
		t.Errorf("owner focus steps = %v, want Shipping then Payments at stage 1", maya.Focus["steps"])
	}

	// The member, human or agent token: Payments, waiting on Shipping outside it.
	for _, sam := range []routeUser{w.member, w.asAgent(t, w.member)} {
		got := w.brief(t, sam, "")
		if got.Focus["id"] != w.payments.ID.String() || !reflect.DeepEqual(got.Focus["waitsOn"], []any{"Shipping"}) ||
			!reflect.DeepEqual(leafTitles(got), []string{"Card"}) {
			t.Errorf("member briefing = focus %v waitsOn %v leaves %v, want Payments waiting on Shipping over Card",
				got.Focus["id"], got.Focus["waitsOn"], leafTitles(got))
		}
	}
	// Shipping blocks Card, outside it.
	if got := w.brief(t, w.owner, w.shipping.ID.String()); !reflect.DeepEqual(got.Focus["blocks"], []any{"Card"}) {
		t.Errorf("Shipping blocks %v, want [Card]", got.Focus["blocks"])
	}

	// A workspace admin who is not the project owner is in charge of the Root too.
	admin := w.newMember(t, "admin")
	if err := w.st.UpdateMemberRole(t.Context(), w.owner.ID, w.project.WorkspaceID, admin.ID, "admin"); err != nil {
		t.Fatal(err)
	}
	if got := w.brief(t, admin, ""); got.Focus["id"] != w.root.ID.String() {
		t.Errorf("admin focus = %v, want Root", got.Focus["id"])
	}

	// Nothing in charge: the whole project, as before.
	bystander := w.newMember(t, "bystander")
	if got := w.brief(t, bystander, ""); got.Focus != nil || got.InCharge != nil || len(got.Leaves) != 2 {
		t.Errorf("bystander briefing = focus %v inCharge %v leaves %v, want the whole project", got.Focus, got.InCharge, leafTitles(got))
	}

	// Two branches: both named, their leaves only, no focus.
	if _, err := w.st.UpdateRoadmapNode(t.Context(), w.owner.ID, w.shipping.ID, store.RoadmapNodeFields{OwnerID: &w.member.ID}); err != nil {
		t.Fatal(err)
	}
	two := w.brief(t, w.member, "")
	want := []map[string]string{{"id": w.payments.ID.String(), "title": "Payments"}, {"id": w.shipping.ID.String(), "title": "Shipping"}}
	if two.Focus != nil || !reflect.DeepEqual(two.InCharge, want) || len(two.Leaves) != 2 {
		t.Errorf("two branches = focus %v inCharge %v leaves %v, want both named and their two leaves", two.Focus, two.InCharge, leafTitles(two))
	}
}
