package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/config"
)

const webhookTestSecret = "test-secret"

// webhookPingBody is a valid-JSON GitHub payload padded to exactly size bytes.
func webhookPingBody(size int) []byte {
	const head, tail = `{"zen":"`, `"}`
	return []byte(head + strings.Repeat("x", size-len(head)-len(tail)) + tail)
}

func webhookSignature(body []byte) string {
	mac := hmac.New(sha256.New, []byte(webhookTestSecret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// postWebhook delivers a GitHub ping through the full handler chain, so the
// global body cap sits in front of the route exactly as in production. A ping
// needs no store, and an empty signature sends no signature header.
func postWebhook(body []byte, signature string) *httptest.ResponseRecorder {
	srv := NewServer(nil, nil, &config.Config{GithubWebhookSecret: webhookTestSecret}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/hooks/github", bytes.NewReader(body))
	r.Header.Set("X-GitHub-Event", "ping")
	if signature != "" {
		r.Header.Set("X-Hub-Signature-256", signature)
	}
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, r)
	return w
}

// Push payloads from busy repos run past the 1 MiB cap every JSON route gets;
// the webhook has its own, larger one, and a delivery under it must be read.
func TestWebhookAcceptsDeliveryOverTheGlobalBodyCap(t *testing.T) {
	body := webhookPingBody(2 << 20)
	w := postWebhook(body, webhookSignature(body))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d (%s), want 200", w.Code, strings.TrimSpace(w.Body.String()))
	}
	var got struct {
		OK    bool   `json:"ok"`
		Event string `json:"event"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || !got.OK || got.Event != "ping" {
		t.Fatalf("body = %q (%v), want ok:true event:ping", w.Body.String(), err)
	}
}

// A large delivery is judged on its signature, not refused as unreadable.
func TestWebhookLargeUnsignedDeliveryIsABadSignature(t *testing.T) {
	if w := postWebhook(webhookPingBody(2<<20), ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d (%s), want 401", w.Code, strings.TrimSpace(w.Body.String()))
	}
}

// The middleware cap is the webhook's single limit: the body is read whole up
// to it, and one byte over fails the read (400) instead of being truncated
// into a signature mismatch (401).
func TestWebhookBodyCapIsEightMiB(t *testing.T) {
	const eightMiB = 8 << 20
	atCap := webhookPingBody(eightMiB)
	if w := postWebhook(atCap, webhookSignature(atCap)); w.Code != http.StatusOK {
		t.Fatalf("8 MiB delivery: status = %d (%s), want 200", w.Code, strings.TrimSpace(w.Body.String()))
	}
	over := webhookPingBody(eightMiB + 1)
	if w := postWebhook(over, webhookSignature(over)); w.Code != http.StatusBadRequest {
		t.Fatalf("8 MiB + 1 delivery: status = %d (%s), want 400", w.Code, strings.TrimSpace(w.Body.String()))
	}
}

// The webhook's room is a path exception, not a new default: every other
// JSON route keeps the 1 MiB cap. The request is valid but for its size, so
// under the cap it would get past validation to the (nil) store; over it the
// decode fails first with 400.
func TestOtherRoutesKeepTheOneMiBBodyCap(t *testing.T) {
	body := `{"token":"` + strings.Repeat("t", 2<<20) + `","password":"longenough1"}`
	if w := postJSON(newValidationServer(), "/api/v1/auth/reset-password", body); w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d (%s), want 400", w.Code, strings.TrimSpace(w.Body.String()))
	}
}
