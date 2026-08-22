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
