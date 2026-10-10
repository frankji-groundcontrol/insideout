package api

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

type fakeThrottleClock struct {
	now time.Time
}

func (c *fakeThrottleClock) time() time.Time { return c.now }
func (c *fakeThrottleClock) advance(d time.Duration) {
	c.now = c.now.Add(d)
}

// throttleTestConfig shrinks the defaults so tests exercise the same code
// paths without sleeping; every limit the test doesn't care about is set
// high enough to stay out of the way.
func throttleTestConfig() throttleConfig {
	return throttleConfig{
		failLimit:  8,
		failWindow: 15 * time.Minute,
		lockout:    15 * time.Minute,
		sendLimit:  3,
		sendWindow: 10 * time.Minute,
		ipLimit:    30,
		ipWindow:   time.Minute,
	}
}

// newThrottleTestServer builds a Server whose only real component is the
// throttle, wrapped around a stub handler that answers every request with
// stubStatus (401 simulates bad credentials, 200 a success).
func newThrottleTestServer(cfg throttleConfig, clock *fakeThrottleClock, stubStatus int) (*Server, http.Handler) {
	s := &Server{authThrottle: newAuthThrottle(cfg, clock.time)}
	stub := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(stubStatus)
	})
	return s, s.withAuthThrottle(stub)
}

func doThrottledRequest(h http.Handler, method, path, body, ip string) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	r := httptest.NewRequest(method, path, reader)
	if ip != "" {
		r.RemoteAddr = ip + ":40000"
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestAuthThrottleLoginLocksAfterRepeatedFailures(t *testing.T) {
	cfg := throttleTestConfig()
	cfg.failLimit = 3
	clock := &fakeThrottleClock{now: time.Now()}
	_, h := newThrottleTestServer(cfg, clock, http.StatusUnauthorized)

	for i := 1; i <= 3; i++ {
		rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login", `{"email":"a@x.com","password":"wrong"}`, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: got %d, want 401 before the lockout trips", i, rec.Code)
		}
	}

	rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login", `{"email":"a@x.com","password":"wrong"}`, "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("after failLimit failures got %d, want 429", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "900" {
		t.Fatalf("Retry-After = %q, want 900 (15-minute lockout)", got)
	}

	clock.advance(cfg.lockout)
	rec = doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login", `{"email":"a@x.com","password":"wrong"}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("after the lockout expired got %d, want the request through (401)", rec.Code)
	}
}

func TestAuthThrottleSuccessResetsFailures(t *testing.T) {
	cfg := throttleTestConfig()
	cfg.failLimit = 3
	clock := &fakeThrottleClock{now: time.Now()}
	s := &Server{authThrottle: newAuthThrottle(cfg, clock.time)}
	stubStatus := http.StatusUnauthorized
	stub := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(stubStatus) })
	h := s.withAuthThrottle(stub)

	body := `{"email":"a@x.com","password":"x"}`
	doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login", body, "")
	doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login", body, "")

	stubStatus = http.StatusOK
	if rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login", body, ""); rec.Code != http.StatusOK {
		t.Fatalf("success path got %d, want 200", rec.Code)
	}

	stubStatus = http.StatusUnauthorized
	for i := 0; i < 2; i++ {
		if rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login", body, ""); rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d after a success: got %d, want 401 (counter must have reset)", i, rec.Code)
		}
	}
}

func TestAuthThrottleLockoutIsPerEmailAndIP(t *testing.T) {
	cfg := throttleTestConfig()
	cfg.failLimit = 1 // every failure locks
	clock := &fakeThrottleClock{now: time.Now()}
	_, h := newThrottleTestServer(cfg, clock, http.StatusUnauthorized)

	// Trip the lockout for a@x.com from 192.0.2.1.
	if rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login", `{"email":"a@x.com","password":"x"}`, "192.0.2.1"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("first attempt got %d, want 401", rec.Code)
	}

	// The same email from the same IP is locked; the key is the email+IP
	// pair, so another IP — and another account on a fresh IP — are not.
	// (A request that reaches the stub 401s and, with failLimit=1, locks its
	// own keys, so each probe below uses a fresh IP.)
	cases := []struct {
		name string
		body string
		ip   string
		want int
	}{
		{"same email same IP is locked", `{"email":"a@x.com","password":"x"}`, "192.0.2.1", http.StatusTooManyRequests},
		{"email match is case-insensitive", `{"email":"A@X.COM","password":"x"}`, "192.0.2.1", http.StatusTooManyRequests},
		{"same email other IP still attempts", `{"email":"a@x.com","password":"x"}`, "192.0.2.2", http.StatusUnauthorized},
		{"other email other IP still attempts", `{"email":"b@x.com","password":"x"}`, "192.0.2.3", http.StatusUnauthorized},
		// "other email, same IP" 429s too — the IP key locked alongside the
		// account key — which the malformed-body test below pins down.
	}
	for _, tc := range cases {
		rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login", tc.body, tc.ip)
		if rec.Code != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, rec.Code, tc.want)
		}
	}
}

func TestAuthThrottleFailedBodyStillCountsAgainstIP(t *testing.T) {
	cfg := throttleTestConfig()
	cfg.failLimit = 1
	clock := &fakeThrottleClock{now: time.Now()}
	_, h := newThrottleTestServer(cfg, clock, http.StatusUnauthorized)

	// A malformed body (no email) can't key per-account, but the failure
	// still locks the IP key — the next legitimate login from that IP 429s.
	if rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login", `{"password":"x"}`, "192.0.2.1"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("first attempt got %d, want 401", rec.Code)
	}
	rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login", `{"email":"victim@x.com","password":"x"}`, "192.0.2.1")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("login after the IP locked got %d, want 429", rec.Code)
	}
}

func TestAuthThrottleSendCapPerEmail(t *testing.T) {
	cfg := throttleTestConfig()
	cfg.sendLimit = 2
	clock := &fakeThrottleClock{now: time.Now()}
	_, h := newThrottleTestServer(cfg, clock, http.StatusOK)

	for i := 1; i <= 2; i++ {
		if rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/magic-link", `{"email":"a@x.com"}`, ""); rec.Code != http.StatusOK {
			t.Fatalf("send %d got %d, want 200", i, rec.Code)
		}
	}
	rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/magic-link", `{"email":"a@x.com"}`, "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("send over the cap got %d, want 429", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "600" {
		t.Fatalf("Retry-After = %q, want 600 (10-minute send window)", got)
	}
	if rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/magic-link", `{"email":"b@x.com"}`, ""); rec.Code != http.StatusOK {
		t.Fatalf("a different email got %d, want 200 (per-email cap)", rec.Code)
	}

	clock.advance(cfg.sendWindow)
	if rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/magic-link", `{"email":"a@x.com"}`, ""); rec.Code != http.StatusOK {
		t.Fatalf("after the send window reset got %d, want 200", rec.Code)
	}
}

func TestAuthThrottleSendCapIPBackstop(t *testing.T) {
	cfg := throttleTestConfig()
	cfg.sendLimit = 100
	cfg.ipLimit = 2
	clock := &fakeThrottleClock{now: time.Now()}
	_, h := newThrottleTestServer(cfg, clock, http.StatusOK)

	// Distinct emails keep the per-email counter low; the per-IP backstop
	// still trips.
	for i := 1; i <= 2; i++ {
		body := `{"email":"user` + strconv.Itoa(i) + `@x.com"}`
		if rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login-code", body, "192.0.2.1"); rec.Code != http.StatusOK {
			t.Fatalf("request %d got %d, want 200", i, rec.Code)
		}
	}
	body := `{"email":"user3@x.com"}`
	if rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login-code", body, "192.0.2.1"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("third send from one IP got %d, want 429 (IP backstop)", rec.Code)
	}
	if rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login-code", body, "192.0.2.2"); rec.Code != http.StatusOK {
		t.Fatalf("same email from another IP got %d, want 200", rec.Code)
	}
}

func TestAuthThrottleVolumeCapPerIP(t *testing.T) {
	cfg := throttleTestConfig()
	cfg.ipLimit = 2
	clock := &fakeThrottleClock{now: time.Now()}
	_, h := newThrottleTestServer(cfg, clock, http.StatusOK)

	for i := 1; i <= 2; i++ {
		if rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/register", `{"email":"new@x.com"}`, "192.0.2.1"); rec.Code != http.StatusOK {
			t.Fatalf("request %d got %d, want 200", i, rec.Code)
		}
	}
	if rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/register", `{"email":"new@x.com"}`, "192.0.2.1"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("register over the IP cap got %d, want 429", rec.Code)
	}
	if rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/register", `{"email":"new@x.com"}`, "192.0.2.2"); rec.Code != http.StatusOK {
		t.Fatalf("register from another IP got %d, want 200", rec.Code)
	}

	clock.advance(cfg.ipWindow)
	if rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/register", `{"email":"new@x.com"}`, "192.0.2.1"); rec.Code != http.StatusOK {
		t.Fatalf("register after the IP window reset got %d, want 200", rec.Code)
	}
}

func TestAuthThrottleRetryAfterCountsDown(t *testing.T) {
	cfg := throttleTestConfig()
	cfg.failLimit = 1
	cfg.lockout = 15 * time.Minute
	clock := &fakeThrottleClock{now: time.Now()}
	_, h := newThrottleTestServer(cfg, clock, http.StatusUnauthorized)

	doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login", `{"email":"a@x.com"}`, "")
	clock.advance(5 * time.Minute)
	rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login", `{"email":"a@x.com"}`, "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("locked request got %d, want 429", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "600" {
		t.Fatalf("Retry-After = %q, want 600 after 5 of 15 minutes elapsed", got)
	}
}

func TestAuthThrottleSkipsUnlistedRoutes(t *testing.T) {
	cfg := throttleTestConfig()
	cfg.failLimit = 1
	cfg.ipLimit = 1
	cfg.sendLimit = 1
	clock := &fakeThrottleClock{now: time.Now()}
	_, h := newThrottleTestServer(cfg, clock, http.StatusOK)

	paths := []string{
		"/api/v1/auth/logout",
		"/api/v1/auth/config",
		"/api/v1/hooks/github",
		"/api/v1/ideas",
	}
	for _, path := range paths {
		for i := 0; i < 5; i++ {
			rec := doThrottledRequest(h, http.MethodGet, path, "", "192.0.2.1")
			if rec.Code != http.StatusOK {
				t.Fatalf("%s request %d got %d, want 200 (unthrottled route)", path, i, rec.Code)
			}
		}
	}
}

func TestAuthThrottleOnlyCountsUnauthorizedForCredentials(t *testing.T) {
	cfg := throttleTestConfig()
	cfg.failLimit = 2
	clock := &fakeThrottleClock{now: time.Now()}
	s := &Server{authThrottle: newAuthThrottle(cfg, clock.time)}
	stubStatus := http.StatusBadRequest // e.g. "invalid request body"
	stub := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(stubStatus) })
	h := s.withAuthThrottle(stub)

	// 400s are client errors but not credential failures: they must not
	// build toward the lockout.
	for i := 0; i < 3; i++ {
		if rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login", `{"email":"a@x.com"}`, ""); rec.Code != http.StatusBadRequest {
			t.Fatalf("request %d got %d, want 400", i, rec.Code)
		}
	}

	stubStatus = http.StatusUnauthorized
	for i := 0; i < 2; i++ {
		if rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login", `{"email":"a@x.com"}`, ""); rec.Code != http.StatusUnauthorized {
			t.Fatalf("credential failure %d got %d, want 401", i, rec.Code)
		}
	}
	if rec := doThrottledRequest(h, http.MethodPost, "/api/v1/auth/login", `{"email":"a@x.com"}`, ""); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("after failLimit 401s got %d, want 429", rec.Code)
	}
}
