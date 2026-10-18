package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIPPrefersRightmostXForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	r.RemoteAddr = "10.0.0.1:40000"
	r.Header.Set("X-Forwarded-For", "1.1.1.1, 2.2.2.2, 3.3.3.3")
	if got := clientIP(r); got != "3.3.3.3" {
		t.Fatalf("got %q, want rightmost XFF entry 3.3.3.3", got)
	}

	r.Header.Set("X-Forwarded-For", "9.9.9.9")
	if got := clientIP(r); got != "9.9.9.9" {
		t.Fatalf("got %q, want 9.9.9.9", got)
	}

	// A trailing empty entry must not hide the client behind RemoteAddr: the
	// resend cooldown and the auth throttle both key on this one helper.
	r.Header.Set("X-Forwarded-For", "9.9.9.9, ")
	if got := clientIP(r); got != "9.9.9.9" {
		t.Fatalf("got %q with a trailing empty entry, want 9.9.9.9", got)
	}

	r.Header.Del("X-Forwarded-For")
	if got := clientIP(r); got != "10.0.0.1" {
		t.Fatalf("got %q, want RemoteAddr host 10.0.0.1", got)
	}

	r.RemoteAddr = "10.0.0.2" // no port
	if got := clientIP(r); got != "10.0.0.2" {
		t.Fatalf("got %q, want 10.0.0.2", got)
	}
}

// REL-13 (review 2): the Railway chain is browser → edge → nginx → server,
// and nginx passes the edge's XFF through untouched. The server therefore
// sees "<client-spoofed…>, <edge-appended real client>" — the rightmost
// non-empty entry must be that client IP, never a shared proxy hop, and a
// trailing empty entry (misbehaving proxy) must not fall back to RemoteAddr.
func TestClientIP_RailwayChainIsPerClient(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	r.RemoteAddr = "10.0.0.1:40000"

	r.Header.Set("X-Forwarded-For", "6.6.6.6, 7.7.7.7, 198.51.100.23")
	if got := clientIP(r); got != "198.51.100.23" {
		t.Fatalf("got %q, want the edge-appended client 198.51.100.23", got)
	}

	r.Header.Set("X-Forwarded-For", "198.51.100.23, ")
	if got := clientIP(r); got != "198.51.100.23" {
		t.Fatalf("got %q with a trailing empty entry, want 198.51.100.23", got)
	}

	r.Header.Set("X-Forwarded-For", ", ,")
	if got := clientIP(r); got != "10.0.0.1" {
		t.Fatalf("all-empty XFF should fall back to RemoteAddr, got %q", got)
	}
}

// rv4-QUAL-02: the resend cooldown kept its own copy of the client-IP
// extraction, and the copy took the rightmost X-Forwarded-For entry even
// when it was empty, then fell back to RemoteAddr: behind the proxy that is
// the shared hop, so every client whose header ended in ", " shared one
// cooldown bucket and the first of them silenced everyone else's sends
// behind the uniform 200. The gate must key on clientIP, which walks past
// empty entries. A server with a nil store answers 500 when a request clears
// the gate (the user lookup dies) and the uniform 200 when the gate blocks
// it, so the status tells the two apart without a database.
func TestSendCooldownKeysOnClientIPPastTrailingEmptyXFF(t *testing.T) {
	for _, path := range []string{
		"/api/v1/auth/magic-link",
		"/api/v1/auth/forgot-password",
		"/api/v1/auth/login-code",
	} {
		t.Run(path, func(t *testing.T) {
			s := newValidationServer()

			first := postAuthSend(s, path, "first-client@example.com", "9.9.9.9, ")
			if first.Code != http.StatusInternalServerError {
				t.Fatalf("first client: status = %d, want 500 (a quiet gate lets the request reach the lookup)", first.Code)
			}
			// A different client behind the same proxy: its own bucket.
			second := postAuthSend(s, path, "second-client@example.com", "8.8.8.8, ")
			if second.Code != http.StatusInternalServerError {
				t.Fatalf("second client: status = %d, want 500 (200 means the gate keyed on the shared RemoteAddr instead of the client's own address)", second.Code)
			}
			// The same client with another address is still held by its IP.
			third := postAuthSend(s, path, "third-client@example.com", "9.9.9.9, ")
			assertUniformOK(t, third)
		})
	}
}
