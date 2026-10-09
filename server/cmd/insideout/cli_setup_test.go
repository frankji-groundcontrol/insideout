package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// The setup subcommand's flag surface mirrors scripts/setup.sh's
// agent contract (docs/plans/2026-10-07-setup-tui.md, join phase),
// and the CLI's credential helpers resolve through clientconfig:
// env beats the config file, the config file beats the hosted
// default. HOME and XDG_CONFIG_HOME are pinned per test so the user's
// real config never influences a result, on macOS or Linux.

func writeClientConfig(t *testing.T, cfg clientconfig.Config) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	// Linux resolves the config dir from XDG_CONFIG_HOME before HOME (and CI
	// sets it), so pinning HOME alone leaves every test on one shared file.
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("INSIDEOUT_API", "")
	t.Setenv("INSIDEOUT_TOKEN", "")
	path, err := clientconfig.DefaultPath()
	if err != nil {
		t.Fatalf("resolve config path: %v", err)
	}
	if err := clientconfig.Save(path, cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func TestClientAPIBaseAndTokenFromConfigFile(t *testing.T) {
	writeClientConfig(t, clientconfig.Config{
		APIBase: "https://selfhosted.example/api/v1",
		Token:   "cfg-token",
	})

	if got := clientAPIBase(); got != "https://selfhosted.example/api/v1" {
		t.Errorf("clientAPIBase() = %q, want the config-file base", got)
	}
	if got := clientToken(); got != "cfg-token" {
		t.Errorf("clientToken() = %q, want the config-file token", got)
	}
}

func TestClientEnvBeatsConfigFile(t *testing.T) {
	writeClientConfig(t, clientconfig.Config{
		APIBase: "https://selfhosted.example/api/v1",
		Token:   "cfg-token",
	})
	t.Setenv("INSIDEOUT_API", "https://env.example/api/v1")
	t.Setenv("INSIDEOUT_TOKEN", "env-token")

	if got := clientAPIBase(); got != "https://env.example/api/v1" {
		t.Errorf("clientAPIBase() = %q, want the env base to win", got)
	}
	if got := clientToken(); got != "env-token" {
		t.Errorf("clientToken() = %q, want the env token to win", got)
	}
}

func TestClientDefaultsWithoutConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("INSIDEOUT_API", "")
	t.Setenv("INSIDEOUT_TOKEN", "")

	if got := clientAPIBase(); got != defaultAPIBase {
		t.Errorf("clientAPIBase() = %q, want the hosted default", got)
	}
	if got := clientToken(); got != "" {
		t.Errorf("clientToken() = %q, want empty", got)
	}
}

func TestSetupHostedFlag(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("INSIDEOUT_API", "")
	t.Setenv("INSIDEOUT_TOKEN", "")

	handled, code := runSetupCommand([]string{"--hosted"})
	if !handled || code != 0 {
		t.Fatalf("setup --hosted = (%v, %d), want (true, 0)", handled, code)
	}
}

func TestSetupLocalNeedsARepositoryCheckout(t *testing.T) {
	// Slice 2 makes --local perform the real assembly; pointed at a
	// directory without checkout markers it must refuse (exit 2)
	// before writing anything.
	t.Setenv("INSIDEOUT_SETUP_REPO_ROOT", t.TempDir())
	t.Setenv("INSIDEOUT_ENV_FILE", filepath.Join(t.TempDir(), "scratch.env"))
	for _, tt := range []struct {
		name string
		args []string
	}{
		{"bundled", []string{"--local", "--postgres", "bundled"}},
		{"own", []string{"--local", "--postgres", "own"}},
		{"bundled no-start", []string{"--local", "--postgres", "bundled", "--no-start"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			handled, code := runSetupCommand(tt.args)
			if !handled || code != 2 {
				t.Fatalf("setup %v = (%v, %d), want (true, 2) outside a checkout", tt.args, handled, code)
			}
		})
	}
}

func TestSetupFlagContractErrors(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
	}{
		{"both modes", []string{"--hosted", "--local"}},
		{"postgres without local", []string{"--postgres", "bundled"}},
		{"no-start without local", []string{"--no-start"}},
		{"local without postgres", []string{"--local"}},
		{"bad postgres", []string{"--local", "--postgres", "wrong"}},
		{"stray arg", []string{"--hosted", "extra"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			handled, code := runSetupCommand(tt.args)
			if !handled || code != 2 {
				t.Fatalf("setup %v = (%v, %d), want (true, 2)", tt.args, handled, code)
			}
		})
	}
}

func TestSetupInteractiveRefusesWithoutTerminal(t *testing.T) {
	// Under go test stdout is a pipe, so ui.Run takes its ErrNoTTY
	// branch and the command must exit 2 with the flags hint.
	handled, code := runSetupCommand(nil)
	if !handled || code != 2 {
		t.Fatalf("setup (no flags, no TTY) = (%v, %d), want (true, 2)", handled, code)
	}
}

func TestSetupProberRebindsClientToProbedURL(t *testing.T) {
	// The prober adapter receives the URL flow resolved and must
	// rebind the shared login client to it before sign-in (flow's
	// contract: probe precedes signIn). Behavioral check: after
	// ProbeAPI succeeds against a test server, AuthConfig() hits
	// that server, not the hosted default.
	marker := filepath.Join(t.TempDir(), "rebound")
	fake := startRebindServer(t, marker)
	base := fake.URL + "/api/v1"

	sc := &setupClient{}
	p := setupProber{sc: sc}
	if err := p.ProbeAPI(t.Context(), base); err != nil {
		t.Fatalf("ProbeAPI: %v", err)
	}
	if _, err := sc.AuthConfig(); err != nil {
		t.Fatalf("AuthConfig on the rebound client: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("AuthConfig did not hit the rebound server: %v", err)
	}
	if got := sc.Token(); got != "" {
		t.Fatalf("probe must not set a token, got %q", got)
	}
}

// startRebindServer serves the two routes the rebind test drives:
// /me answers 401 (a passing API probe), /auth/config touches the
// marker file so the test can see which base the client uses.
func startRebindServer(t *testing.T, marker string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/me":
			w.WriteHeader(http.StatusUnauthorized)
		case "/api/v1/auth/config":
			if err := os.WriteFile(marker, []byte("hit"), 0600); err != nil {
				t.Errorf("write marker: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"signupMode":"invitation_only"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}
