package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/config"
)

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

// varyNamesOrigin reports whether any Vary line of h lists Origin.
func varyNamesOrigin(h http.Header) bool {
	for _, line := range h.Values("Vary") {
		for _, field := range strings.Split(line, ",") {
			if strings.EqualFold(strings.TrimSpace(field), "Origin") {
				return true
			}
		}
	}
	return false
}

func corsTestServer(cfg *config.Config) *Server {
	return NewServer(nil, nil, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
}

var corsOKHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

// The CORS answer depends on the Origin request header, so a shared cache
// keyed without it would hand one origin's answer to another. Vary: Origin
// therefore has to ride on every response from the CORS layer — denied-origin
// and no-Origin ones included, not only the allowed branch.
func TestAllowlistCORSAlwaysVariesOnOrigin(t *testing.T) {
	const allowed = "https://app.example.com"
	h := corsTestServer(&config.Config{CORSOrigins: []string{allowed}}).withAllowlistCORS(corsOKHandler)
	cases := []struct{ name, origin, wantAllowOrigin string }{
		{"denied origin", "https://evil.example.com", ""},
		{"no origin", "", ""},
		{"allowed origin", allowed, allowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/v1/ideas", nil)
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if !varyNamesOrigin(w.Header()) {
				t.Errorf("Vary = %q, want it to name Origin", w.Header().Values("Vary"))
			}
			if got := w.Header().Get("Access-Control-Allow-Origin"); got != tc.wantAllowOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tc.wantAllowOrigin)
			}
		})
	}
}

// Another layer's Vary field must survive: Origin is added, never set over it.
func TestAllowlistCORSKeepsOtherVaryFields(t *testing.T) {
	h := corsTestServer(&config.Config{CORSOrigins: []string{"https://app.example.com"}}).withAllowlistCORS(corsOKHandler)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/ideas", nil)
	r.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	w.Header().Set("Vary", "Accept-Encoding")
	h.ServeHTTP(w, r)
	if got := strings.Join(w.Header().Values("Vary"), ", "); !strings.Contains(got, "Accept-Encoding") || !varyNamesOrigin(w.Header()) {
		t.Fatalf("Vary = %q, want both Accept-Encoding and Origin", got)
	}
}

// The permissive dev middleware reflects the request Origin verbatim, so it
// varies on Origin for the same reason.
func TestDevCORSVariesOnOrigin(t *testing.T) {
	h := corsTestServer(&config.Config{DevPermissiveCORS: true}).withDevCORS(corsOKHandler)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/ideas", nil)
	r.Header.Set("Origin", "https://dev.example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://dev.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want the request origin reflected", got)
	}
	if !varyNamesOrigin(w.Header()) {
		t.Fatalf("Vary = %q, want it to name Origin", w.Header().Values("Vary"))
	}
}
