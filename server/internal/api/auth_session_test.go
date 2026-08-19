package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRefreshTokenFromRequestPrefersCookie(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{"refreshToken":"from-body"}`))
	r.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "from-cookie"})
	got, err := refreshTokenFromRequest(r)
	if err != nil {
		t.Fatalf("refreshTokenFromRequest: %v", err)
	}
	if got != "from-cookie" {
		t.Fatalf("got %q, want cookie value", got)
	}
}

func TestRefreshTokenFromRequestReadsJSONBody(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{"refreshToken":"from-body"}`))
	r.Header.Set("Content-Type", "application/json")
	got, err := refreshTokenFromRequest(r)
	if err != nil {
		t.Fatalf("refreshTokenFromRequest: %v", err)
	}
	if got != "from-body" {
		t.Fatalf("got %q, want from-body", got)
	}
}

func TestRefreshTokenFromRequestEmpty(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	if _, err := refreshTokenFromRequest(r); err == nil {
		t.Fatal("expected error when cookie and body are empty")
	}
}

func TestSessionViewKeepsUserFieldsTopLevel(t *testing.T) {
	v := sessionView{
		userView:     userView{ID: "u1", Email: "a@b.c", Username: "ada", Bio: "", Keywords: []string{}},
		AccessToken:  "access",
		RefreshToken: "refresh",
	}
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"id", "email", "username", "accessToken", "refreshToken"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("missing top-level %q in %s", key, raw)
		}
	}
	if _, wrapped := got["user"]; wrapped {
		t.Fatalf("must not wrap the user object: %s", raw)
	}
}
