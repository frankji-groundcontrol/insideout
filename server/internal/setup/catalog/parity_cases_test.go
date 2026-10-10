package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The cases half of the parity tests: the skeletons and live .env texts fed to
// both readers (harness and diff are in parity_test.go).

// shapes are the ways a real file differs from the one in the repo without
// changing what it says: Windows line ends, a missing final newline, and
// indented lines with trailing blanks.
var shapes = []struct {
	name  string
	apply func(string) string
}{
	{"lf", func(s string) string { return s }},
	{"crlf", func(s string) string { return strings.ReplaceAll(s, "\n", "\r\n") }},
	{"no-final-newline", func(s string) string { return strings.TrimRight(s, "\n") }},
	{"padded", func(s string) string {
		lines := strings.Split(s, "\n")
		for i, l := range lines {
			lines[i] = " \t" + l + "\t "
		}
		return strings.Join(lines, "\n")
	}},
}

type scenario struct{ name, env string }

// derivedScenarios builds live .env texts from the skeleton itself, so they
// track it as it grows: nothing set, every variable on its example, every
// variable blanked, every variable chosen, and every variable a leftover
// placeholder.
func derivedScenarios(vars []Var) []scenario {
	done := map[string]bool{}
	var onExample, blank, chosen, left strings.Builder
	for i, v := range vars {
		if done[v.Name] {
			continue
		}
		done[v.Name] = true
		fmt.Fprintf(&onExample, "%s=%s\n", v.Name, v.Example)
		fmt.Fprintf(&blank, "%s=\n", v.Name)
		fmt.Fprintf(&chosen, "%s=chosen-%d\n", v.Name, i)
		marker := []string{"CHANGE_ME", "x-your-remote-host-y"}[i%2]
		fmt.Fprintf(&left, "%s=%s-%d\n", v.Name, marker, i)
	}
	return []scenario{
		{"nothing set", ""},
		{"every variable on its example", onExample.String()},
		{"every variable blank", blank.String()},
		{"every variable chosen", chosen.String()},
		{"every variable a placeholder", left.String()},
	}
}

// checkParity runs every scenario through every file shape. A skeleton-level
// disagreement shows up again under each of them, so each distinct line is
// printed once: one regression should read as one line, not hundreds.
func checkParity(t *testing.T, example string, scenarios []scenario) {
	t.Helper()
	py := findPython(t)
	printed := map[string]bool{}
	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			for _, s := range shapes {
				t.Run(s.name, func(t *testing.T) {
					ex, en := s.apply(example), s.apply(sc.env)
					for _, d := range diffReadings(ex, en, py.read(t, ex, en)) {
						if !printed[d] {
							printed[d] = true
							t.Error(d)
						}
					}
				})
			}
		})
	}
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestParityFixture(t *testing.T) {
	example := readFixture(t, "skeleton.example")
	vars, err := ParseSkeleton([]byte(example))
	if err != nil {
		t.Fatal(err)
	}
	checkParity(t, example, append(
		[]scenario{{"live.dotenv", readFixture(t, "live.dotenv")}}, derivedScenarios(vars)...))
}

// The repo's own skeleton, so a new variable or a new spelling in it is
// checked the day it is added.
func TestParityRepoSkeleton(t *testing.T) {
	b, err := os.ReadFile("../../../../.env.example")
	if err != nil {
		t.Fatal(err)
	}
	vars, err := ParseSkeleton(b)
	if err != nil {
		t.Fatal(err)
	}
	// ParseSkeleton lists every line, so a repeated name is two rows to Go's
	// callers but one to build(); diffReadings leaves repeats out of the state
	// comparison. The real skeleton has none: keep it so, and the comparison whole.
	names := map[string]bool{}
	for _, v := range vars {
		if names[v.Name] {
			t.Errorf("%s is declared twice in .env.example", v.Name)
		}
		names[v.Name] = true
	}
	checkParity(t, string(b), derivedScenarios(vars))
}

// The fixture is only a guard while it still exercises every rule; trimming it
// down must fail here, not pass quietly.
func TestParityFixtureCoversEveryRule(t *testing.T) {
	vars, err := ParseSkeleton([]byte(readFixture(t, "skeleton.example")))
	if err != nil {
		t.Fatal(err)
	}
	var required, optional, secret, plain bool
	names := map[string]int{}
	for _, v := range vars {
		names[v.Name]++
		required, optional = required || v.Required, optional || !v.Required
		secret, plain = secret || v.Secret, plain || !v.Secret
	}
	if !required || !optional || !secret || !plain {
		t.Errorf("fixture needs required, optional, secret and plain names, got %v/%v/%v/%v",
			required, optional, secret, plain)
	}
	if names["DUP_NAME"] != 2 {
		t.Errorf("fixture needs a name declared twice, DUP_NAME is declared %d time(s)", names["DUP_NAME"])
	}
	live := readFixture(t, "live.dotenv")
	if _, ok := parseEnv(live)["ENV_ONLY"]; !ok || names["ENV_ONLY"] != 0 {
		t.Error("fixture needs a name the live .env sets and the skeleton never declares: ENV_ONLY")
	}
	statuses := map[Status]bool{}
	for _, s := range Evaluate([]byte(live), vars) {
		statuses[s.Status] = true
	}
	for _, want := range []Status{StatusSet, StatusDefault, StatusPlaceholder, StatusMissing, StatusUnset} {
		if !statuses[want] {
			t.Errorf("live.dotenv never produces status %q", want)
		}
	}
}
