package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// rv4-SEC-02: the auth throttle and the resend cooldown key on the caller's
// address, and that address was the tail of X-Forwarded-For whatever stood in
// front of the server. A client that reaches the server directly (or through a
// proxy that appends nothing) writes the header itself, so a fresh value per
// request minted a fresh bucket per request and no cap ever fired: password
// guessing, 6-digit code guessing and mail volume were all unthrottled. With no
// trusted proxy configured, the one address a client cannot forge is the TCP
// peer, so that is the key.

// peerPost is one POST from the single peer 192.0.2.1 carrying xff.
func peerPost(h http.Handler, path, body, xff string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.RemoteAddr = "192.0.2.1:40000"
	r.Header.Set("Content-Type", "application/json")
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestThrottleIgnoresAForgedForwardedForByDefault(t *testing.T) {
	cases := []struct {
		name   string
		path   string
		body   string
		stub   int
		tune   func(*throttleConfig)
		allow  int // requests that pass before the cap answers 429
		credit string
	}{
		{"password guesses", "/api/v1/auth/login", `{"email":"victim@example.com","password":"guess"}`,
			http.StatusUnauthorized, func(c *throttleConfig) { c.failLimit = 2 }, 2, "failed logins"},
		{"login code guesses", "/api/v1/auth/login-code/verify", `{"email":"victim@example.com","code":"000000"}`,
			http.StatusUnauthorized, func(c *throttleConfig) { c.failLimit = 2 }, 2, "failed code checks"},
		{"mail volume", "/api/v1/auth/magic-link", `{"email":"victim@example.com"}`,
			http.StatusOK, func(c *throttleConfig) { c.sendLimit = 2 }, 2, "send requests"},
		{"registration volume", "/api/v1/auth/register", `{}`,
			http.StatusOK, func(c *throttleConfig) { c.ipLimit = 2 }, 2, "requests"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := throttleTestConfig()
			tc.tune(&cfg)
			_, h := newThrottleTestServer(cfg, &fakeThrottleClock{now: time.Now()}, tc.stub)

			for i := 1; i <= tc.allow; i++ {
				if rec := peerPost(h, tc.path, tc.body, fmt.Sprintf("198.51.100.%d", i)); rec.Code != tc.stub {
					t.Fatalf("request %d: status = %d, want %d before the cap", i, rec.Code, tc.stub)
				}
			}
			// Same peer, a header value nobody has used yet: still the same client.
			rec := peerPost(h, tc.path, tc.body, fmt.Sprintf("198.51.100.%d", tc.allow+1))
			if rec.Code != http.StatusTooManyRequests {
				t.Fatalf("request %d from one peer with a fresh X-Forwarded-For: status = %d, want 429 — "+
					"the forged header minted a new bucket, so %d %s never add up", tc.allow+1, rec.Code, tc.allow+1, tc.credit)
			}
		})
	}
}

// The resend cooldown shares the extraction. A server with a nil store answers
// 500 when a request clears the gate (the user lookup dies) and the uniform 200
// when the gate blocks it, so the status tells the two apart without a database.
func TestResendCooldownIgnoresAForgedForwardedForByDefault(t *testing.T) {
	for _, path := range []string{
		"/api/v1/auth/magic-link",
		"/api/v1/auth/forgot-password",
		"/api/v1/auth/login-code",
	} {
		t.Run(path, func(t *testing.T) {
			s := newValidationServer()

			// postAuthSend's request comes from one peer; only the header differs.
			first := postAuthSend(s, path, "first-victim@example.com", "198.51.100.1")
			if first.Code != http.StatusInternalServerError {
				t.Fatalf("first send: status = %d, want 500 (a quiet gate lets the request reach the lookup)", first.Code)
			}
			second := postAuthSend(s, path, "second-victim@example.com", "198.51.100.2")
			if second.Code != http.StatusOK {
				t.Fatalf("second send from the same peer with a fresh X-Forwarded-For: status = %d, want 200 "+
					"(the per-IP cooldown must hold the peer; 500 means the forged header minted a new bucket "+
					"and the sprayer mails one address per request)", second.Code)
			}
		})
	}
}
