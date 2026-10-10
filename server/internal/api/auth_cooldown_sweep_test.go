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
// already read as expired, so a key still inside its window survives it
// and keeps blocking from either side (same email, same IP).
func TestSendCooldownSweepKeepsLiveKeys(t *testing.T) {
	clock := newFakeClock()
	cd := newSendCooldown(authResendCooldown)
	cd.now = clock.now

	stampKeys(cd, overCap) // stale batch
	clock.advance(authResendCooldown / 2)
	cd.allow("live@x.co", "192.0.2.1")
	clock.advance(authResendCooldown/2 + time.Second) // stale batch expired, live key has 29s left
	cd.allow("trigger@x.co", "192.0.2.2")             // over the cap: this call sweeps

	if len(cd.emails) != 2 || len(cd.ips) != 2 {
		t.Fatalf("tracking %d emails and %d ips, want the live and trigger keys only: the stale batch survived", len(cd.emails), len(cd.ips))
	}
	if cd.allow("live@x.co", "203.0.113.5") {
		t.Fatal("a live email key stopped blocking after the sweep")
	}
	if cd.allow("other@x.co", "192.0.2.1") {
		t.Fatal("a live ip key stopped blocking after the sweep")
	}
}
