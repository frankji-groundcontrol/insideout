package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// rv4-REL-07: the emailed auth requests used to wait for the provider and then
// log its failure. The send now happens after the response, so the failure must
// still leave its trail (the log line and the admin probe's LastSendError) and
// the response must not change shape.
func TestAuthMail_ProviderFailureIsLoggedNotSurfaced(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dbURL)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)

	prev := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{"message":"domain is not verified"}`)), Request: r}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = prev })

	var logs bytes.Buffer
	s := NewServer(st, nil, &config.Config{
		MailAPIKey: "test-key", MailFrom: "InsideOut <noreply@example.com>", PublicURL: "http://app.test",
		TrustedProxyHops: 1,
	}, slog.New(slog.NewTextHandler(&logs, nil)), nil, nil)

	for i, tc := range []struct{ path, logged string }{
		{"/api/v1/auth/magic-link", "magic link: send"},
		{"/api/v1/auth/forgot-password", "forgot password: send"},
		{"/api/v1/auth/login-code", "login code: send"},
	} {
		// One user and one address per path: the resend cooldown is shared by the three.
		user, err := st.CreateUser(ctx, "mail-failure-"+uuid.NewString()+"@test.local", "x", "failure-tester")
		if err != nil {
			t.Fatalf("create user: %v", err)
		}
		assertUniformOK(t, postAuthSend(s, tc.path, user.Email, fmt.Sprintf("198.51.100.%d", i+1)))
		s.mail.Wait()
		if got := s.mail.LastSendError(); !strings.Contains(got, "403") {
			t.Errorf("%s: LastSendError = %q, want the provider's 403 (the admin probe reads it)", tc.path, got)
		}
		if !strings.Contains(logs.String(), tc.logged) {
			t.Errorf("%s: the failed send left no %q log line:\n%s", tc.path, tc.logged, logs.String())
		}
	}
}
