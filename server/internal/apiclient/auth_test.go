package apiclient

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// The auth verbs mirror the five web flows so CLI and MCP can
// authenticate every way the web can. Session-shaped calls must store the
// access token exactly like Login; request-shaped calls must not.

func TestRegisterStoresToken(t *testing.T) {
	var gotPath, gotBody string
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		buf := make([]byte, 512)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		w.Write([]byte(`{"email":"a@b.c","username":"Alice","accessToken":"tok-r","refreshToken":"rtok"}`))
	})
	if err := c.Register("a@b.c", "longenough", "Alice", "inv"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if gotPath != "/auth/register" {
		t.Errorf("path = %q, want /auth/register", gotPath)
	}
	if want := `{"email":"a@b.c","inviteCode":"inv","password":"longenough","username":"Alice"}`; gotBody != want {
		t.Errorf("body = %q, want %q", gotBody, want)
	}
	if c.Token() != "tok-r" {
		t.Errorf("Token() = %q, want tok-r", c.Token())
	}
}

func TestVerifyLoginCodeStoresToken(t *testing.T) {
	var gotPath, gotBody string
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		buf := make([]byte, 256)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		w.Write([]byte(`{"accessToken":"tok-c"}`))
	})
	if err := c.VerifyLoginCode("a@b.c", "123456"); err != nil {
		t.Fatalf("VerifyLoginCode: %v", err)
	}
	if gotPath != "/auth/login-code/verify" {
		t.Errorf("path = %q, want /auth/login-code/verify", gotPath)
	}
	if want := `{"code":"123456","email":"a@b.c"}`; gotBody != want {
		t.Errorf("body = %q, want %q", gotBody, want)
	}
	if c.Token() != "tok-c" {
		t.Errorf("Token() = %q, want tok-c", c.Token())
	}
}

func TestConsumeMagicLinkStoresToken(t *testing.T) {
	var gotPath, gotBody string
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		buf := make([]byte, 512)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		w.Write([]byte(`{"accessToken":"tok-m"}`))
	})
	if err := c.ConsumeMagicLink("tok"); err != nil {
		t.Fatalf("ConsumeMagicLink: %v", err)
	}
	if gotPath != "/auth/magic-link/consume" {
		t.Errorf("path = %q, want /auth/magic-link/consume", gotPath)
	}
	if want := `{"token":"tok"}`; gotBody != want {
		t.Errorf("body = %q, want %q", gotBody, want)
	}
	if c.Token() != "tok-m" {
		t.Errorf("Token() = %q, want tok-m", c.Token())
	}
}

func TestAuthRequestVerbsPostRightBodies(t *testing.T) {
	var gotPath, gotBody, gotAuth string
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		buf := make([]byte, 512)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"ok":true}`))
	})
	cases := []struct {
		name string
		call func() error
		path string
		body string
	}{
		{"magic link", func() error { return c.RequestMagicLink("a@b.c") },
			"/auth/magic-link", `{"email":"a@b.c"}`},
		{"login code", func() error { return c.RequestLoginCode("a@b.c") },
			"/auth/login-code", `{"email":"a@b.c"}`},
		{"forgot password", func() error { return c.ForgotPassword("a@b.c") },
			"/auth/forgot-password", `{"email":"a@b.c"}`},
		{"reset password", func() error { return c.ResetPassword("tok", "newlongenough") },
			"/auth/reset-password", `{"password":"newlongenough","token":"tok"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if gotPath != tc.path {
				t.Errorf("path = %q, want %q", gotPath, tc.path)
			}
			if gotBody != tc.body {
				t.Errorf("body = %q, want %q", gotBody, tc.body)
			}
			if gotAuth != "" {
				t.Errorf("pre-auth request must carry no bearer, got %q", gotAuth)
			}
			if c.Token() != "" {
				t.Errorf("request verb must not set a token, got %q", c.Token())
			}
		})
	}
}

// Admin verbs ride the bearer token; revoke pins the exact route.
func TestRevokeInvitationHitsRevokeRoute(t *testing.T) {
	var gotMethod, gotPath, gotAuth string
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"ok":true}`))
	})
	c.SetToken("tok-admin")
	if err := c.RevokeInvitation("inv-1"); err != nil {
		t.Fatalf("RevokeInvitation: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/admin/invitations/inv-1/revoke" {
		t.Errorf("got %s %s, want POST /admin/invitations/inv-1/revoke", gotMethod, gotPath)
	}
	if gotAuth != "Bearer tok-admin" {
		t.Errorf("Authorization = %q, want Bearer tok-admin", gotAuth)
	}
}

// The settings route is PUT, not POST — a POST gets a 405 from the real
// API (caught by the 2026-09-03 live matrix).
func TestSetSignupModeUsesPut(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]string
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Write([]byte(`{"signupMode":"open"}`))
	})
	c.SetToken("tok-admin")
	if _, err := c.SetSignupMode("open"); err != nil {
		t.Fatalf("SetSignupMode: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/admin/settings" {
		t.Errorf("got %s %s, want PUT /admin/settings", gotMethod, gotPath)
	}
	if gotBody["signupMode"] != "open" {
		t.Errorf("body = %v, want signupMode=open", gotBody)
	}
}

// A 401 from an auth endpoint is the flow's own answer ("invalid or
// expired code"), not a missing INSIDEOUT_TOKEN — the error must carry
// the API's text instead of the token hint.
func TestAuthFailureCarriesAPIError(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid or expired code"}`))
	})
	err := c.VerifyLoginCode("a@b.c", "000000")
	if err == nil || !strings.Contains(err.Error(), "invalid or expired code") {
		t.Fatalf("want API error text, got %v", err)
	}
	if strings.Contains(err.Error(), "INSIDEOUT_TOKEN") {
		t.Fatalf("auth 401 must not use the token hint: %v", err)
	}
}
