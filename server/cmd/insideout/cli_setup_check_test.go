package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `setup --check` lanes. Network-dependent lanes are exercised via
// httptest for the API; GitHub's fixed URL and live LLM probes are
// covered by the probe package's own tests and the skip lanes here.

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	w.Close()
	buf := make([]byte, 16384)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

func pinnedNoRepo(t *testing.T) {
	t.Helper()
	t.Setenv("INSIDEOUT_SETUP_REPO_ROOT", t.TempDir()) // markerless → repo lookup fails
	t.Setenv("INSIDEOUT_ENV_FILE", filepath.Join(t.TempDir(), "absent.env"))
	t.Setenv("HOME", t.TempDir())
	// Linux resolves the config dir from XDG_CONFIG_HOME before HOME (and CI
	// sets it), so pinning HOME alone leaves every test on one shared file.
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("INSIDEOUT_API", "")
	t.Setenv("INSIDEOUT_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("INSIDEOUT_LLM_API_KEY", "")
	t.Setenv("DATABASE_URL", "")
}

func TestCheckAPIOKAgainstTestServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	pinnedNoRepo(t)
	t.Setenv("INSIDEOUT_API", srv.URL+"/api/v1")

	handled, code := runSetupCommand([]string{"--check"})
	if !handled || code != 0 {
		t.Fatalf("setup --check = (%v, %d), want (true, 0)", handled, code)
	}
}

func TestCheckAPISkipWhenUnreachable(t *testing.T) {
	pinnedNoRepo(t)
	t.Setenv("INSIDEOUT_API", "http://127.0.0.1:1/api/v1") // closed port → skip, offline-safe

	handled, code := runSetupCommand([]string{"--check"})
	if !handled || code != 0 {
		t.Fatalf("unreachable API must skip (exit 0), got (%v, %d)", handled, code)
	}
}

func TestCheckDBSkipsWithoutDSN(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	pinnedNoRepo(t)
	t.Setenv("INSIDEOUT_API", srv.URL+"/api/v1")

	handled, code := runSetupCommand([]string{"--check"})
	if !handled || code != 0 {
		t.Fatalf("no-DSN db lane must skip, got (%v, %d)", handled, code)
	}
}

func TestCheckDBFailsWhenConfiguredAndDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	pinnedNoRepo(t)
	t.Setenv("INSIDEOUT_API", srv.URL+"/api/v1")
	t.Setenv("DATABASE_URL", "postgres://u:pw@127.0.0.1:1/insideout?sslmode=disable&connect_timeout=2")

	handled, code := runSetupCommand([]string{"--check"})
	if !handled || code != 1 {
		t.Fatalf("configured-but-down database must fail (exit 1), got (%v, %d)", handled, code)
	}
}

func TestCheckLLMLanes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	t.Run("no key skips", func(t *testing.T) {
		pinnedNoRepo(t)
		t.Setenv("INSIDEOUT_API", srv.URL+"/api/v1")
		if handled, code := runSetupCommand([]string{"--check"}); !handled || code != 0 {
			t.Fatalf("got (%v, %d)", handled, code)
		}
	})
	t.Run("bad shape fails", func(t *testing.T) {
		pinnedNoRepo(t)
		t.Setenv("INSIDEOUT_API", srv.URL+"/api/v1")
		t.Setenv("INSIDEOUT_LLM_API_KEY", "short")
		if handled, code := runSetupCommand([]string{"--check"}); !handled || code != 1 {
			t.Fatalf("bad-shape key must fail, got (%v, %d)", handled, code)
		}
	})
	t.Run("live probe skips for non-openai schema", func(t *testing.T) {
		pinnedNoRepo(t)
		t.Setenv("INSIDEOUT_API", srv.URL+"/api/v1")
		t.Setenv("INSIDEOUT_LLM_API_KEY", strings.Repeat("k", 40))
		t.Setenv("INSIDEOUT_LLM_SCHEMA", "anthropic")
		if handled, code := runSetupCommand([]string{"--check", "--llm"}); !handled || code != 0 {
			t.Fatalf("non-openai live probe must skip, got (%v, %d)", handled, code)
		}
	})
	t.Run("default messages schema skips the legacy live probe", func(t *testing.T) {
		pinnedNoRepo(t)
		t.Setenv("INSIDEOUT_API", srv.URL+"/api/v1")
		t.Setenv("INSIDEOUT_LLM_API_KEY", strings.Repeat("k", 40))
		t.Setenv("INSIDEOUT_LLM_BASE_URL", srv.URL)
		t.Setenv("INSIDEOUT_LLM_SCHEMA", "")
		if handled, code := runSetupCommand([]string{"--check", "--llm"}); !handled || code != 0 {
			t.Fatalf("default messages schema must skip the legacy live probe, got (%v, %d)", handled, code)
		}
	})
}

func TestCheckFlagExclusivity(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
	}{
		{"check plus hosted", []string{"--check", "--hosted"}},
		{"check plus local", []string{"--check", "--local"}},
		{"llm without check", []string{"--llm"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			handled, code := runSetupCommand(tt.args)
			if !handled || code != 2 {
				t.Fatalf("setup %v = (%v, %d), want (true, 2)", tt.args, handled, code)
			}
		})
	}
}
