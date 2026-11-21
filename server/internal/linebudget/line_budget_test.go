package linebudget

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// overBudgetFiles lists the non-test Go files allowed over fileLines, each
// with its current line count and an open dated modularity issue in
// docs/issues/. A listed file may not grow, and an entry for a file that
// is gone or back under the budget fails too, so the list only shrinks.
// None today.
var overBudgetFiles = map[string]int{}

// A file past the shared threshold is usually doing more than one thing.
// It fails here until it is split by responsibility or listed above.
func TestServerFilesStayWithinTheLineBudget(t *testing.T) {
	budget := loadBudget(t).FileLines
	var paste []string
	seen := map[string]int{}
	walkSource(t, func(rel, abs string) {
		data, err := os.ReadFile(abs)
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		lines := strings.Count(string(data), "\n")
		seen[rel] = lines
		allowed, listed := overBudgetFiles[rel]
		switch {
		case lines > budget && !listed:
			t.Errorf("%s: %d lines, over the %d-line budget — split it by responsibility or list it with a dated modularity issue", rel, lines, budget)
			paste = append(paste, fmt.Sprintf("%q: %d,", rel, lines))
		case listed && lines > allowed:
			t.Errorf("%s: %d lines, grew past its listed %d — split it instead", rel, lines, allowed)
			paste = append(paste, fmt.Sprintf("%q: %d,", rel, lines))
		}
	})
	for rel := range overBudgetFiles {
		if lines, ok := seen[rel]; !ok || lines <= budget {
			t.Errorf("%s: listed in overBudgetFiles but is gone or back within %d lines — delete the entry and close its issue", rel, budget)
		}
	}
	if len(paste) > 0 {
		t.Logf("overBudgetFiles entries:\n%s", strings.Join(paste, "\n"))
	}
}
