package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// authResendCooldown is the minimum interval between emailed auth sends
// (magic link, password reset, login code) for one email address or one
// client IP. It bounds mailbox spam from a forgotten-password retry loop
// without ever answering differently — the endpoints keep their uniform
// 200 either way.
const authResendCooldown = 60 * time.Second

// sendCooldown is a fixed-window resend gate keyed twice: per email and
// per client IP. A send proceeds only when BOTH keys have been quiet for
// the whole window, and the window anchors at the last actual send — a
// blocked attempt never re-stamps the keys, so re-requesting every 59
// seconds cannot freeze a victim's cooldown indefinitely.
type sendCooldown struct {
	mu     sync.Mutex
	window time.Duration
	now    func() time.Time
	emails map[string]time.Time
	ips    map[string]time.Time
}

func newSendCooldown(window time.Duration) *sendCooldown {
	return &sendCooldown{
		window: window,
		now:    time.Now,
		emails: map[string]time.Time{},
		ips:    map[string]time.Time{},
	}
}

// allow reports whether a send to email from ip may proceed now, stamping
// both keys when it does. A key never used — or last sent outside the
// window — reads as the zero time and is long expired.
func (c *sendCooldown) allow(email, ip string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if now.Sub(c.emails[email]) < c.window {
		return false
	}
	if now.Sub(c.ips[ip]) < c.window {
		return false
	}
	c.emails[email] = now
	c.ips[ip] = now
	return true
}

// cooldownClientIP identifies the requesting client for the resend gate.
// The app is served behind a proxy that appends the client address to
// X-Forwarded-For, so the rightmost entry is the trustworthy one; the
// host part of RemoteAddr covers direct local traffic. It mirrors the
// throttle middleware's IP extraction on purpose — a spoofed leftmost
// XFF hop must not buy a fresh cooldown budget.
func cooldownClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[len(parts)-1]); ip != "" {
			return ip
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
