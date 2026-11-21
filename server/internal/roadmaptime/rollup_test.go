package roadmaptime

import (
	"testing"
	"time"
)

func TestRollupsCountLeavesAtEveryDepth(t *testing.T) {
	soon := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	later := soon.Add(48 * time.Hour)
	got := Rollups([]TreeNode{
		{ID: "root"},
		{ID: "a", ParentID: "root"},
		{ID: "a1", ParentID: "a", Status: "done", Deadline: &soon},
		{ID: "a2", ParentID: "a", Status: "validated"},
		{ID: "b", ParentID: "root", Status: "pending", Deadline: &later},
	})
	root := got["root"]
	if root.Leaves != 3 || root.Done != 2 || root.Validated != 1 || root.EarliestDeadline == nil || !root.EarliestDeadline.Equal(later) {
		t.Errorf("root: %+v, want 3 leaves, 2 done, 1 validated, earliest open deadline %v", root, later)
	}
	if a := got["a"]; a.Leaves != 2 || a.Done != 2 || a.EarliestDeadline != nil {
		t.Errorf("a: %+v, want 2 finished leaves and no open deadline", a)
	}
	if b := got["b"]; b.Leaves != 1 || b.Done != 0 {
		t.Errorf("leaf b: %+v, want itself", b)
	}
	if _, ok := got["missing-parent-is-fine"]; ok {
		t.Error("an unlisted id got a roll-up")
	}
}
