// Package modtidy holds the go.mod tidiness guard. There is no production
// code here: the test is the package, like internal/linebudget.
package modtidy

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// rv4-PM-10: go.mod marked the directly imported charmbracelet/huh as
// "// indirect" and nothing noticed, because nothing ever ran `go mod tidy`.
// `go mod tidy -diff` prints what tidy would change without touching the
// tree, so the go.mod half of that diff must be empty.
//
// The go.sum half is logged, not asserted. Tidy also wants hashes for the
// test-only dependencies of huh, and regenerating go.sum is tracked apart
// from this guard (see the 2026-10-18 rv4 w1-misc-correctness changelog
// fragment). Tighten this to the whole diff once go.sum has been tidied.
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

	if mod := section(out, "go.mod"); mod != "" {
		t.Errorf("go.mod is not tidy; `cd server && go mod tidy` would change it:\n%s", mod)
	}
	if sum := section(out, "go.sum"); sum != "" {
		t.Logf("go.sum also differs from `go mod tidy` (not asserted by this guard)")
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

// section returns the part of a `go mod tidy -diff` output that concerns one
// file. The output is one "diff current/<file> tidy/<file>" block per changed
// file; a file that is already tidy has no block.
func section(out, file string) string {
	var keep []string
	in := false
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "diff ") {
			in = strings.HasSuffix(line, " tidy/"+file)
		}
		if in {
			keep = append(keep, line)
		}
	}
	return strings.TrimSpace(strings.Join(keep, "\n"))
}
