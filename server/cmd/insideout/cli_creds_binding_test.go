package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// Review-4 ARCH-02: the token a saved login holds belongs to the server it
// was saved for. However another server is named (--api or INSIDEOUT_API) the
// token never goes there, and a config file that cannot be read is a loud
// error — not a silent fall back to the hosted default. Every test runs the
// real main() in a child process against recording servers.

// namingB are the two ways to point a verb at server B.
var namingB = []struct {
	name string
	flag bool // true: --api B, false: INSIDEOUT_API=B
}{{"flag", true}, {"env", false}}

func TestStoredTokenIsNeverSentToAnotherServer(t *testing.T) {
	a, b := newRecServer(t), newRecServer(t)
	for _, via := range namingB {
		for _, v := range apiVerbs {
			t.Run(via.name+"/"+v.name(), func(t *testing.T) {
				writeClientConfig(t, clientconfig.Config{APIBase: a.URL, Token: "tok-a"})
				a.reset()
				b.reset()
				api, env := "", cliEnv()
				if via.flag {
					api = b.URL
				} else {
					env = cliEnv("INSIDEOUT_API=" + b.URL)
				}
				res := runCLI(t, env, v.stdin, v.args(api)...)
				if len(b.requests()) == 0 {
					t.Fatalf("the verb never reached the server it was pointed at\nexit %d\n%s", res.code, res.output())
				}
				for _, r := range b.requests() {
					if strings.Contains(r.auth, "tok-a") {
						t.Errorf("the token saved for another server was sent to this one: %s %s (Authorization %q)", r.method, r.path, r.auth)
					}
				}
				if n := len(a.requests()); n != 0 {
					t.Errorf("the saved login's server saw %d request(s)", n)
				}
			})
		}
	}
}

// A config that holds a token but no api_base was written for the hosted
// default, so it is not a credential for a self-hosted server either.
func TestTokenWithoutABaseBelongsToTheHostedDefault(t *testing.T) {
	b := newRecServer(t)
	writeClientConfig(t, clientconfig.Config{Token: "tok-hosted"})
	runCLI(t, cliEnv(), "", "whoami", "--api", b.URL)
	for _, r := range b.requests() {
		if r.auth != "" {
			t.Errorf("a token saved for the hosted default was sent to %s: Authorization %q", b.URL, r.auth)
		}
	}
	if len(b.requests()) == 0 {
		t.Fatal("whoami never reached the server named by --api")
	}
}

// The guards on the other side of the binding: the saved token still serves
// its own server (trailing slash or not), and INSIDEOUT_TOKEN is an explicit
// credential for whichever server is in effect.
func TestSavedAndEnvTokensStillReachTheirServers(t *testing.T) {
	a, b := newRecServer(t), newRecServer(t)
	writeClientConfig(t, clientconfig.Config{APIBase: a.URL + "/", Token: "tok-a"})

	runCLI(t, cliEnv(), "", "whoami", "--api", a.URL)
	runCLI(t, cliEnv("INSIDEOUT_TOKEN=tok-env"), "", "whoami", "--api", b.URL)

	if got := a.requests(); len(got) != 1 || got[0].auth != "Bearer tok-a" {
		t.Errorf("the saved token must reach the server it was saved for, got %+v", got)
	}
	if got := b.requests(); len(got) != 1 || got[0].auth != "Bearer tok-env" {
		t.Errorf("INSIDEOUT_TOKEN must reach the server in effect, got %+v", got)
	}
}

func TestUnreadableConfigIsALoudError(t *testing.T) {
	srv := newRecServer(t)
	for _, v := range []apiVerb{
		{head: words("whoami")},
		{head: words("roadmap", "list"), tail: words("p1")},
		{head: words("guide"), tail: words("p1")},
		{head: words("invite"), tail: words("invitee@example.com")},
		{head: words("login"), tail: words("user@example.com"), stdin: "pw-for-tests\n"},
		{head: words("forgot-password"), tail: words("user@example.com")},
	} {
		t.Run(v.name(), func(t *testing.T) {
			writeClientConfig(t, clientconfig.Config{})
			corruptConfig(t)
			srv.reset()
			res := runCLI(t, cliEnv(), v.stdin, v.args(srv.URL)...)
			if res.code == 0 {
				t.Errorf("exit 0 with an unreadable config\n%s", res.output())
			}
			if !strings.Contains(res.stderr, "parse config") {
				t.Errorf("stderr does not say the config could not be parsed:\n%s", res.stderr)
			}
			if strings.Contains(res.output(), "proxyconnect") {
				t.Errorf("the verb fell back to the hosted default:\n%s", res.output())
			}
			if n := len(srv.requests()); n != 0 {
				t.Errorf("the verb sent %d request(s) despite the unreadable config", n)
			}
		})
	}
}

func TestSetupCheckReportsAnUnreadableConfig(t *testing.T) {
	srv := newRecServer(t)
	writeClientConfig(t, clientconfig.Config{})
	corruptConfig(t)
	env := cliEnv("INSIDEOUT_API="+srv.URL,
		"INSIDEOUT_SETUP_REPO_ROOT="+t.TempDir(), // markerless: no repo .env is read
		"INSIDEOUT_ENV_FILE="+filepath.Join(t.TempDir(), "absent.env"))

	res := runCLI(t, env, "", "setup", "--check")

	if res.code == 0 || !strings.Contains(res.output(), "parse config") {
		t.Errorf("setup --check must fail and name the unreadable config; exit %d\n%s", res.code, res.output())
	}
}

// corruptConfig overwrites the saved config (at the path DefaultPath resolves
// under the pinned HOME) with something that is not JSON.
func corruptConfig(t *testing.T) {
	t.Helper()
	path, err := clientconfig.DefaultPath()
	if err != nil {
		t.Fatalf("resolve config path: %v", err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("corrupt config: %v", err)
	}
}
