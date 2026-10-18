package clientconfig

import (
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// pinConfig gives the test its own HOME (and an unset XDG_CONFIG_HOME, which
// Linux consults first), clears the credential environment, and saves cfg
// where DefaultPath resolves it.
func pinConfig(t *testing.T, cfg Config) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	clearCredentialEnv(t)
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
}

func TestResolveReadsBaseAndTokenFromTheSavedLogin(t *testing.T) {
	pinConfig(t, Config{APIBase: "https://selfhosted.example/api/v1", Token: "cfg-token"})

	got := Resolve("")
	if got.Base != "https://selfhosted.example/api/v1" || got.Token != "cfg-token" {
		t.Fatalf("Resolve(\"\") = %+v, want the saved login", got)
	}
}

func TestResolveBaseIsFlagThenEnvThenSavedThenDefault(t *testing.T) {
	pinConfig(t, Config{APIBase: "https://saved.example/api/v1"})
	if got := Resolve("").Base; got != "https://saved.example/api/v1" {
		t.Errorf("saved base = %q", got)
	}
	t.Setenv("INSIDEOUT_API", "https://env.example/api/v1")
	if got := Resolve("").Base; got != "https://env.example/api/v1" {
		t.Errorf("env must beat the saved login, got %q", got)
	}
	if got := Resolve("https://flag.example/api/v1").Base; got != "https://flag.example/api/v1" {
		t.Errorf("flag must beat env, got %q", got)
	}

	pinConfig(t, Config{})
	if got := Resolve("").Base; got != apiclient.DefaultBase {
		t.Errorf("without any source the base is the hosted default, got %q", got)
	}
}

func TestResolveEnvTokenBeatsTheSavedToken(t *testing.T) {
	pinConfig(t, Config{APIBase: "https://selfhosted.example/api/v1", Token: "cfg-token"})
	t.Setenv("INSIDEOUT_TOKEN", "env-token")

	if got := Resolve("").Token; got != "env-token" {
		t.Fatalf("Resolve token = %q, want the env token to win", got)
	}
}

func TestResolveWithoutAnyConfigIsSignedOutOnTheHostedDefault(t *testing.T) {
	pinConfig(t, Config{})

	got := Resolve("")
	if got.Base != apiclient.DefaultBase || got.Token != "" {
		t.Fatalf("Resolve(\"\") = %+v, want the hosted default and no token", got)
	}
}

func TestCredsClientCarriesBaseAndToken(t *testing.T) {
	c := Creds{Base: "https://x.example/api/v1/", Token: "t"}.Client()
	if c.Token() != "t" {
		t.Fatalf("client token = %q", c.Token())
	}
}
