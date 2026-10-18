package api

import (
	"maps"
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
// per client IP (clientIP, shared with the auth throttle: one extraction,
// so the two gates cannot disagree about who the caller is). A send
// proceeds only when BOTH keys have been quiet for the whole window, and the
// window anchors at the last actual send — a blocked attempt never
// re-stamps the keys, so re-requesting every 59 seconds cannot freeze a
// victim's cooldown indefinitely.
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

// cooldownMaxEntries bounds map growth; past it, the next allow sweeps the
// keys already outside the window. Live keys are never dropped (they are
// the gate), so the maps pass the cap only while that many keys are
// genuinely inside one window: 10k sends a minute is far beyond plausible.
const cooldownMaxEntries = 10_000

// allow reports whether a send to email from ip may proceed now, stamping
// both keys when it does. A key never used — or last sent outside the
// window — reads as the zero time and is long expired.
func (c *sendCooldown) allow(email, ip string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if len(c.emails) > cooldownMaxEntries || len(c.ips) > cooldownMaxEntries {
		c.sweepLocked(now)
	}
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

// sweepLocked drops every key that already reads as expired, so it cannot
// change what allow reports.
func (c *sendCooldown) sweepLocked(now time.Time) {
	expired := func(_ string, t time.Time) bool { return now.Sub(t) >= c.window }
	maps.DeleteFunc(c.emails, expired)
	maps.DeleteFunc(c.ips, expired)
}
