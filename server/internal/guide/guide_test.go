package guide

import (
	"strings"
	"testing"
)

func TestGenerateLeavesAndBranches(t *testing.T) {
	out := Generate("Parity Tool", []Node{
		{ID: "11111111-1111-1111-1111-111111111111", Title: "交付 MVP", Leaf: false},
		{ID: "22222222-2222-2222-2222-222222222222", Title: `Fix "quoting: bug"`, Leaf: true},
	})
	if !strings.HasPrefix(out, "# InsideOut matching guide") || !strings.Contains(out, "version: 1") {
		t.Fatalf("missing header/version:\n%s", out)
	}
	if !strings.Contains(out, "11111111-1111-1111-1111-111111111111 is a branch node") {
		t.Errorf("branch node not commented out:\n%s", out)
	}
	if !strings.Contains(out, `22222222-2222-2222-2222-222222222222:`) {
		t.Errorf("leaf not emitted as key:\n%s", out)
	}
	if !strings.Contains(out, `title: "Fix \"quoting: bug\""`) {
		t.Errorf("title not escaped:\n%s", out)
	}
	for _, want := range []string{"branches: []", "labels:   []", "paths:    []"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	if !strings.Contains(out, `of "Parity Tool"`) || strings.Contains(out, `""Parity`) {
		t.Errorf("project title quoting wrong: %s", out)
	}
}

func TestGenerateEmpty(t *testing.T) {
	out := Generate("Empty", nil)
	if !strings.Contains(out, "{} # no roadmap nodes yet") {
		t.Fatalf("empty roadmap not handled:\n%s", out)
	}
}
