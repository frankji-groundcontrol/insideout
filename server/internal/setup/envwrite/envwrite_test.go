package envwrite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/setup/catalog"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func requirePerm0600(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("%s mode = %v, want 0600", path, got)
	}
}

func valueOf(t *testing.T, envPath, name string) (string, bool) {
	t.Helper()
	s := catalog.Var{Name: name, Required: true}
	states := catalog.Evaluate([]byte(read(t, envPath)), []catalog.Var{s})
	if len(states) != 1 {
		t.Fatalf("Evaluate returned %d states", len(states))
	}
	return states[0].Value, states[0].Status == catalog.StatusSet
}

func TestEnsureFromExampleCreatesMissingEnvWith0600(t *testing.T) {
	dir := t.TempDir()
	example := filepath.Join(dir, ".env.example")
	env := filepath.Join(dir, ".env")
	write(t, example, "DATABASE_URL=\n#INSIDEOUT_ACCESS_TTL=15m\n")
	if err := EnsureFromExample(env, example); err != nil {
		t.Fatalf("EnsureFromExample: %v", err)
	}
	if got, want := read(t, env), "DATABASE_URL=\n#INSIDEOUT_ACCESS_TTL=15m\n"; got != want {
		t.Errorf("env = %q, want a byte copy of the example", got)
	}
	requirePerm0600(t, env)
}

func TestEnsureFromExampleKeepsExistingEnv(t *testing.T) {
	dir := t.TempDir()
	example := filepath.Join(dir, ".env.example")
	env := filepath.Join(dir, ".env")
	write(t, example, "A=1\n")
	write(t, env, "KEEP=mine\n")
	if err := EnsureFromExample(env, example); err != nil {
		t.Fatalf("EnsureFromExample: %v", err)
	}
	if got := read(t, env); got != "KEEP=mine\n" {
		t.Errorf("existing env was overwritten: %q", got)
	}
}

func TestEnsureFromExampleErrorsWithoutExample(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	if err := EnsureFromExample(env, filepath.Join(dir, ".env.example")); err == nil {
		t.Fatal("missing example must be an error, got nil")
	}
	if _, err := os.Stat(env); !os.IsNotExist(err) {
		t.Errorf("no env file must be created when the example is missing, stat err = %v", err)
	}
}

func TestSetVarReplacesInPlaceAndPreservesTheRest(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	write(t, env, "A=1\n# a comment\n#B=\nC=3\n")
	if err := SetVar(env, "B", "two"); err != nil {
		t.Fatalf("SetVar: %v", err)
	}
	want := "A=1\n# a comment\nB=two\nC=3\n"
	if got := read(t, env); got != want {
		t.Errorf("file = %q, want %q (only the target line may change)", got, want)
	}
	requirePerm0600(t, env)
	if value, set := valueOf(t, env, "B"); !set || value != "two" {
		t.Errorf("B = %q set=%v, want two/true", value, set)
	}
}

func TestSetVarTargetsTheLastOccurrence(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	write(t, env, "K=first\n# note\nK=second\n")
	if err := SetVar(env, "K", "edited"); err != nil {
		t.Fatalf("SetVar: %v", err)
	}
	want := "K=first\n# note\nK=edited\n"
	if got := read(t, env); got != want {
		t.Errorf("file = %q, want %q (last occurrence wins)", got, want)
	}
	if err := ClearVar(env, "K"); err != nil {
		t.Fatalf("ClearVar: %v", err)
	}
	if got, want := read(t, env), "K=first\n# note\n#K=\n"; got != want {
		t.Errorf("after ClearVar file = %q, want %q (clear also targets the last)", got, want)
	}
}

func TestSetVarPrefersALiveLineOverLaterCommentedOnes(t *testing.T) {
	// python _last_match: the last UNCOMMENTED assignment is the effective
	// one even when commented lines follow it.
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	write(t, env, "#B=1\nB=two\n#B=3\n")
	if err := SetVar(env, "B", "new"); err != nil {
		t.Fatalf("SetVar: %v", err)
	}
	if got, want := read(t, env), "#B=1\nB=new\n#B=3\n"; got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
}

func TestSetVarUncommentsACommentedOnlyKeyInPlace(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	write(t, env, "#ONLY=old\n")
	if err := SetVar(env, "ONLY", "new"); err != nil {
		t.Fatalf("SetVar: %v", err)
	}
	if got, want := read(t, env), "ONLY=new\n"; got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
}

func TestSetVarAppendsWhenAbsentAndUnfusesTheFinalNewline(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	write(t, env, "A=1")
	if err := SetVar(env, "NEW", "v"); err != nil {
		t.Fatalf("SetVar: %v", err)
	}
	if got, want := read(t, env), "A=1\nNEW=v\n"; got != want {
		t.Errorf("file = %q, want %q (a missing final newline is unfused before appending)", got, want)
	}
}

func TestSetVarCreatesAMissingFile(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	if err := SetVar(env, "K", "v"); err != nil {
		t.Fatalf("SetVar: %v", err)
	}
	if got, want := read(t, env), "K=v\n"; got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
	requirePerm0600(t, env)
}

func TestSetVarHandlesADSN(t *testing.T) {
	// A DSN is full of / : @ & — the reason the writer rebuilds the line
	// instead of running a substitution.
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	write(t, env, "DATABASE_URL=\n")
	dsn := "postgres://insideout_app:p@ss/w:rd&x@10.0.0.1:5432/insideout?sslmode=require"
	if err := SetVar(env, "DATABASE_URL", dsn); err != nil {
		t.Fatalf("SetVar: %v", err)
	}
	if value, set := valueOf(t, env, "DATABASE_URL"); !set || value != dsn {
		t.Errorf("DSN round-trip = %q set=%v", value, set)
	}
}

func TestSetVarLeavesNoTempFileBehind(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	write(t, env, "K=1\n")
	if err := SetVar(env, "K", "2"); err != nil {
		t.Fatalf("SetVar: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestClearVarCommentsOutNotDeletes(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	write(t, env, "A=1\nGITHUB_TOKEN=ghp_x\nC=3\n")
	if err := ClearVar(env, "GITHUB_TOKEN"); err != nil {
		t.Fatalf("ClearVar: %v", err)
	}
	if got, want := read(t, env), "A=1\n#GITHUB_TOKEN=\nC=3\n"; got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
	if _, set := valueOf(t, env, "GITHUB_TOKEN"); set {
		t.Error("cleared key must no longer parse as set")
	}
	// Clearing an already-commented line rewrites the last commented match.
	if err := ClearVar(env, "GITHUB_TOKEN"); err != nil {
		t.Fatalf("ClearVar again: %v", err)
	}
	if got, want := read(t, env), "A=1\n#GITHUB_TOKEN=\nC=3\n"; got != want {
		t.Errorf("clearing a cleared key changed the file: %q", got)
	}
}

func TestClearVarIsNoOpWithoutTheKeyOrFile(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	if err := ClearVar(env, "K"); err != nil {
		t.Errorf("ClearVar on a missing file: %v", err)
	}
	if _, err := os.Stat(env); !os.IsNotExist(err) {
		t.Errorf("ClearVar must not create a file, stat err = %v", err)
	}
	write(t, env, "A=1\n")
	if err := ClearVar(env, "ABSENT"); err != nil {
		t.Fatalf("ClearVar: %v", err)
	}
	if got := read(t, env); got != "A=1\n" {
		t.Errorf("ClearVar of an absent key changed the file: %q", got)
	}
}

// TestQuoteRoundTripsAwkwardValues mirrors scripts/test_env_writes.py's
// t_quote_round_trips_awkward_values value-for-value: every rendered form must
// read back through the catalog's parser as the original value.
func TestQuoteRoundTripsAwkwardValues(t *testing.T) {
	for _, raw := range []string{
		"two words", "with'single", "with\"double", "with$dollar",
		"back`tick", "trailing ", "a=b c", "plain", "", "p@ss/w:rd&x",
	} {
		t.Run(raw, func(t *testing.T) {
			dir := t.TempDir()
			env := filepath.Join(dir, ".env")
			write(t, env, "K="+quote(raw)+"\n")
			got, _ := valueOf(t, env, "K")
			if got != raw {
				t.Errorf("quote round-trip: got %q, want %q", got, raw)
			}
		})
	}
}

func TestQuoteRenderedForms(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", "plain"},
		{"", ""},
		{"two words", "'two words'"},
		{"with$dollar", "'with$dollar'"},
		{"with'single", `"with'single"`},
		{`he said "hi"`, `'he said "hi"'`},
		{"it's $5 `x` \\y", "\"it's \\$5 \\`x\\` \\\\y\""},
	}
	for _, tc := range cases {
		if got := quote(tc.in); got != tc.want {
			t.Errorf("quote(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestSetVarSpacePasswordRoundTrips pins the reason quoting exists: a space is
// legal in a Postgres password, and a bare `KEY=two words` would make bash
// treat the second word as a command.
func TestSetVarSpacePasswordRoundTrips(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	write(t, env, "DATABASE_URL=\n")
	dsn := "postgres://insideout_app:two word pass@10.0.0.1:5432/insideout"
	if err := SetVar(env, "DATABASE_URL", dsn); err != nil {
		t.Fatalf("SetVar: %v", err)
	}
	if got := read(t, env); !strings.Contains(got, "'"+dsn+"'") {
		t.Errorf("a value with a space must be quoted, line = %q", got)
	}
	if value, set := valueOf(t, env, "DATABASE_URL"); !set || value != dsn {
		t.Errorf("space password round-trip = %q set=%v", value, set)
	}
}
