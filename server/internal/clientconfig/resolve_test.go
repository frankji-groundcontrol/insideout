package clientconfig

import (
	"os"
	"strings"
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

func mustResolve(t *testing.T, flagAPI string) Creds {
	t.Helper()
	got, err := Resolve(flagAPI)
	if err != nil {
		t.Fatalf("Resolve(%q): %v", flagAPI, err)
	}
	return got
}

func TestResolveReadsBaseAndTokenFromTheSavedLogin(t *testing.T) {
	pinConfig(t, Config{APIBase: "https://selfhosted.example/api/v1", Token: "cfg-token"})

	if got := mustResolve(t, ""); got.Base != "https://selfhosted.example/api/v1" || got.Token != "cfg-token" {
		t.Fatalf("Resolve(\"\") = %+v, want the saved login", got)
	}
}

func TestResolveBaseIsFlagThenEnvThenSavedThenDefault(t *testing.T) {
	pinConfig(t, Config{APIBase: "https://saved.example/api/v1"})
	if got := mustResolve(t, "").Base; got != "https://saved.example/api/v1" {
		t.Errorf("saved base = %q", got)
	}
	t.Setenv("INSIDEOUT_API", "https://env.example/api/v1")
	if got := mustResolve(t, "").Base; got != "https://env.example/api/v1" {
		t.Errorf("env must beat the saved login, got %q", got)
	}
	if got := mustResolve(t, "https://flag.example/api/v1").Base; got != "https://flag.example/api/v1" {
		t.Errorf("flag must beat env, got %q", got)
	}

	pinConfig(t, Config{})
	if got := mustResolve(t, "").Base; got != apiclient.DefaultBase {
		t.Errorf("without any source the base is the hosted default, got %q", got)
	}
}

// Review-4 ARCH-02: the saved token is a credential for the saved base only.
func TestResolveSavedTokenIsBoundToItsBase(t *testing.T) {
	const a, b = "https://a.example/api/v1", "https://b.example/api/v1"
	pinConfig(t, Config{APIBase: a, Token: "tok-a"})

	for _, tc := range []struct{ flag, want string }{
		{"", "tok-a"},
		{a, "tok-a"},
		{a + "/", "tok-a"}, // a trailing slash names the same server
		{b, ""},
	} {
		if got := mustResolve(t, tc.flag).Token; got != tc.want {
			t.Errorf("Resolve(%q).Token = %q, want %q", tc.flag, got, tc.want)
		}
	}

	t.Setenv("INSIDEOUT_API", b)
	if got := mustResolve(t, "").Token; got != "" {
		t.Errorf("INSIDEOUT_API pointing elsewhere must not carry the saved token, got %q", got)
	}
}

// A config that holds a token but no api_base was saved for the hosted default.
func TestResolveTokenWithoutABaseBelongsToTheHostedDefault(t *testing.T) {
	pinConfig(t, Config{Token: "tok-hosted"})

	if got := mustResolve(t, ""); got.Base != apiclient.DefaultBase || got.Token != "tok-hosted" {
		t.Errorf("Resolve(\"\") = %+v, want the hosted default with the saved token", got)
	}
	if got := mustResolve(t, "https://selfhosted.example/api/v1").Token; got != "" {
		t.Errorf("a self-hosted server must not receive the hosted token, got %q", got)
	}
}

func TestResolveEnvTokenIsAnExplicitCredentialForTheEffectiveBase(t *testing.T) {
	pinConfig(t, Config{APIBase: "https://a.example/api/v1", Token: "cfg-token"})
	t.Setenv("INSIDEOUT_TOKEN", "env-token")

	for _, flag := range []string{"", "https://a.example/api/v1", "https://b.example/api/v1"} {
		if got := mustResolve(t, flag).Token; got != "env-token" {
			t.Errorf("Resolve(%q).Token = %q, want the env token", flag, got)
		}
	}
}

func TestResolveWithoutAnyConfigIsSignedOutOnTheHostedDefault(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	clearCredentialEnv(t)

	if got := mustResolve(t, ""); got.Base != apiclient.DefaultBase || got.Token != "" {
		t.Fatalf("Resolve(\"\") = %+v, want the hosted default and no token", got)
	}
}

func TestResolveRefusesAnUnreadableConfig(t *testing.T) {
	pinConfig(t, Config{})
	path, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, flag := range []string{"", "https://b.example/api/v1"} {
		got, err := Resolve(flag)
		if err == nil || !strings.Contains(err.Error(), "parse config "+path) {
			t.Fatalf("Resolve(%q) error = %v, want one naming the unreadable config", flag, err)
		}
		if got != (Creds{}) {
			t.Errorf("Resolve(%q) = %+v with an error, want zero Creds", flag, got)
		}
	}
}

// A machine with no config directory at all (no HOME) has no saved login; it
// is not an error, and the environment still works.
func TestResolveWithoutAConfigDirectoryUsesTheEnvironmentAlone(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("INSIDEOUT_API", "https://env.example/api/v1")
	t.Setenv("INSIDEOUT_TOKEN", "env-token")
	if _, err := DefaultPath(); err == nil {
		t.Skip("this platform resolves a config directory without HOME")
	}

	got := mustResolve(t, "")
	if got.Base != "https://env.example/api/v1" || got.Token != "env-token" {
		t.Fatalf("Resolve(\"\") = %+v, want the environment's base and token", got)
	}
}

func TestCredsClientCarriesTheToken(t *testing.T) {
	if got := (Creds{Base: "https://x.example/api/v1/", Token: "t"}).Client().Token(); got != "t" {
		t.Fatalf("client token = %q", got)
	}
}
