package api

import (
	"net/http"
	"net/url"
	"strings"
)

// originAllowed reports whether origin is on the allow-list. The token
// "localhost" matches http(s)://localhost:<port> and 127.0.0.1 only — not
// lookalike hosts like localhost.evil.com.
func originAllowed(origin string, allowed []string) bool {
	if origin == "" || len(allowed) == 0 {
		return false
	}
	for _, raw := range allowed {
		a := strings.TrimSpace(raw)
		if a == "" {
			continue
		}
		if a == origin {
			return true
		}
		if a == "localhost" && localhostOrigin(origin) {
			return true
		}
	}
	return false
}

func localhostOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	host := u.Hostname()
	return host == "localhost" || host == "127.0.0.1"
}

func splitCORSOrigins(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (s *Server) withAllowlistCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allowed or not, the answer depends on Origin, so a cache must key
		// on it. Add, not Set: another layer's Vary field survives.
		w.Header().Add("Vary", "Origin")
		origin := r.Header.Get("Origin")
		if originAllowed(origin, s.cfg.CORSOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
