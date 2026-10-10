package mail

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ARCH-03: the admin mail probe surfaces LastSendError so a swallowed
// send failure is more than a log line. A failing send must record its
// error; a later successful send must clear it again.
func TestLastSendErrorTracksOutcome(t *testing.T) {
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer failing.Close()
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()

	s := NewSender("re-test-key", "noreply@example.com", testLogger(), false)
	s.client = ok.Client()

	if got := s.LastSendError(); got != "" {
		t.Fatalf("fresh sender: LastSendError %q, want empty", got)
	}

	s.endpoint = failing.URL
	if err := s.Send(context.Background(), "user@example.com", "Hello", "t", "h"); err == nil {
		t.Fatal("send against a failing provider must error")
	}
	if got := s.LastSendError(); !strings.Contains(got, "403") {
		t.Fatalf("after a failed send: LastSendError %q, want the provider status", got)
	}

	s.endpoint = ok.URL
	if err := s.Send(context.Background(), "user@example.com", "Hello", "t", "h"); err != nil {
		t.Fatalf("send against a working provider: %v", err)
	}
	if got := s.LastSendError(); got != "" {
		t.Fatalf("after a successful send: LastSendError %q, want empty", got)
	}
}
