package api

import (
	"net/http"
	"testing"
	"time"
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
