// Package roadmaptime implements PRODUCT.md's time-first rules:
// deadline pressure states (normal → near → high risk → overdue) and
// the Progress view (Now dominant, at most three justified Next, Done
// with its evidence). Pure: callers pass plain values and a clock.
package roadmaptime

import (
	"sort"
	"time"
)

// Pressure states, in the order PRODUCT.md escalates them.
const (
	Normal   = "normal"
	Near     = "near"
	HighRisk = "high_risk"
	Overdue  = "overdue"
)

// Pressure classifies a deadline against now.
func Pressure(deadline, now time.Time) string {
	if deadline.IsZero() {
		return ""
	}
	d := deadline.Sub(now)
	switch {
	case d < 0:
		return Overdue
	case d <= 48*time.Hour:
		return HighRisk
	case d <= 7*24*time.Hour:
		return Near
	default:
		return Normal
	}
}

// Node is the reduced node shape the Progress assembly needs.
type Node struct {
	ID         string
	Title      string
	Status     string
	Deadline   *time.Time
	Leaf       bool
	Acceptance string
	OwnerName  string
}

// ProgressItem is one entry in the Progress view.
type ProgressItem struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`
	Deadline   *time.Time `json:"deadline,omitempty"`
	Pressure   string     `json:"pressure,omitempty"`
	Acceptance string     `json:"acceptance,omitempty"`
	OwnerName  string     `json:"ownerName,omitempty"`
}

// Progress is PRODUCT.md's default Roadmap view: Now is dominant,
// Next shows at most three justified (deadlined) items, Done counts
// with its evidence. Work without a deadline cannot enter Now —
// in-progress items missing one are surfaced as NeedsDeadline instead.
type Progress struct {
	Now           []ProgressItem `json:"now"`
	NeedsDeadline []ProgressItem `json:"needsDeadline"`
	Next          []ProgressItem `json:"next"`
	DoneCount     int            `json:"doneCount"`
	// DoneItems lists the finished leaves with their acceptance and owner,
	// so Done answers what was finished and by whom, not only how many.
	DoneItems []ProgressItem `json:"doneItems"`
}

// Assemble builds the Progress view. Now: in-progress leaves with a
// deadline (pressure computed). Next: pending leaves with deadlines,
// earliest three. Done: the finished leaves, listed.
func Assemble(nodes []Node, now time.Time) Progress {
	p := Progress{Now: []ProgressItem{}, NeedsDeadline: []ProgressItem{}, Next: []ProgressItem{}, DoneItems: []ProgressItem{}}
	var pending []Node
	for _, n := range nodes {
		if !n.Leaf {
			continue
		}
		switch n.Status {
		case "in_progress":
			item := item(n, now)
			if n.Deadline == nil {
				p.NeedsDeadline = append(p.NeedsDeadline, ProgressItem{ID: n.ID, Title: n.Title, Status: n.Status, Acceptance: n.Acceptance, OwnerName: n.OwnerName})
			} else {
				p.Now = append(p.Now, item)
			}
		case "pending":
			if n.Deadline != nil {
				pending = append(pending, n)
			}
		case "done", "validated":
			// A validated leaf is a finished leaf plus its human confirm;
			// Done lists both or finished work would vanish on validate.
			p.DoneCount++
			p.DoneItems = append(p.DoneItems, item(n, now))
		}
	}
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Deadline.Before(*pending[j].Deadline)
	})
	for i := 0; i < len(pending) && i < 3; i++ {
		p.Next = append(p.Next, item(pending[i], now))
	}
	return p
}

func item(n Node, now time.Time) ProgressItem {
	it := ProgressItem{ID: n.ID, Title: n.Title, Status: n.Status, Deadline: n.Deadline, Acceptance: n.Acceptance, OwnerName: n.OwnerName}
	if n.Deadline != nil {
		it.Pressure = Pressure(*n.Deadline, now)
	}
	return it
}
