package presence

import (
	"testing"
	"time"
)

func newTestRegistry() (*Registry, *time.Time) {
	base := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)
	clock := base
	r := New(30*time.Second, func() time.Time { return clock })
	return r, &clock
}

func TestTouchListPrune(t *testing.T) {
	r, clock := newTestRegistry()
	r.Touch("p", "s1", "u1", "A")
	r.Touch("p", "s2", "u2", "B")
	if got := r.List("p"); len(got) != 2 {
		t.Fatalf("want 2 entries, got %v", got)
	}
	*clock = clock.Add(45 * time.Second) // s1 went stale
	if got := r.Touch("p", "s2", "u2", "B"); len(got) != 1 || got[0].SessionID != "s2" {
		t.Fatalf("prune failed: %v", got)
	}
}

func TestLeaveAndEmpty(t *testing.T) {
	r, _ := newTestRegistry()
	r.Touch("p", "s1", "u1", "A")
	r.Leave("p", "s1")
	if got := r.List("p"); len(got) != 0 {
		t.Fatalf("want empty after leave, got %v", got)
	}
}

func TestSubscribeReceivesChanges(t *testing.T) {
	r, _ := newTestRegistry()
	ch, cancel := r.Subscribe("p")
	defer cancel()
	r.Touch("p", "s1", "u1", "A")
	select {
	case snap := <-ch:
		if len(snap) != 1 || snap[0].Name != "A" {
			t.Fatalf("bad snapshot: %v", snap)
		}
	case <-time.After(time.Second):
		t.Fatal("no snapshot delivered")
	}
	r.Leave("p", "s1")
	select {
	case snap := <-ch:
		if len(snap) != 0 {
			t.Fatalf("leave not broadcast: %v", snap)
		}
	case <-time.After(time.Second):
		t.Fatal("leave not delivered")
	}
}

func TestListIsDeterministic(t *testing.T) {
	r, _ := newTestRegistry()
	r.Touch("p", "zz", "u", "Z")
	r.Touch("p", "aa", "u", "A")
	got := r.List("p")
	if got[0].SessionID != "aa" {
		t.Fatalf("order not deterministic: %v", got)
	}
}
