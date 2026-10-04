package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSendPostsResendPayload(t *testing.T) {
	var gotAuth, gotFrom, gotTo, gotSubject, gotText string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		gotFrom, gotTo, gotSubject, gotText = body["from"], body["to"], body["subject"], body["text"]
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewSender("re-test-key", "InsideOut <noreply@example.com>", testLogger(), false)
	s.client = srv.Client()
	// Point the sender at the test server instead of the real API.
	s.endpoint = srv.URL

	if err := s.Send(context.Background(), "user@example.com", "Hello", "the link: http://x/l?t=1", "<p>the link</p>"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if gotAuth != "Bearer re-test-key" {
		t.Fatalf("auth header = %q", gotAuth)
	}
	if gotFrom != "InsideOut <noreply@example.com>" || gotTo != "user@example.com" {
		t.Fatalf("from/to = %q/%q", gotFrom, gotTo)
	}
	if gotSubject != "Hello" || gotText != "the link: http://x/l?t=1" {
		t.Fatalf("subject/text = %q/%q", gotSubject, gotText)
	}
}

func TestSendSurfacesProviderError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer srv.Close()

	s := NewSender("re-test-key", "noreply@example.com", testLogger(), false)
	s.client = srv.Client()
	s.endpoint = srv.URL

	if err := s.Send(context.Background(), "user@example.com", "Hello", "t", "h"); err == nil {
		t.Fatal("expected error on non-2xx provider response")
	}
}

func TestDevModeLogsInsteadOfSending(t *testing.T) {
	s := NewSender("", "noreply@example.com", testLogger(), true)
	if s.Enabled() {
		t.Fatal("empty API key must mean dev mode")
	}
	// No HTTP call is possible (no key); the contract is "logs and returns nil".
	if err := s.Send(context.Background(), "user@example.com", "Hello", "t", "h"); err != nil {
		t.Fatalf("dev-mode send must not error: %v", err)
	}
}

func TestDevModeOmitsBodyUnlessOptedIn(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	s := NewSender("", "noreply@example.com", log, false)

	if err := s.Send(context.Background(), "user@example.com", "Hello", "https://x/consume?token=live-secret", "h"); err != nil {
		t.Fatalf("dev-mode send: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "user@example.com") || !strings.Contains(out, "Hello") {
		t.Fatalf("log should keep to/subject, got %q", out)
	}
	if strings.Contains(out, "live-secret") {
		t.Fatalf("log must not carry the message body by default, got %q", out)
	}
}

func TestDevModeLogsBodyWhenOptedIn(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	s := NewSender("", "noreply@example.com", log, true)

	if err := s.Send(context.Background(), "user@example.com", "Hello", "https://x/consume?token=dev-link", "h"); err != nil {
		t.Fatalf("dev-mode send: %v", err)
	}
	if !strings.Contains(buf.String(), "dev-link") {
		t.Fatalf("opted-in dev mode must log the body, got %q", buf.String())
	}
}
