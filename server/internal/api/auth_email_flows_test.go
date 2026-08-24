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

// newValidationServer builds a Server whose store is nil: these tests only
// exercise request validation, which runs before any store access.
func newValidationServer() *Server {
	return NewServer(nil, nil, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
}

func postJSON(s *Server, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	return w
}

func TestMagicLinkRequestRejectsInvalidEmail(t *testing.T) {
	w := postJSON(newValidationServer(), "/api/v1/auth/magic-link", `{"email":"not-an-email"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestMagicLinkConsumeRequiresToken(t *testing.T) {
	w := postJSON(newValidationServer(), "/api/v1/auth/magic-link/consume", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestForgotPasswordRejectsInvalidEmail(t *testing.T) {
	w := postJSON(newValidationServer(), "/api/v1/auth/forgot-password", `{"email":"nope"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestResetPasswordValidation(t *testing.T) {
	s := newValidationServer()
	cases := []struct {
		name string
		body string
	}{
		{"missing token", `{"password":"longenough1"}`},
		{"short password", `{"token":"t","password":"short"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := postJSON(s, "/api/v1/auth/reset-password", tc.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", w.Code)
			}
		})
	}
}

func TestLoginCodeRequestRejectsInvalidEmail(t *testing.T) {
	w := postJSON(newValidationServer(), "/api/v1/auth/login-code", `{"email":"not-an-email"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestLoginCodeVerifyValidation(t *testing.T) {
	s := newValidationServer()
	cases := []struct {
		name string
		body string
	}{
		{"invalid email", `{"email":"nope","code":"123456"}`},
		{"missing code", `{"email":"a@b.co"}`},
		{"short code", `{"email":"a@b.co","code":"12345"}`},
		{"non-numeric code", `{"email":"a@b.co","code":"12ab56"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := postJSON(s, "/api/v1/auth/login-code/verify", tc.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", w.Code)
			}
		})
	}
}
