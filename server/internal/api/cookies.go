package api

import (
	"net/http"
	"time"
)

const (
	accessCookieName  = "insideout_access"
	refreshCookieName = "insideout_refresh"
)

func (s *Server) setAuthCookies(w http.ResponseWriter, accessToken, refreshToken string, accessTTL, refreshTTL time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(accessTTL.Seconds()),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    refreshToken,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(refreshTTL.Seconds()),
	})
}

func (s *Server) clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: accessCookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name: refreshCookieName, Value: "", Path: "/api/v1/auth", MaxAge: -1,
		HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode,
	})
}

// bearerOrCookie extracts an access token from the Authorization header
// (for tests/CLI use) falling back to the access cookie (for the browser).
func bearerOrCookie(r *http.Request) string {
	if h := r.Header.Get("Authorization"); len(h) > 7 && h[:7] == "Bearer " {
		return h[7:]
	}
	if c, err := r.Cookie(accessCookieName); err == nil {
		return c.Value
	}
	return ""
}
