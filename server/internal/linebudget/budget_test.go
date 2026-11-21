// Package linebudget holds the server's half of the code-budget guard:
// one responsibility per file, files and functions kept short. There is
// no production code here — the tests are the package, mirroring
// client/test/line_budget_test.dart. Both guards read their thresholds
// from config/code-budget.json, so the Go and Dart limits cannot drift.
package linebudget

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type codeBudget struct {
	FileLines     int `json:"fileLines"`
	FunctionLines int `json:"functionLines"`
}

// serverRoot is the Go module root, from this file's own location, so the
// scan works no matter which directory go test runs from.
func serverRoot() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

func loadBudget(t *testing.T) codeBudget {
	t.Helper()
	path := filepath.Join(serverRoot(), "..", "config", "code-budget.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var b codeBudget
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if b.FileLines <= 0 || b.FunctionLines <= 0 {
		t.Fatalf("%s: fileLines and functionLines must both be positive, got %+v", path, b)
	}
	return b
}

// walkSource calls fn with the root-relative path of every non-test Go
// file under cmd/ and internal/, and fails if the scan finds almost
// nothing (a broken scan would pass vacuously).
func walkSource(t *testing.T, fn func(rel, abs string)) {
	t.Helper()
	root := serverRoot()
	files := 0
	for _, dir := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files++
			fn(filepath.ToSlash(rel), path)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	if files < 100 {
		t.Fatalf("scanned only %d non-test Go files; the scan is broken", files)
	}
}
