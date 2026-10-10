// Package linebudget holds the server's half of the modularity guard.
// There is no production code here — the test is the package, mirroring
// client/test/line_budget_test.dart.
package linebudget

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// AGENTS.md keeps implementation files under ~350 lines repo-wide; the
// client side has had its guard since the 2026-10-10 sweep, the server
// tree had none. The server's largest file today is 343 lines
// (internal/api/prds.go), so the budget starts at 400 — green today, and
// the ratchet: files only ever come back under it, never drift past it
// unnoticed. A file that crosses the budget fails here until it is split
// or listed below with a dated modularity issue.
func TestServerFilesStayWithinTheLineBudget(t *testing.T) {
	// The module root from this file's own location, so the scan works no
	// matter which directory go test runs from.
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")

	const budget = 400
	// Files allowed over the budget, each with an open dated modularity
	// issue in docs/issues/. None today.
	overBudgetAllowed := map[string]bool{}

	type offender struct {
		path  string
		lines int
	}
	var over []offender
	files := 0
	for _, dir := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			files++
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			lines := strings.Count(string(data), "\n")
			if lines > budget && !overBudgetAllowed[rel] {
				over = append(over, offender{rel, lines})
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	// A scan that finds almost nothing would pass vacuously.
	if files < 100 {
		t.Fatalf("scanned only %d non-test Go files; the scan is broken", files)
	}
	for _, o := range over {
		t.Errorf("%s: %d lines, over the %d-line budget — split it or list it with a dated modularity issue", o.path, o.lines, budget)
	}
}
