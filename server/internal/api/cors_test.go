package api

import "testing"

func TestOriginAllowedExact(t *testing.T) {
	allowed := []string{"https://app.example.com"}
	if !originAllowed("https://app.example.com", allowed) {
		t.Fatal("exact origin should be allowed")
	}
	if originAllowed("https://evil.example.com", allowed) {
		t.Fatal("other origin must be denied")
	}
}

func TestOriginAllowedLocalhostToken(t *testing.T) {
	allowed := []string{"localhost"}
	if !originAllowed("http://localhost:5173", allowed) {
		t.Fatal("localhost port should be allowed")
	}
	if !originAllowed("http://127.0.0.1:8081", allowed) {
		t.Fatal("127.0.0.1 should be allowed via localhost token")
	}
	if originAllowed("https://localhost.evil.com", allowed) {
		t.Fatal("suffix lookalike must be denied")
	}
}

func TestOriginAllowedEmptyListDenies(t *testing.T) {
	if originAllowed("http://localhost:1", nil) {
		t.Fatal("empty allow-list must deny")
	}
}
