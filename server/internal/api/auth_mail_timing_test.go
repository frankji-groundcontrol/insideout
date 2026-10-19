package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// rv4-SEC-03 / rv4-REL-07: the three emailed-auth requests (magic link, forgot
// password, login code) answered a known address only after the token was
// stored and the mail provider had replied, and an unknown address at once. The
// uniform 200 body hid nothing: the response time said which addresses have
// accounts. The same synchronous send held the request open for as long as the
// provider took (the client timeout is 10 s).

// slowMailProvider stands in for the mail provider: every send takes delay to
// be accepted, and the returned counter is how many were. The sender's HTTP
// client has no transport of its own, so it follows http.DefaultTransport at
// request time (see fakeResend); package api runs its tests sequentially, so
// the swap is race-free, and nothing leaves the machine.
func slowMailProvider(t *testing.T, delay time.Duration) *atomic.Int64 {
	t.Helper()
	var accepted atomic.Int64
	prev := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		select {
		case <-time.After(delay):
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
		accepted.Add(1)
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader("{}")), Request: r}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = prev })
	return &accepted
}

// eventually polls cond until it holds or the deadline passes.
func eventually(t *testing.T, within time.Duration, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(within); !cond(); {
		if time.Now().After(deadline) {
			t.Fatalf("gave up after %v waiting for %s", within, what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestAuthMail_KnownAddressAnswersLikeUnknown(t *testing.T) {
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

	const delay = 200 * time.Millisecond
	accepted := slowMailProvider(t, delay)
	s := NewServer(st, nil, &config.Config{
		MailAPIKey: "test-key", MailFrom: "InsideOut <noreply@example.com>", PublicURL: "http://app.test",
		TrustedProxyHops: 1, // one client address per request, so neither gate joins in
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)

	n := 0
	post := func(t *testing.T, path, email string) time.Duration {
		t.Helper()
		n++
		body, err := json.Marshal(map[string]string{"email": email})
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		r := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", n))
		w := httptest.NewRecorder()
		start := time.Now()
		s.Handler().ServeHTTP(w, r)
		took := time.Since(start)
		if w.Code != http.StatusOK {
			t.Fatalf("POST %s: status = %d, want the uniform 200: %s", path, w.Code, w.Body.String())
		}
		return took
	}

	for _, path := range []string{"/api/v1/auth/magic-link", "/api/v1/auth/forgot-password", "/api/v1/auth/login-code"} {
		t.Run(path, func(t *testing.T) {
			const timed = 3
			sentBefore := accepted.Load()
			var known, unknown []time.Duration
			for i := 0; i <= timed; i++ { // pass 0 is an untimed warm-up
				user, err := st.CreateUser(ctx, "mail-timing-"+uuid.NewString()+"@test.local", "x", "timing-tester")
				if err != nil {
					t.Fatalf("create user: %v", err)
				}
				k := post(t, path, user.Email)
				u := post(t, path, "no-such-user-"+uuid.NewString()+"@test.local")
				if i > 0 {
					known, unknown = append(known, k), append(unknown, u)
				}
			}
			// Whenever it happens, every known address still gets its mail.
			eventually(t, 5*time.Second, "every known address's mail to reach the provider", func() bool {
				return accepted.Load() == sentBefore+timed+1
			})

			slices.Sort(known)
			slices.Sort(unknown)
			medKnown, medUnknown := known[timed/2], unknown[timed/2]
			t.Logf("median response with a %v mail provider: known address = %v, unknown address = %v", delay, medKnown, medUnknown)
			if medKnown-medUnknown > delay/2 {
				t.Fatalf("a known address took %v and an unknown one %v: the response time reveals which addresses have accounts "+
					"(and holds the request open for the whole provider round trip)", medKnown, medUnknown)
			}
		})
	}
}
