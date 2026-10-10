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
