package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Local-mode helpers (slice 2). The pure parts are unit-tested here;
// the composed path is exercised live against a real PostgreSQL
// cluster (practice record), so these tests never need Docker or a
// database.

func TestBundledDSN(t *testing.T) {
	for _, tt := range []struct {
		name       string
		port, pass string
		want       string
	}{
		{"defaults", "", "", "postgres://insideout_app:insideout_app_dev_password@127.0.0.1:5442/insideout?sslmode=disable"},
		{"from env values", "5443", "s3cret", "postgres://insideout_app:s3cret@127.0.0.1:5443/insideout?sslmode=disable"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := bundledDSN(tt.port, tt.pass); got != tt.want {
				t.Fatalf("bundledDSN() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestJWTNeedsRotation(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want bool
	}{
		{"", true},
		{"change_me_to_a_long_random_string", true},
		{"short", true},
		{strings.Repeat("x", 31), true},
		{strings.Repeat("x", 32), false},
	} {
		if got := jwtNeedsRotation(tt.in); got != tt.want {
			t.Errorf("jwtNeedsRotation(%d chars) = %v, want %v", len(tt.in), got, tt.want)
		}
	}
}

func TestChildEnvInjectsWrittenValues(t *testing.T) {
	vals := map[string]string{
		"DATABASE_URL":         "postgres://scratch",
		"DATABASE_OWNER_URL":   "postgres://owner-scratch",
		"INSIDEOUT_JWT_SECRET": strings.Repeat("k", 64),
	}
	env := childEnv([]string{
		"PATH=/bin",
		"DATABASE_URL=postgres://old",
		"DATABASE_OWNER_URL=postgres://old-owner",
		"INSIDEOUT_COOKIE_SECURE=1",
	}, vals)
	seen := map[string]string{}
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		seen[k] = v
	}
	if seen["DATABASE_URL"] != "postgres://scratch" {
		t.Errorf("DATABASE_URL = %q, want the setup-written value", seen["DATABASE_URL"])
	}
	if seen["DATABASE_OWNER_URL"] != "postgres://owner-scratch" {
		t.Errorf("DATABASE_OWNER_URL = %q, want the setup-written value (the server migrates at startup as the owner)", seen["DATABASE_OWNER_URL"])
	}
	if seen["INSIDEOUT_JWT_SECRET"] != vals["INSIDEOUT_JWT_SECRET"] {
		t.Errorf("INSIDEOUT_JWT_SECRET not injected")
	}
	if seen["INSIDEOUT_COOKIE_SECURE"] != "0" {
		t.Errorf("INSIDEOUT_COOKIE_SECURE = %q, want 0 for plain-http local serving", seen["INSIDEOUT_COOKIE_SECURE"])
	}
	if seen["PATH"] != "/bin" {
		t.Errorf("unrelated environ entries must pass through, PATH = %q", seen["PATH"])
	}
}

func TestReadEnvValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("# comment\nA=1\nB = \"two words\"\nexport C=3\n\nB=override\n"), 0600)
	vals := readEnvValues(path)
	if vals["A"] != "1" || vals["B"] != "override" || vals["C"] != "3" {
		t.Fatalf("readEnvValues = %#v", vals)
	}
	if readEnvValues(filepath.Join(dir, "missing")) == nil {
		t.Fatal("missing file should yield an empty, usable map")
	}
}

func TestFindRepoRootNeedsCheckoutMarkers(t *testing.T) {
	t.Setenv("INSIDEOUT_SETUP_REPO_ROOT", "")
	// A temp dir has no markers; the walk must exhaust without panic.
	if _, err := findRepoRootFrom(t.TempDir()); err == nil {
		t.Fatal("expected an error outside a checkout")
	}
}

func findRepoRootFrom(start string) (string, error) {
	dir := start
	for {
		if repoMarker(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func TestPrintSummaryGateMasksSecretsAndNamesOutstanding(t *testing.T) {
	dir := t.TempDir()
	example := filepath.Join(dir, ".env.example")
	env := filepath.Join(dir, ".env")
	os.WriteFile(example, []byte("DATABASE_URL=change_me_db\nINSIDEOUT_JWT_SECRET=change_me_to_a_long_random_string\n#INSIDEOUT_ADDR=:8080\n"), 0644)
	os.WriteFile(env, []byte("DATABASE_URL=postgres://user:supersecret@h/db\nINSIDEOUT_JWT_SECRET="+strings.Repeat("k", 64)+"\n"), 0600)

	out := captureStderr(t, func() {
		if err := printSummaryGate(env, example); err != nil {
			t.Errorf("printSummaryGate: %v", err)
		}
	})
	if strings.Contains(out, "supersecret") {
		t.Fatalf("summary leaked a secret value: %q", out)
	}
	if !strings.Contains(out, "Required: all satisfied") {
		t.Fatalf("summary should report required satisfied, got: %q", out)
	}

	// And the outstanding lane: an unset required var is named, value-free.
	os.WriteFile(env, []byte("INSIDEOUT_JWT_SECRET="+strings.Repeat("k", 64)+"\n"), 0600)
	out = captureStderr(t, func() {
		if err := printSummaryGate(env, example); err != nil {
			t.Errorf("printSummaryGate: %v", err)
		}
	})
	if !strings.Contains(out, "DATABASE_URL") {
		t.Fatalf("summary should name the outstanding required var, got: %q", out)
	}
}

// captureStderr swaps os.Stderr for a pipe around fn and returns what
// was written.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = old }()
	fn()
	w.Close()
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

func TestLocalNoStartWritesEnvEndToEnd(t *testing.T) {
	// A fake checkout: markers only — no-start never reaches compose
	// or the database, so the write path is fully testable.
	repo := t.TempDir()
	for _, name := range []string{"docker-compose.yml", ".env.example"} {
		if err := os.WriteFile(filepath.Join(repo, name), []byte("# stub\nDATABASE_URL=change_me_db\nINSIDEOUT_JWT_SECRET=change_me_to_a_long_random_string\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(repo, "server"), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(repo, "server", "go.mod"), []byte("module stub\n"), 0644)

	scratchEnv := filepath.Join(t.TempDir(), "scratch.env")
	t.Setenv("INSIDEOUT_SETUP_REPO_ROOT", repo)
	t.Setenv("INSIDEOUT_ENV_FILE", scratchEnv)
	t.Setenv("HOME", t.TempDir())
	// Linux resolves the config dir from XDG_CONFIG_HOME before HOME (and CI
	// sets it), so pinning HOME alone leaves every test on one shared file.
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("INSIDEOUT_API", "")
	t.Setenv("INSIDEOUT_TOKEN", "")

	// The DSN arrives on stdin, the agent path (test stdin is a pipe).
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("postgres://setup_test:pw@127.0.0.1:5599/insideout?sslmode=disable\n")
	w.Close()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	handled, code := runSetupCommand([]string{"--local", "--postgres", "own", "--no-start"})
	if !handled || code != 0 {
		t.Fatalf("setup --local --postgres own --no-start = (%v, %d), want (true, 0)", handled, code)
	}

	vals := readEnvValues(scratchEnv)
	if vals["DATABASE_URL"] != "postgres://setup_test:pw@127.0.0.1:5599/insideout?sslmode=disable" {
		t.Errorf("DATABASE_URL not stored: %q", vals["DATABASE_URL"])
	}
	if len(vals["INSIDEOUT_JWT_SECRET"]) < 32 || vals["INSIDEOUT_JWT_SECRET"] == "change_me_to_a_long_random_string" {
		t.Errorf("INSIDEOUT_JWT_SECRET not rotated: %d chars", len(vals["INSIDEOUT_JWT_SECRET"]))
	}
}

func TestLocalBundledFullRunFailsCleanlyWithoutDocker(t *testing.T) {
	// An empty PATH hides docker whether or not the host has it, so this
	// branch runs everywhere instead of skipping on a machine (or CI
	// runner) that ships docker.
	t.Setenv("PATH", t.TempDir())
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "docker-compose.yml"), []byte("services: {}\n"), 0644)
	os.WriteFile(filepath.Join(repo, ".env.example"), []byte("DATABASE_URL=change_me_db\nINSIDEOUT_JWT_SECRET=change_me_to_a_long_random_string\n"), 0644)
	os.MkdirAll(filepath.Join(repo, "server"), 0755)
	os.WriteFile(filepath.Join(repo, "server", "go.mod"), []byte("module stub\n"), 0644)

	t.Setenv("INSIDEOUT_SETUP_REPO_ROOT", repo)
	t.Setenv("INSIDEOUT_ENV_FILE", filepath.Join(t.TempDir(), "scratch.env"))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")

	handled, code := runSetupCommand([]string{"--local", "--postgres", "bundled"})
	if !handled || code != 1 {
		t.Fatalf("setup --local --postgres bundled (no docker) = (%v, %d), want (true, 1)", handled, code)
	}
}
