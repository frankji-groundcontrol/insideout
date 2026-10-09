package mcpserver

import (
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// sessionToken must resolve like every other client surface
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

func TestSessionTokenFallsBackToConfigFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// Linux resolves the config dir from XDG_CONFIG_HOME before HOME (and CI
	// sets it), so pinning HOME alone leaves every test on one shared file.
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("INSIDEOUT_TOKEN", "")
	setConfigToken(t, "cfg-file-token")

	if got := sessionToken(); got != "cfg-file-token" {
		t.Fatalf("sessionToken() = %q, want the config-file token", got)
	}
}

func TestSessionTokenEnvBeatsConfigFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("INSIDEOUT_TOKEN", "env-token")
	setConfigToken(t, "cfg-file-token")

	if got := sessionToken(); got != "env-token" {
		t.Fatalf("sessionToken() = %q, want the env token to win", got)
	}
}

func TestSessionTokenInProcessBeatsBoth(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("INSIDEOUT_TOKEN", "env-token")
	setConfigToken(t, "cfg-file-token")
	setSessionToken("in-process-token")
	t.Cleanup(func() { setSessionToken("") })

	if got := sessionToken(); got != "in-process-token" {
		t.Fatalf("sessionToken() = %q, want the in-process token to win", got)
	}
}
