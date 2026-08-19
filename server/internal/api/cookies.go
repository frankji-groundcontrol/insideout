package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
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

// refreshTokenFromRequest prefers the httpOnly refresh cookie (Nuxt) and
// otherwise reads {"refreshToken":"..."} so Flutter can refresh without
// cookies. Cookie wins when both are present.
func refreshTokenFromRequest(r *http.Request) (string, error) {
	if c, err := r.Cookie(refreshCookieName); err == nil && c.Value != "" {
		return c.Value, nil
	}
	var body struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := httpx.DecodeJSON(r, &body); err != nil {
		if errors.Is(err, io.EOF) {
			return "", errors.New("no refresh token")
		}
		// A missing/invalid body is the same to the caller: no token.
		var syn *json.SyntaxError
		if errors.As(err, &syn) || err == io.ErrUnexpectedEOF {
			return "", errors.New("no refresh token")
		}
		return "", err
	}
	tok := strings.TrimSpace(body.RefreshToken)
	if tok == "" {
		return "", errors.New("no refresh token")
	}
	return tok, nil
}
