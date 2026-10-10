package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// This package and scripts/env_catalog.py both read .env.example, on purpose
// (docs/issues/2026-10-07-env-catalog-go-python-duplication.md). These tests
// are what keeps them one schema: the same skeleton and the same live .env go
// through both readers and the per-variable results are diffed. Declarations
// are compared in file order (name, required, example, default in effect, help
// text, secret); live state as value and status.
//
// Not compared, by design: Display (Go prints "set (N chars)", the TUI prints
// bullets; both are gated on the same secret decision, which is compared) and
// build()'s attention sort (Go lists in file order and has no sort to compare).
// A name declared twice is compared line by line only: ParseSkeleton lists
// every line and leaves the merge to callers, while build() merges.
//
// Scope: the grammar .env.example documents, as UTF-8 with space and tab blanks
// and LF or CRLF line ends. Outside it the readers do differ, and nothing here
// pretends otherwise: Python's splitlines(), strip() and \s also take lone CR,
// VT, FF, FS to US, NEL, U+2028/9 as line breaks and NBSP, U+3000 and the other
// Unicode spaces as blanks, where this package splits on LF and its \s is ASCII.
// No rule in the repo says which is right, so neither side was changed for it.
//
// The tests need python3 and FAIL without it: CI treats a skip as a failure,
// so a quiet skip would turn this guard off exactly where it matters.

type pyDecl struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Example  string `json:"example"`
	Comment  string `json:"comment"`
}

type pyVar struct {
	Name     string  `json:"name"`
	Required bool    `json:"required"`
	Value    *string `json:"value"`
	Status   string  `json:"status"`
	Secret   bool    `json:"secret"`
}

type pyReading struct {
	Skeleton []pyDecl `json:"skeleton"`
	Vars     []pyVar  `json:"vars"`
}

// pyTool is the interpreter plus the reader it runs, resolved once per test so
// a missing python3 is one failure, not one per subtest.
type pyTool struct{ python, scripts, driver string }

func findPython(t *testing.T) pyTool {
	t.Helper()
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Fatalf("python3 is not on PATH (%v): the catalog parity test needs it and must not skip; install python3", err)
	}
	scripts, err := filepath.Abs("../../../../scripts")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(scripts, "env_catalog.py")); err != nil {
		t.Fatalf("the Python reader is missing: %v", err)
	}
	driver, err := filepath.Abs("testdata/pyreader.py")
	if err != nil {
		t.Fatal(err)
	}
	return pyTool{py, scripts, driver}
}

// read runs env_catalog.py over a scratch repo root holding example and env.
// The root is scratch so the real .env is unreachable, and the child gets an
// empty environment so a stray INSIDEOUT_ENV_FILE cannot redirect the read.
func (p pyTool) read(t *testing.T, example, env string) pyReading {
	t.Helper()
	root := t.TempDir()
	for name, body := range map[string]string{".env.example": example, ".env": env} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	// -I: no cwd, PYTHON* or user site on sys.path; -B: no .pyc written into
	// scripts/; -X utf8: the skeleton is bilingual, so do not trust the locale.
	cmd := exec.CommandContext(ctx, p.python, "-I", "-B", "-X", "utf8", p.driver, p.scripts, root)
	cmd.Env = []string{}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("python reader failed: %v\n%s", err, stderr.String())
	}
	var r pyReading
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		t.Fatalf("python reader printed unparseable JSON: %v", err)
	}
	return r
}

// diffReadings returns one line per disagreement between this package and the
// Python reader on (example, env). Lines carry names and statuses, never a
// live value: only skeleton text (public) and value lengths are quoted.
func diffReadings(example, env string, py pyReading) []string {
	var diffs []string
	add := func(format string, a ...any) { diffs = append(diffs, fmt.Sprintf(format, a...)) }

	vars, err := ParseSkeleton([]byte(example))
	if err != nil {
		return []string{fmt.Sprintf("ParseSkeleton: %v", err)}
	}
	if len(vars) != len(py.Skeleton) {
		// Everything after the first missing line would be misaligned noise.
		var g, p []string
		for _, v := range vars {
			g = append(g, v.Name)
		}
		for _, d := range py.Skeleton {
			p = append(p, d.Name)
		}
		return []string{fmt.Sprintf("declarations differ: go %d %v, python %d %v", len(g), g, len(p), p)}
	}
	pyByName := map[string]pyVar{}
	for _, p := range py.Vars {
		pyByName[p.Name] = p
	}
	declared := map[string]int{}
	for i, v := range vars {
		p := py.Skeleton[i]
		declared[v.Name]++
		if v.Name != p.Name {
			add("declaration %d: name go=%q python=%q", i, v.Name, p.Name)
			continue
		}
		if v.Required != p.Required {
			add("%s: required go=%v python=%v", v.Name, v.Required, p.Required)
		}
		if v.Example != p.Example {
			add("%s: example go=%q python=%q", v.Name, v.Example, p.Example)
		}
		// Python has no field for the default in effect: it is the example of an
		// optional variable, and nothing for a required one.
		wantDefault := p.Example
		if p.Required {
			wantDefault = ""
		}
		if v.Default != wantDefault {
			add("%s: default in effect go=%q python=%q", v.Name, v.Default, wantDefault)
		}
		if v.Comment != p.Comment {
			add("%s: help text go=%q python=%q", v.Name, v.Comment, p.Comment)
		}
		if s := pyByName[v.Name].Secret; v.Secret != s || Secret(v.Name) != s {
			add("%s: secret go=%v/%v python=%v", v.Name, v.Secret, Secret(v.Name), s)
		}
	}

	// build() also lists names only the .env mentions, as optional variables
	// with no example. Evaluate reports what it is given, so give it the same.
	all := append([]Var(nil), vars...)
	var extra []string
	for name := range parseEnv(env) {
		if declared[name] == 0 {
			extra = append(extra, name)
		}
	}
	sort.Strings(extra)
	for _, name := range extra {
		all = append(all, Var{Name: name, Secret: Secret(name)})
	}
	seen := map[string]bool{}
	for _, s := range Evaluate([]byte(env), all) {
		name := s.Var.Name
		seen[name] = true
		p, ok := pyByName[name]
		if !ok {
			add("%s: go lists it, python does not", name)
			continue
		}
		if declared[name] == 0 && Secret(name) != p.Secret { // declared names were checked above
			add("%s: secret go=%v python=%v", name, Secret(name), p.Secret)
		}
		if declared[name] > 1 {
			continue // the one place the readers differ by contract: merging
		}
		if s.Var.Required != p.Required {
			add("%s: required go=%v python=%v", name, s.Var.Required, p.Required)
		}
		if string(s.Status) != p.Status {
			add("%s: status go=%s python=%s", name, s.Status, p.Status)
		}
		pv := ""
		if p.Value != nil {
			pv = *p.Value
		}
		if s.Value != pv {
			add("%s: live value differs (go %d bytes, python %d bytes)", name, len(s.Value), len(pv))
		}
	}
	for _, p := range py.Vars {
		if !seen[p.Name] {
			add("%s: python lists it, go does not", p.Name)
		}
	}
	return diffs
}
