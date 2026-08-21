package prdcommit

import "testing"

func TestDiff(t *testing.T) {
	prev := map[string]string{"problem": "old problem", "keep": "same", "gone": "x"}
	curr := map[string]string{"problem": "new problem text", "keep": "same", "fresh": "y"}
	d := Diff(prev, curr)
	secs := d["sections"].(map[string]any)
	if secs["keep"] != nil {
		t.Errorf("unchanged section should be absent: %v", secs)
	}
	if secs["problem"].(map[string]any)["change"] != "changed" {
		t.Errorf("problem should be changed: %v", secs["problem"])
	}
	if secs["fresh"].(map[string]any)["change"] != "added" {
		t.Errorf("fresh should be added: %v", secs["fresh"])
	}
	if secs["gone"].(map[string]any)["change"] != "removed" {
		t.Errorf("gone should be removed: %v", secs["gone"])
	}
	c := d["counts"].(map[string]int)
	if c["added"] != 1 || c["removed"] != 1 || c["changed"] != 1 {
		t.Errorf("counts wrong: %v", c)
	}
}

func TestDiffDeterministic(t *testing.T) {
	a := Diff(map[string]string{"b": "1", "a": "2"}, map[string]string{"a": "3", "c": "4"})
	b := Diff(map[string]string{"a": "2", "b": "1"}, map[string]string{"c": "4", "a": "3"})
	if len(a["sections"].(map[string]any)) != len(b["sections"].(map[string]any)) {
		t.Error("diff depends on map order")
	}
}

func TestDiffEmpty(t *testing.T) {
	d := Diff(nil, nil)
	if c := d["counts"].(map[string]int); c["added"]+c["removed"]+c["changed"] != 0 {
		t.Errorf("empty diff should be empty: %v", c)
	}
}
