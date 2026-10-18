package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/roadmaptime"
)

// pinClock makes every view this server renders measure against at, for the
// rest of the test (L1, ARCH-04, QUAL-05).
func (w *routeWorld) pinClock(at time.Time) {
	w.srv.clock = func() time.Time { return at }
}

// A Server built without a clock (a test literal) reads the wall clock; one
// with a clock reads that.
func TestServerNowDefaultsToTheWallClock(t *testing.T) {
	before := time.Now()
	got := (&Server{}).now()
	if got.Before(before) || got.After(time.Now()) {
		t.Fatalf("an unpinned server read %v, want the wall clock", got)
	}
	pinned := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	if got := (&Server{clock: func() time.Time { return pinned }}).now(); !got.Equal(pinned) {
		t.Fatalf("a pinned server read %v, want %v", got, pinned)
	}
}

// L1: the quiet flag in the project responses follows the server clock. The
// project was created just now (database time), so the instants are taken
// from its own createdAt and the test holds on any real date: exactly 14 days
// after creation is not quiet, one second more is.
func TestProjectRoutesFlaggedFollowTheServerClock(t *testing.T) {
	w := newRouteWorld(t)
	created := w.project.CreatedAt
	listPath := "/api/v1/workspaces/" + w.project.WorkspaceID.String() + "/projects"
	getPath := "/api/v1/projects/" + w.project.ID.String()

	flagged := func(at time.Time) (list, detail bool) {
		w.pinClock(at)
		code, body := w.call(w.owner, "GET", listPath, "")
		expect(t, "list projects", code, body, http.StatusOK)
		found := false
		for _, p := range decodeAs[[]projectView](t, body) {
			if p.ID == w.project.ID.String() {
				list, found = p.Flagged, true
			}
		}
		if !found {
			t.Fatalf("project %s missing from the board list: %s", w.project.ID, body)
		}
		code, body = w.call(w.owner, "GET", getPath, "")
		expect(t, "get project", code, body, http.StatusOK)
		return list, decodeAs[projectView](t, body).Flagged
	}

	if list, detail := flagged(created.Add(quietAfter)); list || detail {
		t.Fatalf("14 days after creation: list flagged=%v detail flagged=%v, want both false", list, detail)
	}
	if list, detail := flagged(created.Add(quietAfter + time.Second)); !list || !detail {
		t.Fatalf("14 days and a second after creation: list flagged=%v detail flagged=%v, want both true", list, detail)
	}
}

// roadmapDeadline is an arbitrary fixed deadline. The roadmap tests move the
// server clock around it instead of waiting for a real date.
var roadmapDeadline = time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)

// pressureAt pins the clock at each escalation state PRODUCT.md names, by its
// offset from roadmapDeadline.
var pressureAt = []struct {
	name   string
	offset time.Duration
	want   string
}{
	{"a month out is normal", -30 * 24 * time.Hour, roadmaptime.Normal},
	{"seven days out is near", -7 * 24 * time.Hour, roadmaptime.Near},
	{"two days out is high risk", -48 * time.Hour, roadmaptime.HighRisk},
	{"a second past is overdue", time.Second, roadmaptime.Overdue},
}

// roadmapNode creates a root node over the API as the owner, then applies the
// PATCH bodies in order. It returns the node id.
func (w *routeWorld) roadmapNode(t *testing.T, title string, patches ...string) string {
	t.Helper()
	code, body := w.call(w.owner, "POST", "/api/v1/projects/"+w.project.ID.String()+"/roadmap", `{"title":"`+title+`"}`)
	expect(t, "create "+title, code, body, http.StatusCreated)
	id := decodeAs[roadmapNodeView](t, body).ID
	for _, patch := range patches {
		code, body = w.call(w.owner, "PATCH", "/api/v1/roadmap/"+id, patch)
		expect(t, "patch "+title, code, body, http.StatusOK)
	}
	return id
}

func roadmapDeadlineJSON() string {
	return `"deadline":"` + roadmapDeadline.Format(time.RFC3339) + `"`
}

// QUAL-05: a node's deadline pressure, in every response that renders the
// node, follows the server clock.
func TestRoadmapNodePressureFollowsTheServerClock(t *testing.T) {
	w := newRouteWorld(t)
	id := w.roadmapNode(t, "Ship the pilot")
	list := "/api/v1/projects/" + w.project.ID.String() + "/roadmap"

	for _, tc := range pressureAt {
		t.Run(tc.name, func(t *testing.T) {
			w.pinClock(roadmapDeadline.Add(tc.offset))

			code, body := w.call(w.owner, "PATCH", "/api/v1/roadmap/"+id, `{`+roadmapDeadlineJSON()+`}`)
			expect(t, "set deadline", code, body, http.StatusOK)
			if got := decodeAs[roadmapNodeView](t, body).Pressure; got != tc.want {
				t.Errorf("PATCH response pressure = %q, want %q", got, tc.want)
			}

			code, body = w.call(w.owner, "GET", list, "")
			expect(t, "list roadmap", code, body, http.StatusOK)
			listed := ""
			for _, n := range decodeAs[[]roadmapNodeView](t, body) {
				if n.ID == id {
					listed = n.Pressure
				}
			}
			if listed != tc.want {
				t.Errorf("list pressure = %q, want %q", listed, tc.want)
			}

			code, body = w.call(w.owner, "POST", "/api/v1/roadmap/"+id+"/move", `{"parentId":null,"position":0}`)
			expect(t, "move node", code, body, http.StatusOK)
			if got := decodeAs[roadmapNodeView](t, body).Pressure; got != tc.want {
				t.Errorf("move response pressure = %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("the validate response follows the clock too", func(t *testing.T) {
		vid := w.roadmapNode(t, "Prove the pilot", `{"status":"in_progress",`+roadmapDeadlineJSON()+`}`, `{"status":"done"}`)
		w.pinClock(roadmapDeadline.Add(-30 * 24 * time.Hour))
		code, body := w.call(w.owner, "POST", "/api/v1/roadmap/"+vid+"/validate", `{"reason":"pilot users finished the tour"}`)
		expect(t, "validate", code, body, http.StatusOK)
		if got := decodeAs[roadmapNodeView](t, body).Pressure; got != roadmaptime.Normal {
			t.Errorf("validate response pressure = %q, want %q", got, roadmaptime.Normal)
		}
	})
}

// QUAL-05: the Progress view assembles against the server clock, so the
// pressure on its Now, Next and Done items moves with it.
func TestRoadmapProgressPressureFollowsTheServerClock(t *testing.T) {
	w := newRouteWorld(t)
	w.roadmapNode(t, "Building now", `{"status":"in_progress",`+roadmapDeadlineJSON()+`}`)
	w.roadmapNode(t, "Up next", `{`+roadmapDeadlineJSON()+`}`)
	w.roadmapNode(t, "Finished", `{"status":"in_progress",`+roadmapDeadlineJSON()+`}`, `{"status":"done"}`)
	path := "/api/v1/projects/" + w.project.ID.String() + "/roadmap/progress"

	for _, tc := range pressureAt {
		t.Run(tc.name, func(t *testing.T) {
			w.pinClock(roadmapDeadline.Add(tc.offset))
			code, body := w.call(w.owner, "GET", path, "")
			expect(t, "progress", code, body, http.StatusOK)
			p := decodeAs[roadmaptime.Progress](t, body)
			if len(p.Now) != 1 || len(p.Next) != 1 || len(p.DoneItems) != 1 {
				t.Fatalf("now/next/done = %d/%d/%d items, want one each: %s", len(p.Now), len(p.Next), len(p.DoneItems), body)
			}
			for _, lane := range []struct {
				name string
				item roadmaptime.ProgressItem
			}{{"now", p.Now[0]}, {"next", p.Next[0]}, {"done", p.DoneItems[0]}} {
				if lane.item.Pressure != tc.want {
					t.Errorf("%s pressure = %q, want %q", lane.name, lane.item.Pressure, tc.want)
				}
			}
		})
	}
}
