package api

import (
	"fmt"
	"testing"
	"time"
)

// overCap is one past cooldownMaxEntries: enough keys that the next allow()
// is allowed to sweep.
const overCap = 10_001

// stampKeys has n distinct (email, ip) pairs send at the clock's current
// time — the keys a long-lived process piles up, one per address that ever
// asked for a send.
func stampKeys(cd *sendCooldown, n int) {
	for i := 0; i < n; i++ {
		cd.allow(fmt.Sprintf("user%d@x.co", i), fmt.Sprintf("10.0.%d.%d", i>>8, i&255))
	}
}

// TestSendCooldownSweepsExpiredKeysPastTheCap: allow() only ever inserted
// keys, so every distinct email and client IP that ever asked for a send
// stayed in the maps for the life of the process. Past the cap, the next
// call must shed the keys whose window has passed.
func TestSendCooldownSweepsExpiredKeysPastTheCap(t *testing.T) {
	clock := newFakeClock()
	cd := newSendCooldown(authResendCooldown)
	cd.now = clock.now

	stampKeys(cd, overCap)
	clock.advance(authResendCooldown + time.Second)
	if !cd.allow("fresh@x.co", "192.0.2.1") {
		t.Fatal("a send after the window must be allowed")
	}
	if len(cd.emails) != 1 || len(cd.ips) != 1 {
		t.Fatalf("tracking %d emails and %d ips, want only the one live key each: expired keys are never deleted", len(cd.emails), len(cd.ips))
	}
}

// TestSendCooldownSweepKeepsLiveKeys: the sweep may only drop keys that
// already read as expired. Every call over the cap sweeps, so a key still
// inside its window must survive one and keep blocking from either side
// (same email, same IP), both while the whole batch is live and once it
// has expired around a newer key.
func TestSendCooldownSweepKeepsLiveKeys(t *testing.T) {
	clock := newFakeClock()
	cd := newSendCooldown(authResendCooldown)
	cd.now = clock.now

	stampKeys(cd, overCap) // the maps are over the cap from here on
	clock.advance(authResendCooldown / 2)
	cd.allow("live@x.co", "192.0.2.1") // sweeps with the whole batch half a window old
	if cd.allow("user0@x.co", "203.0.113.5") {
		t.Fatal("the sweep dropped a key still inside its window")
	}

	clock.advance(authResendCooldown/2 + time.Second) // batch expired, live key has 29s left
	cd.allow("trigger@x.co", "192.0.2.2")             // sweeps again, now with stale keys to drop

	if cd.allow("live@x.co", "203.0.113.5") {
		t.Fatal("a live email key stopped blocking after the sweep")
	}
	if cd.allow("other@x.co", "192.0.2.1") {
		t.Fatal("a live ip key stopped blocking after the sweep")
	}
	if len(cd.emails) != 2 || len(cd.ips) != 2 {
		t.Fatalf("tracking %d emails and %d ips, want the live and trigger keys only: the stale batch survived", len(cd.emails), len(cd.ips))
	}
}

// TestSendCooldownSweepsWhicheverMapOutgrowsTheCap: the maps grow
// independently (an office behind one NAT adds emails only, a botnet aimed
// at one mailbox adds IPs only), so either one past the cap must trigger
// the sweep.
func TestSendCooldownSweepsWhicheverMapOutgrowsTheCap(t *testing.T) {
	cases := []struct {
		name  string
		email func(i int) string
		ip    func(i int) string
	}{
		{"many emails, one ip",
			func(i int) string { return fmt.Sprintf("user%d@x.co", i) },
			func(int) string { return "192.0.2.1" }},
		{"one email, many ips",
			func(int) string { return "victim@x.co" },
			func(i int) string { return fmt.Sprintf("10.0.%d.%d", i>>8, i&255) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clock := newFakeClock()
			cd := newSendCooldown(authResendCooldown)
			cd.now = clock.now

			// One send per window, so each earlier key has expired by the
			// next call. The last send finds one map over the cap and sweeps.
			for i := 0; i <= overCap; i++ {
				if !cd.allow(tc.email(i), tc.ip(i)) {
					t.Fatalf("send %d was blocked, want every spaced send allowed", i)
				}
				clock.advance(authResendCooldown + time.Second)
			}
			if len(cd.emails) != 1 || len(cd.ips) != 1 {
				t.Fatalf("tracking %d emails and %d ips, want 1 each: the map past the cap was never swept", len(cd.emails), len(cd.ips))
			}
		})
	}
}
