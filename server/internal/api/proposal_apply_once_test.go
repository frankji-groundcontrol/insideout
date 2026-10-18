package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// rv4-FEAT-02: accepting a proposal with apply:true creates its roadmap nodes
// once, all or nothing, and says so when it cannot. Real PostgreSQL through the
// whole handler chain (DATABASE_URL); the failure case also needs
// DATABASE_OWNER_URL, to make the database refuse one insert for real.

const acceptAndApply = `{"decision":"accepted","apply":true}`

// proposeItems files a proposal carrying one add_node item per title, as the
// project owner, and returns the proposal's update id.
func (w *routeWorld) proposeItems(t *testing.T, titles ...string) string {
	t.Helper()
	items := make([]string, len(titles))
	for i, title := range titles {
		items[i] = `{"title":"` + title + `"}`
	}
	status, body := w.call(w.owner, http.MethodPost, "/api/v1/agent/propose",
		`{"projectId":"`+w.project.ID.String()+`","kind":"structure","summary":"apply once","items":[`+strings.Join(items, ",")+`]}`)
	expect(t, "owner proposes", status, body, http.StatusCreated)
	return decodeAs[struct{ ID string }](t, body).ID
}

// countTitled is how many roadmap nodes carry the title.
func (w *routeWorld) countTitled(t *testing.T, title string) int {
	t.Helper()
	nodes, err := w.st.ListRoadmap(context.Background(), w.owner.ID, w.project.ID)
	if err != nil {
		t.Fatalf("list roadmap: %v", err)
	}
	n := 0
	for _, node := range nodes {
		if node.Title == title {
			n++
		}
	}
	return n
}

// decisionOf is the recorded decision of a proposal ("" while it is open).
func (w *routeWorld) decisionOf(t *testing.T, updateID string) string {
	t.Helper()
	updates, err := w.st.ListProjectUpdates(context.Background(), w.owner.ID, w.project.ID, 100, nil)
	if err != nil {
		t.Fatalf("list updates: %v", err)
	}
	for _, u := range updates {
		if u.ID.String() == updateID {
			return u.Decision
		}
	}
	t.Fatalf("proposal %s is not in the project timeline", updateID)
	return ""
}

func appliedCount(t *testing.T, body string) int {
	t.Helper()
	if strings.Contains(body, `"appliedNodes":null`) {
		t.Fatalf("answered %s, want appliedNodes to be an empty list, not null", body)
	}
	return len(decodeAs[struct{ AppliedNodes []string }](t, body).AppliedNodes)
}

func TestAcceptingTwiceAppliesTheProposalOnce(t *testing.T) {
	w := newRouteWorld(t)
	a, b := "Alpha "+uuid.NewString()[:8], "Beta "+uuid.NewString()[:8]
	id := w.proposeItems(t, a, b)

	status, body := w.decide(w.owner, id, acceptAndApply)
	expect(t, "first accept", status, body, http.StatusOK)
	if n := appliedCount(t, body); n != 2 {
		t.Fatalf("first accept applied %d nodes, want 2; body %s", n, body)
	}
	status, body = w.decide(w.owner, id, acceptAndApply)
	expect(t, "second accept", status, body, http.StatusOK)
	if n := appliedCount(t, body); n != 0 {
		t.Fatalf("second accept applied %d nodes, want 0 (the items were already applied); body %s", n, body)
	}
	for _, title := range []string{a, b} {
		if n := w.countTitled(t, title); n != 1 {
			t.Fatalf("the roadmap has %d nodes titled %q, want 1", n, title)
		}
	}
}

func TestConcurrentAcceptsApplyTheProposalOnce(t *testing.T) {
	w := newRouteWorld(t)
	title := "Raced " + uuid.NewString()[:8]
	id := w.proposeItems(t, title)

	const racers = 4
	type answer struct {
		status int
		body   string
	}
	gate := make(chan struct{})
	answers := make(chan answer, racers)
	for i := 0; i < racers; i++ {
		go func() {
			<-gate
			status, body := w.decide(w.owner, id, acceptAndApply)
			answers <- answer{status, body}
		}()
	}
	close(gate)
	applied := 0
	for i := 0; i < racers; i++ {
		a := <-answers
		if a.status != http.StatusOK {
			t.Errorf("a racing accept answered %d, want 200; body %s", a.status, a.body)
			continue
		}
		applied += appliedCount(t, a.body)
	}
	if applied != 1 {
		t.Errorf("%d racing accepts applied %d nodes in total, want 1", racers, applied)
	}
	if n := w.countTitled(t, title); n != 1 {
		t.Errorf("the roadmap has %d nodes titled %q, want 1", n, title)
	}
}

// poisonRoadmapTitle makes the database refuse one exact node title (see the
// store package's twin of this helper). The title must be hex and hyphens
// only: DDL takes no bind parameters.
func poisonRoadmapTitle(t *testing.T, title string) (lift func()) {
	t.Helper()
	ownerURL := os.Getenv("DATABASE_OWNER_URL")
	if ownerURL == "" {
		t.Skip("DATABASE_OWNER_URL not set, skipping the failure-injection check")
	}
	ctx := context.Background()
	owner, err := store.Open(ctx, ownerURL)
	if err != nil {
		t.Fatalf("open owner store: %v", err)
	}
	t.Cleanup(owner.Close)
	name := "rv4_poison_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := owner.Pool.Exec(ctx, fmt.Sprintf(
		`ALTER TABLE insideout.roadmap_nodes ADD CONSTRAINT %s CHECK (title <> '%s') NOT VALID`, name, title)); err != nil {
		t.Fatalf("poison title: %v", err)
	}
	lift = func() {
		if _, err := owner.Pool.Exec(ctx, `ALTER TABLE insideout.roadmap_nodes DROP CONSTRAINT IF EXISTS `+name); err != nil {
			t.Errorf("lift poison: %v", err)
		}
	}
	t.Cleanup(lift)
	return lift
}

func TestAcceptApplyFailureIsAServerErrorAndChangesNothing(t *testing.T) {
	w := newRouteWorld(t)
	poison := "poison-" + uuid.NewString()
	first := "First " + uuid.NewString()[:8]
	lift := poisonRoadmapTitle(t, poison)
	id := w.proposeItems(t, first, poison)

	status, body := w.decide(w.owner, id, acceptAndApply)
	expect(t, "accept while the database refuses item 2", status, body, http.StatusInternalServerError)
	if n := w.countTitled(t, first); n != 0 {
		t.Fatalf("the failed accept left %d node(s) titled %q; apply is all or nothing", n, first)
	}
	if d := w.decisionOf(t, id); d != "" {
		t.Fatalf("the failed accept recorded the decision %q; a 500 must change nothing", d)
	}

	lift()
	status, body = w.decide(w.owner, id, acceptAndApply)
	expect(t, "accept again once the cause is gone", status, body, http.StatusOK)
	if n := appliedCount(t, body); n != 2 {
		t.Fatalf("the retry applied %d nodes, want both; body %s", n, body)
	}
	for _, title := range []string{first, poison} {
		if n := w.countTitled(t, title); n != 1 {
			t.Fatalf("the roadmap has %d nodes titled %q, want 1", n, title)
		}
	}
}
