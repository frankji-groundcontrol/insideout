package mcpserver

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// The MCP client must resolve credentials like every other client surface
// (docs/plans/2026-10-07-setup-tui.md): the environment wins, and a
// token written by `insideout setup`'s config file is the fallback.
// These tests pin HOME and XDG_CONFIG_HOME so the user's real config file
// never influences them.

// setConfigToken writes token to the config file location
// clientconfig itself resolves under the test's pinned environment —
// darwin's UserConfigDir ignores XDG_CONFIG_HOME, so the path must
// come from DefaultPath(), not from an assumed layout.
func setConfigToken(t *testing.T, token string) {
	t.Helper()
	path, err := clientconfig.DefaultPath()
	if err != nil {
		t.Fatalf("resolve config path: %v", err)
	}
	if err := clientconfig.Save(path, clientconfig.Config{Token: token}); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func TestClientTokenFallsBackToConfigFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// Linux resolves the config dir from XDG_CONFIG_HOME before HOME (and CI
	// sets it), so pinning HOME alone leaves every test on one shared file.
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("INSIDEOUT_TOKEN", "")
	setConfigToken(t, "cfg-file-token")

	if got := client("").Token(); got != "cfg-file-token" {
		t.Fatalf("client token = %q, want the config-file token", got)
	}
}

func TestClientTokenEnvBeatsConfigFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("INSIDEOUT_TOKEN", "env-token")
	setConfigToken(t, "cfg-file-token")

	if got := client("").Token(); got != "env-token" {
		t.Fatalf("client token = %q, want the env token to win", got)
	}
}

func TestClientTokenInProcessBeatsBoth(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("INSIDEOUT_TOKEN", "env-token")
	setConfigToken(t, "cfg-file-token")
	setSessionToken("in-process-token")
	t.Cleanup(func() { setSessionToken("") })

	if got := client("").Token(); got != "in-process-token" {
		t.Fatalf("client token = %q, want the in-process token to win", got)
	}
}

// recordingAPI answers every request {"email":"a@example.com"} and returns,
// via the second value, what it was asked as "METHOD path Authorization".
func recordingAPI(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Method+" "+r.URL.Path+" "+r.Header.Get("Authorization"))
		mu.Unlock()
		w.Write([]byte(`{"email":"a@example.com"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
}

// pinConfigHome gives the test its own empty config home and clears the
// credential environment, so only what the test saves can influence it.
func pinConfigHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("INSIDEOUT_API", "")
	t.Setenv("INSIDEOUT_TOKEN", "")
	setSessionToken("")
}

func saveConfig(t *testing.T, cfg clientconfig.Config) {
	t.Helper()
	path, err := clientconfig.DefaultPath()
	if err != nil {
		t.Fatalf("resolve config path: %v", err)
	}
	if err := clientconfig.Save(path, cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// Review-4 ARCH-01: the MCP binary passes no --api, so its tools must follow
// the saved login's api_base — the same base its token is read for — instead
// of the hosted default.
func TestBuildFollowsTheSavedLoginsBase(t *testing.T) {
	pinConfigHome(t)
	srv, seen := recordingAPI(t)
	saveConfig(t, clientconfig.Config{APIBase: srv.URL, Token: "cfg-tok"})

	res := drive(t, "", []string{toolCall(1, "whoami", `{}`)})

	if res[1].Result.IsError {
		t.Fatalf("whoami failed: %+v", res[1])
	}
	if got := seen(); len(got) != 1 || got[0] != "GET /me Bearer cfg-tok" {
		t.Fatalf("saved api_base server saw %q, want one authenticated GET /me", got)
	}
}
