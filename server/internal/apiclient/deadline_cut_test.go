package apiclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Review-4 REL-06, the clock side. The client has no timeout of its own any more:
// each request is cut by the deadline its route needs. These tests shrink the two
// deadlines to milliseconds and watch when calls really end.

// shrinkDeadlines sets the quick and the model deadline for the rest of the test.
func shrinkDeadlines(t *testing.T, quick, ai time.Duration) {
	t.Helper()
	oldQuick, oldAI := quickDeadline, aiDeadline
	quickDeadline, aiDeadline = quick, ai
	t.Cleanup(func() { quickDeadline, aiDeadline = oldQuick, oldAI })
}

// slowAPI answers every request with {} after delay; a client that leaves
// sooner frees the handler at once.
func slowAPI(t *testing.T, delay time.Duration) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-time.After(delay):
			w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	c := New(srv.URL + "/api/v1")
	c.SetToken("tok-session")
	return c
}

// wantCutAt requires err to be a deadline, and the call to have ended between
// min and max after it started.
func wantCutAt(t *testing.T, name string, start time.Time, err error, min, max time.Duration) {
	t.Helper()
	took := time.Since(start)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("%s: err = %v, want it to wrap context.DeadlineExceeded", name, err)
	}
	if took < min || took > max {
		t.Errorf("%s: ended after %v, want between %v and %v", name, took, min, max)
	}
}

func TestQuickCallIsCutAtTheQuickDeadline(t *testing.T) {
	shrinkDeadlines(t, 150*time.Millisecond, 5*time.Second)
	c := New(blockingAPI(t).URL + "/api/v1")
	c.SetToken("tok-session")

	start := time.Now()
	_, err := c.Whoami()

	wantCutAt(t, "whoami", start, err, 140*time.Millisecond, time.Second)
}

func TestModelCallOutlivesTheQuickDeadlineButNotItsOwn(t *testing.T) {
	shrinkDeadlines(t, 100*time.Millisecond, 700*time.Millisecond)

	t.Run("a draft the model takes a while on", func(t *testing.T) {
		c := slowAPI(t, 350*time.Millisecond) // past the quick deadline, inside the model's
		if _, err := c.Whoami(); !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("whoami: err = %v, want the quick deadline to cut a call as slow as the draft", err)
		}
		if _, err := c.BuildFromPrd("p1"); err != nil {
			t.Errorf("build: %v, want the draft to finish: it is within the model's deadline", err)
		}
	})

	t.Run("a server that never answers", func(t *testing.T) {
		c := New(blockingAPI(t).URL + "/api/v1")
		c.SetToken("tok-session")

		start := time.Now()
		_, err := c.BuildFromPrd("p1")

		wantCutAt(t, "build", start, err, 600*time.Millisecond, 3*time.Second)
	})
}

// The deadline covers the body: a server that sends its headers and then goes
// quiet does not hold the caller past it.
func TestDeadlineCoversTheResponseBody(t *testing.T) {
	shrinkDeadlines(t, 150*time.Millisecond, 5*time.Second)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":`))
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	t.Cleanup(srv.Close)
	c := New(srv.URL + "/api/v1")
	c.SetToken("tok-session")

	start := time.Now()
	_, err := c.Whoami()

	wantCutAt(t, "whoami", start, err, 140*time.Millisecond, time.Second)
}

func TestDeadlineForClassifiesByMethodAndRoute(t *testing.T) {
	for _, tc := range []struct {
		method, route string
		wantModel     bool
	}{
		{http.MethodGet, "/me", false},
		{http.MethodPost, "/prds/p1/build", true},
		{http.MethodPost, "/prds/a%2Fb/build", true}, // an escaped id stays one segment
		{http.MethodPost, "/prds/%2E%2E/build", true},
		{http.MethodGet, "/prds/p1/build", false},          // only the POST drafts
		{http.MethodPost, "/prds/p1/build/confirm", false}, // writes the tree the caller holds
		{http.MethodPost, "/prds/p1/build/", false},
		{http.MethodGet, "/prds/p1/export?format=markdown", false},
		{http.MethodPost, "/ideas/i1/convert", false}, // no model call on the server
	} {
		want := quickDeadline
		if tc.wantModel {
			want = aiDeadline
		}
		if got := deadlineFor(tc.method, tc.route); got != want {
			t.Errorf("%s %s: deadline %v, want %v", tc.method, tc.route, got, want)
		}
	}
}
