package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/config"
)

// TEST-13: the GitHub webhook is the one route that lets an unauthenticated
// caller in, so its HMAC check is the only gate. These drive the handler over
// HTTP; none of them reach the store (it is nil here), which is itself the
// point for the rejected and ignored cases.

const hookSecret = "whsec-test-secret"

func hookServer(secret string) *Server {
	return NewServer(nil, nil, &config.Config{GithubWebhookSecret: secret}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
}

func hmacHeader(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// deliver posts one webhook delivery. signature "" sends no signature header.
func deliver(s *Server, event, body, signature string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/hooks/github", strings.NewReader(body))
	r.Header.Set("X-GitHub-Event", event)
	if signature != "" {
		r.Header.Set("X-Hub-Signature-256", signature)
	}
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	return w
}

func TestGithubWebhookIsOffWithoutASecret(t *testing.T) {
	const body = `{"zen":"x"}`
	w := deliver(hookServer(""), "ping", body, hmacHeader("", body))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, want 503 when no webhook secret is configured; body %s", w.Code, w.Body)
	}
}

func TestGithubWebhookRejectsBadSignatures(t *testing.T) {
	const body = `{"zen":"x"}`
	cases := map[string]string{
		"no signature header":          "",
		"not the sha256= form":         "sha1=" + strings.Repeat("0", 40),
		"not hex":                      "sha256=zz",
		"signed with another secret":   hmacHeader("another-secret", body),
		"signed over a different body": hmacHeader(hookSecret, `{"zen":"y"}`),
	}
	s := hookServer(hookSecret)
	for name, sig := range cases {
		t.Run(name, func(t *testing.T) {
			// A ping, so a skipped check would answer 200 instead of 401.
			w := deliver(s, "ping", body, sig)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status %d, want 401; body %s", w.Code, w.Body)
			}
		})
	}
}

func TestGithubWebhookAnswersASignedPing(t *testing.T) {
	const body = `{"zen":"keep it logically awesome"}`
	w := deliver(hookServer(hookSecret), "ping", body, hmacHeader(hookSecret, body))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, want 200; body %s", w.Code, w.Body)
	}
	got := decodeAs[map[string]any](t, w.Body.String())
	if got["ok"] != true || got["event"] != "ping" {
		t.Fatalf("body %s, want ok:true for the ping event", w.Body)
	}
}

func TestGithubWebhookIgnoresOtherSignedEvents(t *testing.T) {
	const body = `{"action":"created"}`
	w := deliver(hookServer(hookSecret), "star", body, hmacHeader(hookSecret, body))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, want 200; body %s", w.Code, w.Body)
	}
	if got := decodeAs[map[string]any](t, w.Body.String()); got["ignored"] != "star" {
		t.Fatalf("body %s, want ignored:star", w.Body)
	}
}
