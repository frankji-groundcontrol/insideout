package apiclient

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// Review-4 REL-06. One 20 s total timeout on the shared http.Client cuts a call
// that waits on the model: POST /prds/{id}/build drafts the whole roadmap with
// the planner (PlanMVP), and a slow provider takes longer than that. A request
// needs a deadline that fits what its route does: short for a read, much longer
// for a model call, and always one, so a hung server can not hold the caller for
// ever. The tests look at the deadline each request is sent with.

// deadlineRecorder notes how long each request's context allows it to run when
// it is sent, then hands the request to the real transport.
type deadlineRecorder struct {
	next    http.RoundTripper
	mu      sync.Mutex
	allowed map[string]time.Duration // "METHOD /path" -> time left at send; 0: no deadline at all
}

func (d *deadlineRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	var left time.Duration
	if dl, ok := req.Context().Deadline(); ok {
		left = time.Until(dl)
	}
	d.mu.Lock()
	d.allowed[req.Method+" "+req.URL.Path] = left
	d.mu.Unlock()
	return d.next.RoundTrip(req)
}

// recordedClient is a client on a real server that answers every route at once,
// with every request it sends recorded.
func recordedClient(t *testing.T) (*Client, *deadlineRecorder) {
	t.Helper()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(api.Close)
	c := New(api.URL + "/api/v1")
	c.SetToken("tok-session")
	rec := &deadlineRecorder{next: c.hc.Transport, allowed: map[string]time.Duration{}}
	c.hc.Transport = rec
	return c, rec
}

// wantDeadline checks the deadline a recorded request was sent with: always one;
// a minute or less when the server answers from its database; more than a minute
// when the call waits on the model.
func wantDeadline(t *testing.T, rec *deadlineRecorder, route string, waitsOnModel bool) {
	t.Helper()
	rec.mu.Lock()
	left, sent := rec.allowed[route]
	rec.mu.Unlock()
	switch {
	case !sent:
		t.Errorf("%s was never sent", route)
	case left <= 0:
		t.Errorf("%s was sent with no deadline: a hung server would hold the caller for ever", route)
	case waitsOnModel && left <= time.Minute:
		t.Errorf("%s waits on the model but was allowed only %v: a slow provider is cut off mid-draft", route, left.Round(time.Second))
	case !waitsOnModel && left > time.Minute:
		t.Errorf("%s is answered from the database but was allowed %v: a hung server would hold the caller that long", route, left.Round(time.Second))
	}
}

func TestPlannerBuildOutlivesTheQuickDeadline(t *testing.T) {
	c, rec := recordedClient(t)
	if _, err := c.Whoami(); err != nil {
		t.Fatalf("whoami: %v", err)
	}
	if _, err := c.BuildFromPrd("p1"); err != nil {
		t.Fatalf("build: %v", err)
	}
	if _, err := c.ConfirmBuildFromPrd("p1", json.RawMessage(`{}`), 0); err != nil {
		t.Fatalf("confirm build: %v", err)
	}

	wantDeadline(t, rec, "GET /api/v1/me", false)                     // a read
	wantDeadline(t, rec, "POST /api/v1/prds/p1/build", true)          // drafts with the planner's model
	wantDeadline(t, rec, "POST /api/v1/prds/p1/build/confirm", false) // only writes the tree the caller holds
}
