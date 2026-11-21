package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// rv5 row 2.7 (TREEB-08): every node view carries a roll-up over the leaves
// below it, counted by the same leaf rule as Progress (a validated leaf is
// finished work), and Progress reads one subtree. Real PostgreSQL.

type rollupView struct {
	Leaves           int     `json:"leaves"`
	Done             int     `json:"done"`
	Validated        int     `json:"validated"`
	EarliestDeadline *string `json:"earliestDeadline"`
}

// rollupWorld deepens the tree world: Card gets two leaves, Card A done and
// Card B validated, and Shipping (a leaf) a deadline. Root → Payments → Card
// → {Card A, Card B}; Root → Shipping.
func newRollupWorld(t *testing.T) (*treeWorld, uuid.UUID, string) {
	t.Helper()
	w := newTreeWorld(t)
	add := func(title string) uuid.UUID {
		t.Helper()
		body := w.must(t, w.owner, "POST", w.proj()+"/roadmap", `{"parentId":"`+w.card.ID.String()+`","title":`+js(title)+`}`, http.StatusCreated)
		return uuid.MustParse(decodeAs[map[string]any](t, body)["id"].(string))
	}
	finish := func(id uuid.UUID) {
		t.Helper()
		for _, step := range []string{`{"status":"pending"}`, `{"status":"in_progress"}`, `{"status":"done","completionReason":"It works end to end."}`} {
			w.must(t, w.owner, "PATCH", nodePath(id), step, http.StatusOK)
		}
	}
	a, b := add("Card A"), add("Card B")
	finish(a)
	finish(b)
	w.must(t, w.owner, "POST", nodePath(b)+"/validate", `{"reason":"Seen in production."}`, http.StatusOK)
	deadline := time.Now().Add(72 * time.Hour).UTC().Truncate(time.Second).Format(time.RFC3339)
	w.must(t, w.owner, "PATCH", nodePath(w.shipping.ID), `{"deadline":"`+deadline+`"}`, http.StatusOK)
	return w, a, deadline
}

func rollupOf(t *testing.T, n map[string]any) rollupView {
	t.Helper()
	raw, ok := n["rollup"].(map[string]any)
	if !ok {
		t.Fatalf("node %v has no rollup", n["title"])
	}
	v := rollupView{Leaves: int(raw["leaves"].(float64)), Done: int(raw["done"].(float64)), Validated: int(raw["validated"].(float64))}
	if d, ok := raw["earliestDeadline"].(string); ok {
		v.EarliestDeadline = &d
	}
	return v
}

func sameInstant(p *string, want string) bool {
	if p == nil {
		return false
	}
	a, err1 := time.Parse(time.RFC3339, *p)
	b, err2 := time.Parse(time.RFC3339, want)
	return err1 == nil && err2 == nil && a.Equal(b)
}

func TestEveryNodeViewRollsUpTheLeavesBelowIt(t *testing.T) {
	w, cardA, deadline := newRollupWorld(t)
	byTitle := map[string]rollupView{}
	for _, n := range decodeAs[[]map[string]any](t, w.must(t, w.member, "GET", w.proj()+"/roadmap", "", http.StatusOK)) {
		byTitle[n["title"].(string)] = rollupOf(t, n)
	}
	root := byTitle["Root"]
	if root.Leaves != 3 || root.Done != 2 || root.Validated != 1 || !sameInstant(root.EarliestDeadline, deadline) {
		t.Errorf("Root rollup: %+v, want 3 leaves, 2 done, 1 validated, earliest %s", root, deadline)
	}
	if p := byTitle["Payments"]; p.Leaves != 2 || p.Done != 2 || p.Validated != 1 || p.EarliestDeadline != nil {
		t.Errorf("Payments rollup: %+v, want 2 leaves two levels down, both finished, no open deadline", p)
	}
	if s := byTitle["Shipping"]; s.Leaves != 1 || s.Done != 0 || !sameInstant(s.EarliestDeadline, deadline) {
		t.Errorf("Shipping (a leaf) rollup: %+v, want itself", s)
	}
	if a := byTitle["Card A"]; a.Leaves != 1 || a.Done != 1 || a.Validated != 0 {
		t.Errorf("Card A rollup: %+v, want itself, done", a)
	}

	one := decodeAs[nodeReadView](t, w.must(t, w.member, "GET", nodePath(w.payments.ID), "", http.StatusOK))
	if r := rollupOf(t, one.Node); r.Leaves != 2 || r.Done != 2 {
		t.Errorf("single read of Payments: rollup %+v, want 2 leaves, 2 done", r)
	}
	edited := decodeAs[map[string]any](t, w.must(t, w.member, "PATCH", nodePath(w.payments.ID), `{"title":"Payments v2"}`, http.StatusOK))
	if r := rollupOf(t, edited); r.Leaves != 2 || r.Done != 2 {
		t.Errorf("PATCH response for Payments: rollup %+v, want 2 leaves, 2 done", r)
	}
	moved := decodeAs[map[string]any](t, w.must(t, w.member, "POST", nodePath(cardA)+"/move", `{"parentId":"`+w.payments.ID.String()+`","position":0}`, http.StatusOK))
	if r := rollupOf(t, moved); r.Leaves != 1 || r.Done != 1 || moved["canWrite"] != true {
		t.Errorf("move response: rollup %+v canWrite %v, want itself and true", r, moved["canWrite"])
	}
	created := decodeAs[map[string]any](t, w.must(t, w.owner, "POST", w.proj()+"/roadmap", `{"parentId":"`+w.shipping.ID.String()+`","title":"Labels"}`, http.StatusCreated))
	if r := rollupOf(t, created); r.Leaves != 1 || r.Done != 0 || created["creatorName"] != w.owner.Username {
		t.Errorf("create response: rollup %+v creatorName %v, want itself and the owner", r, created["creatorName"])
	}
}

func TestProgressReadsOneSubtree(t *testing.T) {
	w, _, _ := newRollupWorld(t)
	type progress struct {
		Now       []map[string]any `json:"now"`
		Next      []map[string]any `json:"next"`
		DoneCount int              `json:"doneCount"`
	}
	base := w.proj() + "/roadmap/progress"
	all := decodeAs[progress](t, w.must(t, w.member, "GET", base, "", http.StatusOK))
	if all.DoneCount != 2 || len(all.Next) != 1 {
		t.Fatalf("whole project progress: %+v, want 2 done and Shipping next", all)
	}
	sub := decodeAs[progress](t, w.must(t, w.member, "GET", base+"?root="+w.payments.ID.String(), "", http.StatusOK))
	if sub.DoneCount != 2 || len(sub.Next) != 0 {
		t.Errorf("Payments progress: %+v, want 2 done and nothing next", sub)
	}
	ship := decodeAs[progress](t, w.must(t, w.member, "GET", base+"?root="+w.shipping.ID.String(), "", http.StatusOK))
	if ship.DoneCount != 0 || len(ship.Next) != 1 {
		t.Errorf("Shipping progress: %+v, want Shipping next and nothing done", ship)
	}
	w.must(t, w.member, "GET", base+"?root="+uuid.NewString(), "", http.StatusNotFound)
}
