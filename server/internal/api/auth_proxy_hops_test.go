package api

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/config"
)

// rv4-SEC-02, the other half: INSIDEOUT_TRUSTED_PROXY_HOPS says how many proxies
// in front of the server append to X-Forwarded-For, and clientIP takes that Nth
// entry from the right. Everything left of it is the client's to write.

func requestWith(remote string, forwarded ...string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	r.RemoteAddr = remote
	for _, line := range forwarded {
		r.Header.Add("X-Forwarded-For", line)
	}
	return r
}

func TestClientIPDefaultsToThePeer(t *testing.T) {
	r := requestWith("10.0.0.1:40000", "1.1.1.1, 2.2.2.2")
	for name, s := range map[string]*Server{
		"no config":     {},
		"zero hops":     {cfg: &config.Config{}},
		"explicit zero": {cfg: &config.Config{TrustedProxyHops: 0}},
	} {
		if got := s.clientIP(r); got != "10.0.0.1" {
			t.Errorf("%s: clientIP = %q, want the TCP peer 10.0.0.1 (no header is trusted)", name, got)
		}
	}
}

func TestClientIPTakesTheNthEntryFromTheRight(t *testing.T) {
	const chain = "6.6.6.6, 203.0.113.1, 198.51.100.2, 192.0.2.3"
	for hops, want := range map[int]string{1: "192.0.2.3", 2: "198.51.100.2", 3: "203.0.113.1", 4: "6.6.6.6"} {
		if got := clientIPBehind(hops, requestWith("10.0.0.1:40000", chain)); got != want {
			t.Errorf("hops=%d: clientIP = %q, want %q", hops, got, want)
		}
	}
}

// A repeated header is one list: a proxy may append a header line of its own,
// and the client may send a first line of its own. Reading only the first line
// would hand the client the key.
func TestClientIPJoinsRepeatedForwardedForLines(t *testing.T) {
	r := requestWith("10.0.0.1:40000", "9.9.9.9", "203.0.113.5")
	if got := clientIPBehind(1, r); got != "203.0.113.5" {
		t.Errorf("hops=1 over two lines: clientIP = %q, want the proxy's line 203.0.113.5, not the client's 9.9.9.9", got)
	}
	if got := clientIPBehind(2, r); got != "9.9.9.9" {
		t.Errorf("hops=2 over two lines: clientIP = %q, want 9.9.9.9", got)
	}
}

// When the count and the chain disagree, the answer is the shared peer bucket,
// never a string the client chose.
func TestClientIPFallsBackToThePeerWhenTheChainDoesNotFit(t *testing.T) {
	cases := []struct {
		name string
		hops int
		xff  string
	}{
		{"fewer entries than hops", 2, "203.0.113.5"},
		{"no header at all", 1, ""},
		{"entry is not an address", 1, "6.6.6.6, unknown"},
		{"entry carries a port", 1, "203.0.113.5:4711"},
		{"only empty entries", 1, ", ,"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := clientIPBehind(tc.hops, requestWith("10.0.0.1:40000", tc.xff)); got != "10.0.0.1" {
				t.Errorf("clientIP = %q, want the peer 10.0.0.1", got)
			}
		})
	}
}

func TestClientIPNormalizesTheAddressItKeysOn(t *testing.T) {
	for xff, want := range map[string]string{
		"::ffff:203.0.113.5": "203.0.113.5", // a v4 client seen through a dual-stack listener
		"2001:DB8::1":        "2001:db8::1", // one client, one key, whatever the spelling
	} {
		if got := clientIPBehind(1, requestWith("10.0.0.1:40000", xff)); got != want {
			t.Errorf("X-Forwarded-For %q: clientIP = %q, want %q", xff, got, want)
		}
	}
}

// Hosted shape: one trusted proxy appends the real client to whatever the
// client sent. Rotating the forged left side must not mint buckets; a second
// real client behind the same proxy keeps its own.
func TestThrottleBehindOneProxyKeysOnTheAppendedEntry(t *testing.T) {
	cfg := throttleTestConfig()
	cfg.ipLimit = 2
	s, h := newThrottleTestServer(cfg, &fakeThrottleClock{now: time.Now()}, http.StatusOK)
	s.cfg = &config.Config{TrustedProxyHops: 1}

	for i := 1; i <= 2; i++ {
		forged := fmt.Sprintf("%d.%d.%d.%d, 203.0.113.9", i, i, i, i)
		if rec := peerPost(h, "/api/v1/auth/register", `{}`, forged); rec.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200 before the cap", i, rec.Code)
		}
	}
	if rec := peerPost(h, "/api/v1/auth/register", `{}`, "7.7.7.7, 203.0.113.9"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("third request from the proxy-appended client 203.0.113.9: status = %d, want 429 (the forged left side is not part of the key)", rec.Code)
	}
	if rec := peerPost(h, "/api/v1/auth/register", `{}`, "7.7.7.7, 203.0.113.10"); rec.Code != http.StatusOK {
		t.Fatalf("a different client behind the same proxy: status = %d, want 200 (its own bucket)", rec.Code)
	}
}

// Behind a proxy with no hop count configured every client shares the proxy's
// bucket, which is quiet and wrong; the server says so once.
func TestIgnoredForwardedForIsLoggedOnce(t *testing.T) {
	var logs bytes.Buffer
	newServer := func(hops int) *Server {
		return NewServer(nil, nil, &config.Config{TrustedProxyHops: hops}, slog.New(slog.NewTextHandler(&logs, nil)), nil, nil)
	}
	count := func() int { return strings.Count(logs.String(), "X-Forwarded-For ignored") }

	s := newServer(0)
	s.clientIP(requestWith("10.0.0.1:40000"))
	if count() != 0 {
		t.Fatalf("a request without the header logged a warning:\n%s", logs.String())
	}
	s.clientIP(requestWith("10.0.0.1:40000", "1.1.1.1"))
	s.clientIP(requestWith("10.0.0.1:40000", "2.2.2.2"))
	if count() != 1 {
		t.Fatalf("two requests with an ignored header logged %d warnings, want exactly 1:\n%s", count(), logs.String())
	}
	if !strings.Contains(logs.String(), "INSIDEOUT_TRUSTED_PROXY_HOPS") {
		t.Errorf("the warning must name the setting to change:\n%s", logs.String())
	}

	before := count()
	newServer(1).clientIP(requestWith("10.0.0.1:40000", "1.1.1.1"))
	if count() != before {
		t.Errorf("a server that trusts one hop warned about a header it uses:\n%s", logs.String())
	}
}
