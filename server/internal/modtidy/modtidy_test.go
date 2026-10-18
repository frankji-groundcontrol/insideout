// Package modtidy holds the go.mod tidiness guard. There is no production
// code here: the test is the package, like internal/linebudget.
package modtidy

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// rv4-PM-10: go.mod marked the directly imported charmbracelet/huh as
// "// indirect" and nothing noticed, because nothing ever ran `go mod tidy`.
// `go mod tidy -diff` prints what tidy would change, in go.mod and go.sum
// alike, without touching the tree, so its output must be empty.
func TestGoModIsTidy(t *testing.T) {
	// The module root from this file's own location, so the check works no
	// matter which directory go test runs from.
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")

	// Offline first: the module cache is warm wherever the module builds, and
	// the guard should not need a network. Fall back to the ambient settings
	// when a test-only module is missing from the cache.
	out, errOut, err := tidyDiff(root, "GOPROXY=off")
	if out == "" && err != nil {
		out, errOut, err = tidyDiff(root)
	}
	// Exit status 1 with a diff on stdout is the expected "not tidy" answer;
	// an error with nothing on stdout means tidy itself could not run.
	if out == "" && err != nil {
		t.Fatalf("go mod tidy -diff could not run: %v\n%s", err, errOut)
	}

	if out != "" {
		t.Errorf("go.mod or go.sum is not tidy; `cd server && go mod tidy` would change it:\n%s", out)
	}
}

// tidyDiff runs `go mod tidy -diff` in root with extra environment entries.
func tidyDiff(root string, env ...string) (stdout, stderr string, err error) {
	cmd := exec.Command("go", "mod", "tidy", "-diff")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), env...)
	var o, e bytes.Buffer
	cmd.Stdout, cmd.Stderr = &o, &e
	err = cmd.Run()
	return o.String(), e.String(), err
}
