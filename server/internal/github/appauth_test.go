package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// guideServer serves a raw guide body with the given status.
func guideServer(status int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

// A guide at or under the 1 MiB cap comes back whole; one byte over is
// rejected. Guards the cap's boundary while it moves from a
// buffer-after-the-fact check to a streaming one.
func TestFetchGuideFileCapBoundary(t *testing.T) {
	const cap = 1 << 20
	at := guideServer(http.StatusOK, strings.Repeat("x", cap))
	defer at.Close()
	apiBaseURL = at.URL
	defer func() { apiBaseURL = "https://api.github.com" }()

	got, err := FetchGuideFile(context.Background(), "", "own", "repo", "main")
	if err != nil || len(got) != cap {
		t.Fatalf("at cap: got %d bytes, err %v; want %d bytes, no error", len(got), err, cap)
	}

	over := guideServer(http.StatusOK, strings.Repeat("x", cap+1))
	defer over.Close()
	apiBaseURL = over.URL

	if _, err := FetchGuideFile(context.Background(), "", "own", "repo", "main"); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("over cap: want guide-too-large error, got %v", err)
	}
}

// SEC-02: the cap must fire while the body streams in, not after the whole
// body has been buffered. The upstream here never finishes its 8 MiB write
// unless the client keeps reading; a FetchGuideFile that abandons the
// response at the cap makes the upstream write fail (client gone), while
// one that buffers everything first lets the write complete cleanly.
func TestFetchGuideFileStopsReadingAtTheCap(t *testing.T) {
	const total = 8 << 20
	var abandoned atomic.Bool
	done := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		chunk := make([]byte, 64<<10)
		for written := 0; written < total; written += len(chunk) {
			if _, err := w.Write(chunk); err != nil || r.Context().Err() != nil {
				abandoned.Store(true)
				return
			}
		}
	}))
	defer srv.Close()
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = "https://api.github.com" }()

	if _, err := FetchGuideFile(context.Background(), "", "own", "repo", "main"); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("want guide-too-large error, got %v", err)
	}
	select {
	case <-done:
		if !abandoned.Load() {
			t.Fatalf("upstream wrote all %d MiB before the cap fired — the whole body was buffered first", total>>20)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("upstream write neither finished nor failed within 10s")
	}
}
