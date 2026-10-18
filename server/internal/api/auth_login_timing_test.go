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
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// timedLogin POSTs a login through the full handler stack and returns the
// status and the wall time. Every call gets its own client IP (via the header
// clientIP trusts behind one proxy) so the failure lockout never joins in.
func timedLogin(t *testing.T, s *Server, n int, email string) (int, time.Duration) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"email": email, "password": "not-the-password"})
	if err != nil {
		t.Fatalf("marshal login body: %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", n))
	w := httptest.NewRecorder()

	start := time.Now()
	s.Handler().ServeHTTP(w, r)
	return w.Code, time.Since(start)
}

// A login for an unknown email must cost what a wrong password for a known
// email costs. argon2id dominates the latter (about 12 ms on a laptop, more on
// a CI runner); an unknown email that skips it answers in about a millisecond,
// so the response time alone tells an attacker which emails have accounts. The
// floor is deliberately loose (half, on the median of three interleaved tries)
// so scheduler noise cannot flake it while the unfixed gap, an order of
// magnitude, still trips it.
func TestLogin_UnknownEmailCostsLikeWrongPassword(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)

	hash, err := auth.HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	known, err := st.CreateUser(ctx, "login-timing-"+uuid.NewString()+"@test.local", hash, "timing-tester")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	unknownEmail := "no-such-user-" + uuid.NewString() + "@test.local"
	s := NewServer(st, nil, &config.Config{TrustedProxyHops: 1}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)

	// Both paths must end in the same 401, or comparing their times is moot.
	login := func(n int, email string) time.Duration {
		t.Helper()
		code, d := timedLogin(t, s, n, email)
		if code != http.StatusUnauthorized {
			t.Fatalf("login %d: status = %d, want 401", n, code)
		}
		return d
	}

	// One untimed pass over each path absorbs one-off costs (first connection,
	// allocator growth, a lazily built dummy hash).
	login(1, known.Email)
	login(2, unknownEmail)

	// Interleave so CPU or GC noise lands on both sides.
	var knownTimes, unknownTimes []time.Duration
	for i := 0; i < 3; i++ {
		knownTimes = append(knownTimes, login(3+2*i, known.Email))
		unknownTimes = append(unknownTimes, login(4+2*i, unknownEmail))
	}
	slices.Sort(knownTimes)
	slices.Sort(unknownTimes)
	medKnown, medUnknown := knownTimes[1], unknownTimes[1]
	t.Logf("median login: known email + wrong password = %v, unknown email = %v", medKnown, medUnknown)

	if medUnknown < medKnown/2 {
		t.Fatalf("unknown-email login took %v, under half of a known-email wrong-password login (%v): response time reveals which emails have accounts", medUnknown, medKnown)
	}
}
