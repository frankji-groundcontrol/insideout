package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
)

// authThrottleRules maps each auth endpoint to how attempts are counted.
// Anything not listed (logout, the public auth/config, the GitHub webhook,
// every non-auth route) passes through untouched. Addresses issue
// 2026-08-24-auth-endpoints-no-rate-limiting.
var authThrottleRules = map[string]throttleCategory{
	"/api/v1/auth/login":              throttleCredentials, // password guesses
	"/api/v1/auth/login-code/verify":  throttleCredentials, // 6-digit code guesses
	"/api/v1/auth/magic-link":         throttleSends,       // email-bomb / mail spend
	"/api/v1/auth/forgot-password":    throttleSends,
	"/api/v1/auth/login-code":         throttleSends,
	"/api/v1/auth/register":           throttleVolume,
	"/api/v1/auth/refresh":            throttleVolume,
	"/api/v1/auth/magic-link/consume": throttleVolume, // token probes cost a DB write
	"/api/v1/auth/reset-password":     throttleVolume,
}

type throttleCategory int

const (
	// throttleCredentials counts only failed responses (401) against
	// email+IP and IP keys; failLimit failures within failWindow lock both
	// keys for lockout. A success (2xx) clears them.
	throttleCredentials throttleCategory = iota
	// throttleSends counts every request: sendLimit per sendWindow per
	// email+IP, with the per-IP cap as a backstop — each hit to a real
	// address costs a sent email.
	throttleSends
	// throttleVolume counts every request against the IP key only.
	throttleVolume
)

// throttleConfig holds every limit and window the throttle applies; a value
// with zero fields disables nothing gracefully, so constructors must come
// from defaultThrottleConfig (tests shrink the numbers and inject a fake
// clock instead of sleeping).
type throttleConfig struct {
	failLimit  int           // credential failures within failWindow before lockout
	failWindow time.Duration // how long a failure counts for
	lockout    time.Duration // how long a tripped key refuses attempts
	sendLimit  int           // send requests per sendWindow per email+IP
	sendWindow time.Duration
	ipLimit    int // requests per ipWindow per IP (backstop for every category)
	ipWindow   time.Duration
}

// defaultThrottleConfig follows the issue's target design: docmind's 8
// failures -> 15-minute lockout on credentials, 3 sends / 10 minutes per
// email on the mail endpoints, plus a per-IP backstop so an attacker
// rotating addresses can't turn volume abuse into unlimited probing.
func defaultThrottleConfig() throttleConfig {
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

type failEntry struct {
	count       int
	windowStart time.Time
	lockedUntil time.Time
}

type hitEntry struct {
	count       int
	windowStart time.Time
}

// authThrottle is the in-memory per-email+IP / per-IP counter behind
// withAuthThrottle. Two maps: failure state for credential lockouts, fixed
// windows for send/volume caps. The clock is injectable so tests can run
// without sleeping.
type authThrottle struct {
	mu    sync.Mutex
	cfg   throttleConfig
	now   func() time.Time
	fails map[string]*failEntry
	hits  map[string]*hitEntry
}

func newAuthThrottle(cfg throttleConfig, now func() time.Time) *authThrottle {
	if now == nil {
		now = time.Now
	}
	return &authThrottle{cfg: cfg, now: now, fails: make(map[string]*failEntry), hits: make(map[string]*hitEntry)}
}

// throttleMaxEntries bounds map growth; past it, the next mutation sweeps
// expired entries. 10k live auth attempts is already far beyond plausible.
const throttleMaxEntries = 10_000

// locked reports whether a credential key is in lockout, and for how much
// longer.
func (t *authThrottle) locked(key string) (bool, time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	e := t.fails[key]
	if e == nil {
		return false, 0
	}
	if rem := e.lockedUntil.Sub(t.now()); rem > 0 {
		return true, rem
	}
	return false, 0
}

// recordFail adds a failure to a credential key; reaching failLimit trips a
// lockout and starts a fresh window once it expires.
func (t *authThrottle) recordFail(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.fails) > throttleMaxEntries {
		t.evictFailsLocked()
	}
	now := t.now()
	e := t.fails[key]
	if e == nil || now.Sub(e.windowStart) >= t.cfg.failWindow {
		e = &failEntry{windowStart: now}
		t.fails[key] = e
	}
	e.count++
	if e.count >= t.cfg.failLimit {
		e.lockedUntil = now.Add(t.cfg.lockout)
		e.count = 0
		e.windowStart = now
	}
}

// clearFails drops failure state after a success, so one bad streak can't
// hang over a user who then signs in normally.
func (t *authThrottle) clearFails(keys ...string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, k := range keys {
		delete(t.fails, k)
	}
}

// takeHit consumes one request from a fixed window; over the limit it
// reports how long until the window resets.
func (t *authThrottle) takeHit(key string, limit int, window time.Duration) (bool, time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.hits) > throttleMaxEntries {
		t.evictHitsLocked(window)
	}
	now := t.now()
	e := t.hits[key]
	if e == nil || now.Sub(e.windowStart) >= window {
		e = &hitEntry{windowStart: now}
		t.hits[key] = e
	}
	e.count++
	if e.count > limit {
		return true, window - now.Sub(e.windowStart)
	}
	return false, 0
}

func (t *authThrottle) evictFailsLocked() {
	now := t.now()
	for k, e := range t.fails {
		if now.Sub(e.windowStart) >= t.cfg.failWindow && !now.Before(e.lockedUntil) {
			delete(t.fails, k)
		}
	}
}

func (t *authThrottle) evictHitsLocked(window time.Duration) {
	now := t.now()
	for k, e := range t.hits {
		if now.Sub(e.windowStart) >= window {
			delete(t.hits, k)
		}
	}
}

// withAuthThrottle rate-limits the auth endpoints listed in
// authThrottleRules. Credential endpoints lock after failLimit 401s per
// email+IP (and per IP); mail-sending endpoints cap sends per email+IP; all
// of it has a per-IP backstop. The 429 is the same uniform answer for known
// and unknown accounts, so the throttle never doubles as an
// account-existence oracle.
func (s *Server) withAuthThrottle(next http.Handler) http.Handler {
	// Constructing this middleware is part of startup (main builds the
	// handler chain once), which is when the single-replica contract that
	// all this in-process state relies on gets announced — single_replica.go.
	s.startSingletonGuard()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		category, ok := authThrottleRules[r.URL.Path]
		if !ok || r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}

		ip := clientIP(r)
		email := ""
		if category == throttleCredentials || category == throttleSends {
			email = throttleEmail(w, r)
		}
		ipKey := "i|" + ip
		accountKey := "e|" + email + "|" + ip

		if category == throttleCredentials {
			for _, key := range []string{accountKey, ipKey} {
				if locked, retryIn := s.authThrottle.locked(key); locked {
					writeThrottled(w, retryIn)
					return
				}
			}
		} else {
			// Every request counts against the send/volume windows whether
			// or not the handler would have accepted it.
			if category == throttleSends {
				if over, retryIn := s.authThrottle.takeHit(accountKey, s.authThrottle.cfg.sendLimit, s.authThrottle.cfg.sendWindow); over {
					writeThrottled(w, retryIn)
					return
				}
			}
			if over, retryIn := s.authThrottle.takeHit(ipKey, s.authThrottle.cfg.ipLimit, s.authThrottle.cfg.ipWindow); over {
				writeThrottled(w, retryIn)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		switch {
		case rec.status == http.StatusUnauthorized:
			s.authThrottle.recordFail(ipKey)
			s.authThrottle.recordFail(accountKey)
		case rec.status < http.StatusBadRequest:
			s.authThrottle.clearFails(ipKey, accountKey)
		}
	})
}

// writeThrottled is the uniform 429 every throttled auth endpoint answers;
// Retry-After carries the remaining lockout/window in whole seconds.
func writeThrottled(w http.ResponseWriter, retryIn time.Duration) {
	secs := int64((retryIn + time.Second - 1) / time.Second)
	if secs < 1 {
		secs = 1
	}
	w.Header().Set("Retry-After", strconv.FormatInt(secs, 10))
	httpx.WriteError(w, http.StatusTooManyRequests, "too many attempts, try again later", "", nil)
}

// throttleEmail reads the JSON body's email field (the only per-account key
// the throttle needs) and restores the body for the handler. A missing or
// unreadable email just means the per-account key is empty — the request
// still counts against the IP backstop.
func throttleEmail(w http.ResponseWriter, r *http.Request) string {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil {
		return ""
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	var payload struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(payload.Email))
}

// clientIP returns the caller's IP for the auth throttle and resend
// cooldown. The deployment chain is browser → Railway edge → nginx →
// server: the Railway edge APPENDS the real client IP to X-Forwarded-For
// (client-supplied spoofed values stay to its left), and nginx passes that
// chain through untouched (client/nginx.conf, REL-13) — so the rightmost
// non-empty entry is the edge-appended client IP and client-prepended
// garbage is discarded by taking the tail. Local compose runs have no edge
// in front of nginx, so a locally-spoofed XFF is trusted there — dev only.
// Without the header (direct local runs, tests) RemoteAddr is used.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			if ip := strings.TrimSpace(parts[i]); ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
