package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// fakeClock drives a sendCooldown without sleeps; tests advance it past
// the cooldown window instead of waiting wall-clock time.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }
func newFakeClock() *fakeClock               { return &fakeClock{t: time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)} }

func TestSendCooldownBlocksRepeatForSameEmail(t *testing.T) {
	cd := newSendCooldown(authResendCooldown)
	if !cd.allow("a@x.co", "1.1.1.1") {
		t.Fatal("first send must be allowed")
	}
	// The email key blocks even from a different IP — the per-email gate
	// is what stops mailbox spam, the per-IP gate only blunts spraying.
	if cd.allow("a@x.co", "2.2.2.2") {
		t.Fatal("repeat for the same email within the window must be blocked")
	}
	if !cd.allow("b@x.co", "3.3.3.3") {
		t.Fatal("an unrelated email/IP pair must be unaffected")
	}
}

func TestSendCooldownBlocksSameIPDifferentEmail(t *testing.T) {
	cd := newSendCooldown(authResendCooldown)
	if !cd.allow("a@x.co", "9.9.9.9") {
		t.Fatal("first send must be allowed")
	}
	// The IP key blocks a different email from the same source — this is
	// what stops one client from spraying many addresses.
	if cd.allow("b@x.co", "9.9.9.9") {
		t.Fatal("same IP with a different email within the window must be blocked")
	}
	if !cd.allow("b@x.co", "8.8.8.8") {
		t.Fatal("the same email from a fresh IP must be allowed")
	}
}

func TestSendCooldownAllowsAfterWindow(t *testing.T) {
	clock := newFakeClock()
	cd := newSendCooldown(authResendCooldown)
	cd.now = clock.now

	if !cd.allow("a@x.co", "1.1.1.1") {
		t.Fatal("first send must be allowed")
	}
	clock.advance(authResendCooldown + time.Second)
	if !cd.allow("a@x.co", "1.1.1.1") {
		t.Fatal("send must be allowed again after the window")
	}
}

func TestSendCooldownBlockedAttemptDoesNotExtendWindow(t *testing.T) {
	clock := newFakeClock()
	cd := newSendCooldown(authResendCooldown)
	cd.now = clock.now

	if !cd.allow("a@x.co", "1.1.1.1") {
		t.Fatal("first send must be allowed")
	}
	clock.advance(30 * time.Second)
	if cd.allow("a@x.co", "1.1.1.1") {
		t.Fatal("repeat within the window must be blocked")
	}
	// The window anchors at the last ACTUAL send, not at the blocked
	// attempt — otherwise re-requesting every 59s would freeze the
	// cooldown forever.
	clock.advance(31 * time.Second)
	if !cd.allow("a@x.co", "1.1.1.1") {
		t.Fatal("window must not be extended by blocked attempts")
	}
}

// TestSendCooldownEndpointsReturnUniform200 primes the gate as if a first
// send already happened, then hits each send endpoint through HTTP: the
// response must be the same uniform 200 {"ok":true} as every other
// outcome, and nothing may reach the mail path. The server has a nil
// store, so reaching the lookup would panic — surviving to the 200 is
// itself proof the gate short-circuited before any send work.
func TestSendCooldownEndpointsReturnUniform200(t *testing.T) {
	endpoints := []struct {
		path  string
		email string
	}{
		{"/api/v1/auth/magic-link", "cooled-magic@test.local"},
		{"/api/v1/auth/forgot-password", "cooled-forgot@test.local"},
		{"/api/v1/auth/login-code", "cooled-code@test.local"},
	}
	for _, ep := range endpoints {
		t.Run(ep.path, func(t *testing.T) {
			cd := newSendCooldown(authResendCooldown)
			// Prime: stand in for the first request's send so the HTTP
			// request below lands inside the window. httptest requests
			// carry RemoteAddr 192.0.2.1:1234, which clientIP
			// reduces to 192.0.2.1.
			cd.allow(ep.email, "192.0.2.1")

			var mailLog bytes.Buffer
			s := NewServer(nil, nil, &config.Config{},
				slog.New(slog.NewTextHandler(&mailLog, nil)), nil, nil)
			s.authCooldown = cd

			w := postJSON(s, ep.path, `{"email":"`+ep.email+`"}`)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (uniform response on cooldown)", w.Code)
			}
			if got := w.Body.String(); got != "{\"ok\":true}\n" {
				t.Fatalf("body = %q, want uniform {\"ok\":true}", got)
			}
			if strings.Contains(mailLog.String(), "mail dev mode") {
				t.Fatal("cooldown-blocked request must not reach the mail sender")
			}
		})
	}
}

// postAuthSend POSTs an email-only auth-send body, pinning the client IP
// via X-Forwarded-For (the same header clientIP trusts) when xff is set.
func postAuthSend(s *Server, path, email, xff string) *httptest.ResponseRecorder {
	body, err := json.Marshal(map[string]string{"email": email})
	if err != nil {
		panic(err)
	}
	r := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	return w
}

func countDevModeSends(buf *bytes.Buffer) int {
	return bytes.Count(buf.Bytes(), []byte("mail dev mode"))
}

func assertUniformOK(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (uniform response on cooldown)", w.Code)
	}
	if got := w.Body.String(); got != "{\"ok\":true}\n" {
		t.Fatalf("body = %q, want uniform {\"ok\":true}", got)
	}
}

// TestSendCooldownIsServerScoped pins the gate's lifetime to its Server, like
// authThrottle: a send one Server recorded must not suppress the first send
// on another (a process-wide gate makes every Server, and so every test,
// share one window). The servers have a nil store, so a request that clears
// the gate dies in the user lookup and withRecover answers 500, while a
// blocked one gets the uniform 200: the status tells them apart without a DB.
func TestSendCooldownIsServerScoped(t *testing.T) {
	const email, ip = "scoped@test.local", "192.0.2.77"
	for _, name := range []string{"first", "second"} {
		w := postAuthSend(newValidationServer(), "/api/v1/auth/magic-link", email, ip)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("%s server: status = %d, want 500 (a fresh Server's first send must clear its own gate and reach the lookup; 200 means another Server's window blocked it)", name, w.Code)
		}
	}
}

// TestInjectedCooldownsStayWithTheirServer covers the test seam: tests give
// the server they build a fake-clock gate, and two such tests alive at once
// (t.Parallel) must each answer to their own gate. A swapped package global
// cannot: the second injection replaces the first, so the first test's
// primed window silently stops blocking and nothing but "no test is
// parallel" keeps that from showing.
func TestInjectedCooldownsStayWithTheirServer(t *testing.T) {
	const email, ip = "injected@test.local", "192.0.2.88"
	gateA := newSendCooldown(authResendCooldown)
	gateB := newSendCooldown(authResendCooldown)

	a := newValidationServer()
	a.authCooldown = gateA
	b := newValidationServer() // the other test's setup, running alongside
	b.authCooldown = gateB

	gateA.allow(email, ip) // only A's window has a send in it

	if w := postAuthSend(a, "/api/v1/auth/magic-link", email, ip); w.Code != http.StatusOK {
		t.Fatalf("server a: status = %d, want 200 (its own primed gate must block it, not the gate b installed)", w.Code)
	}
	if w := postAuthSend(b, "/api/v1/auth/magic-link", email, ip); w.Code != http.StatusInternalServerError {
		t.Fatalf("server b: status = %d, want 500 (a's window must not block b, whose gate has no send in it)", w.Code)
	}
}

// TestEmailAuthCooldownEndToEnd exercises the real handler stack — store,
// token minting, and the dev-mode mail sender (counted via its log line)
// — behind the cooldown gate. It needs a real Postgres like every other
// store-backed test in this repo; without DATABASE_URL it skips.
func TestEmailAuthCooldownEndToEnd(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	st, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)

	mkEmail := func(tag string) string {
		return "cooldown-" + tag + "-" + uuid.NewString() + "@test.local"
	}
	ctx := context.Background()
	u1, err := st.CreateUser(ctx, mkEmail("one"), "x", "cooldown-tester")
	if err != nil {
		t.Fatalf("create user 1: %v", err)
	}
	u2, err := st.CreateUser(ctx, mkEmail("two"), "x", "cooldown-tester")
	if err != nil {
		t.Fatalf("create user 2: %v", err)
	}

	var mailLog bytes.Buffer
	s := NewServer(st, nil, &config.Config{},
		slog.New(slog.NewTextHandler(&mailLog, nil)), nil, nil)

	clock := newFakeClock()
	cd := newSendCooldown(authResendCooldown)
	cd.now = clock.now
	s.authCooldown = cd

	const ipA, ipB, ipC = "192.0.2.1", "198.51.100.7", "203.0.113.9"

	assertUniformOK(t, postAuthSend(s, "/api/v1/auth/magic-link", u1.Email, ipA))
	if got := countDevModeSends(&mailLog); got != 1 {
		t.Fatalf("after first request: sends = %d, want 1", got)
	}

	// Same email, same IP: blocked, still 200.
	assertUniformOK(t, postAuthSend(s, "/api/v1/auth/magic-link", u1.Email, ipA))
	// Same IP, different email: blocked by the IP key, still 200.
	assertUniformOK(t, postAuthSend(s, "/api/v1/auth/magic-link", u2.Email, ipA))
	// Same email, different IP: blocked by the email key, still 200.
	assertUniformOK(t, postAuthSend(s, "/api/v1/auth/magic-link", u1.Email, ipB))
	if got := countDevModeSends(&mailLog); got != 1 {
		t.Fatalf("after cooldown-blocked repeats: sends = %d, want 1", got)
	}

	// Past the window the send goes out again — no sleeping, the fake
	// clock jumps.
	clock.advance(authResendCooldown + time.Second)
	assertUniformOK(t, postAuthSend(s, "/api/v1/auth/magic-link", u1.Email, ipB))
	if got := countDevModeSends(&mailLog); got != 2 {
		t.Fatalf("after window expiry: sends = %d, want 2", got)
	}

	// The gate is shared across the three send endpoints: u1 mailed a
	// magic link a moment ago, so both other endpoints must stay quiet.
	assertUniformOK(t, postAuthSend(s, "/api/v1/auth/forgot-password", u1.Email, ipC))
	assertUniformOK(t, postAuthSend(s, "/api/v1/auth/login-code", u1.Email, ipC))
	if got := countDevModeSends(&mailLog); got != 2 {
		t.Fatalf("cross-endpoint cooldown: sends = %d, want 2", got)
	}

	// And once the window passes, the login-code send path itself works.
	clock.advance(authResendCooldown + time.Second)
	assertUniformOK(t, postAuthSend(s, "/api/v1/auth/login-code", u1.Email, ipC))
	if got := countDevModeSends(&mailLog); got != 3 {
		t.Fatalf("login-code after window expiry: sends = %d, want 3", got)
	}
}
