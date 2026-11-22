package roadmaptime

import (
	"testing"
	"time"
)

func TestPressure(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	cases := map[string]time.Time{
		Overdue:  now.Add(-time.Hour),
		HighRisk: now.Add(24 * time.Hour),
		Near:     now.Add(3 * 24 * time.Hour),
		Normal:   now.Add(30 * 24 * time.Hour),
	}
	for want, dl := range cases {
		if got := Pressure(dl, now); got != want {
			t.Errorf("Pressure(%v) = %s, want %s", dl, got, want)
		}
	}
	if Pressure(time.Time{}, now) != "" {
		t.Error("zero deadline should have no pressure")
	}
}

func TestNodePressureFinishedWorkHasNone(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	late := now.Add(-time.Hour)
	for _, s := range []string{"done", "validated"} {
		if got := NodePressure(s, late, now); got != "" {
			t.Errorf("NodePressure(%s, overdue) = %q, want none", s, got)
		}
	}
	if got := NodePressure("in_progress", late, now); got != Overdue {
		t.Errorf("NodePressure(in_progress, overdue) = %q, want %s", got, Overdue)
	}
}

func ptr(t time.Time) *time.Time { return &t }

func TestAssemble(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	nodes := []Node{
		{ID: "a", Title: "active with deadline", Status: "in_progress", Deadline: ptr(now.Add(24 * time.Hour)), Leaf: true},
		{ID: "b", Title: "active no deadline", Status: "in_progress", Leaf: true},
		{ID: "c", Title: "pending near", Status: "pending", Deadline: ptr(now.Add(2 * 24 * time.Hour)), Leaf: true},
		{ID: "d", Title: "pending far", Status: "pending", Deadline: ptr(now.Add(20 * 24 * time.Hour)), Leaf: true},
		{ID: "e", Title: "pending no deadline", Status: "pending", Leaf: true},
		{ID: "f", Title: "done leaf", Status: "done", Leaf: true},
		{ID: "g", Title: "branch active", Status: "in_progress", Deadline: ptr(now), Leaf: false},
	}
	p := Assemble(nodes, now)
	if len(p.Now) != 1 || p.Now[0].ID != "a" || p.Now[0].Pressure != HighRisk {
		t.Errorf("Now wrong: %+v", p.Now)
	}
	if len(p.NeedsDeadline) != 1 || p.NeedsDeadline[0].ID != "b" {
		t.Errorf("NeedsDeadline wrong: %+v", p.NeedsDeadline)
	}
	if len(p.Next) != 2 || p.Next[0].ID != "c" || p.Next[1].ID != "d" {
		t.Errorf("Next wrong (order/coverage): %+v", p.Next)
	}
	if p.DoneCount != 1 {
		t.Errorf("DoneCount wrong: %d", p.DoneCount)
	}
}

func TestNextCapsAtThree(t *testing.T) {
	now := time.Now()
	var nodes []Node
	for i := 0; i < 5; i++ {
		nodes = append(nodes, Node{ID: string(rune('a' + i)), Status: "pending",
			Deadline: ptr(now.Add(time.Duration(i+1) * time.Hour)), Leaf: true})
	}
	if got := len(Assemble(nodes, now).Next); got != 3 {
		t.Errorf("Next should cap at 3, got %d", got)
	}
}

// Next puts ready work before work that waits (2026-11-22 plan, S2): on the
// Spark Matching branch "Daily drop UI" is due first but waits on the
// in-progress "Compatibility scoring", so the ready items lead; each group
// keeps the deadline order.
func TestNextPutsReadyItemsBeforeWaitingOnes(t *testing.T) {
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	day := func(d int) *time.Time { return ptr(now.Add(time.Duration(d) * 24 * time.Hour)) }
	scoring := Ref{ID: "cs", Title: "Compatibility scoring"}
	nodes := []Node{
		{ID: "cs", Title: "Compatibility scoring", Status: "in_progress", Deadline: day(4), Leaf: true},
		{ID: "dd", Title: "Daily drop UI", Status: "pending", Deadline: day(9), Leaf: true, WaitsOn: []Ref{scoring}},
		{ID: "me", Title: "Match explanation copy", Status: "pending", Deadline: day(18), Leaf: true},
		{ID: "ps", Title: "Prompt picker", Status: "pending", Deadline: day(12), Leaf: true},
	}
	p := Assemble(nodes, now)
	got := []string{}
	for _, it := range p.Next {
		got = append(got, it.ID)
	}
	if len(got) != 3 || got[0] != "ps" || got[1] != "me" || got[2] != "dd" {
		t.Fatalf("Next = %v, want ready ps, me, then the waiting dd", got)
	}
	if w := p.Next[2].WaitsOn; len(w) != 1 || w[0] != scoring {
		t.Errorf("Daily drop UI waitsOn = %+v, want Compatibility scoring", w)
	}
	if p.Next[0].WaitsOn != nil {
		t.Errorf("a ready item carries no waitsOn: %+v", p.Next[0])
	}
}
