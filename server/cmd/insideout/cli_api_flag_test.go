package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// Review-4 ARCH-01: every verb that talks to the API resolves its base the
// same way (--api, then INSIDEOUT_API, then the saved login, then the hosted
// API) and builds its client only after its flags are parsed. These tests run
// the real main() in a child process, against recording servers, for every
// such verb.

type seenReq struct{ method, path, auth string }

// recServer answers every request 200 {} and remembers what it was asked.
type recServer struct {
	*httptest.Server
	mu   sync.Mutex
	seen []seenReq
}

func newRecServer(t *testing.T) *recServer {
	t.Helper()
	rs := &recServer{}
	rs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rs.mu.Lock()
		rs.seen = append(rs.seen, seenReq{r.Method, r.URL.Path, r.Header.Get("Authorization")})
		rs.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(rs.Close)
	return rs
}

func (rs *recServer) requests() []seenReq {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return append([]seenReq(nil), rs.seen...)
}

func (rs *recServer) reset() {
	rs.mu.Lock()
	rs.seen = nil
	rs.mu.Unlock()
}

// cliEnv is the child's environment: its HOME (where writeClientConfig put
// the config), plus a dead proxy so a verb that falls back to the hosted API
// fails at once instead of reaching a real host. Loopback servers bypass
// the proxy.
func cliEnv(extra ...string) []string {
	env := []string{
		"HOME=" + os.Getenv("HOME"),
		"HTTPS_PROXY=http://127.0.0.1:1", "HTTP_PROXY=http://127.0.0.1:1",
	}
	return append(env, extra...)
}

// apiVerb is one verb that makes API calls: its head words, then (after the
// --api flag, if any) its remaining arguments and a scripted stdin.
type apiVerb struct {
	head, tail []string
	stdin      string
}

func (v apiVerb) name() string {
	n := ""
	for _, w := range v.head {
		n += w + " "
	}
	return n[:len(n)-1]
}

func (v apiVerb) args(api string) []string {
	out := append([]string(nil), v.head...)
	if api != "" {
		out = append(out, "--api", api)
	}
	return append(out, v.tail...)
}

func words(w ...string) []string { return w }

var apiVerbs = []apiVerb{
	{head: words("whoami")},
	{head: words("workspaces")},
	{head: words("workspaces", "create"), tail: words("--title", "T")},
	{head: words("projects"), tail: words("ws1")},
	{head: words("projects", "create"), tail: words("--title", "T", "ws1")},
	{head: words("prd"), tail: words("p1")},
	{head: words("prd", "update"), tail: words("--title", "T", "p1")},
	{head: words("ideas"), tail: words("ws1")},
	{head: words("evidence"), tail: words("n1")},
	{head: words("join"), tail: words("code1")},
	{head: words("idea", "create"), tail: words("--title", "T", "ws1")},
	{head: words("idea", "update"), tail: words("--title", "T", "i1")},
	{head: words("idea", "delete"), tail: words("i1")},
	{head: words("idea", "convert"), tail: words("i1")},
	{head: words("idea", "proposal-decide"), tail: words("--accept", "u1")},
	{head: words("roadmap", "list"), tail: words("p1")},
	{head: words("roadmap", "progress"), tail: words("p1")},
	{head: words("roadmap", "presence"), tail: words("p1")},
	{head: words("roadmap", "add"), tail: words("--title", "T", "p1")},
	{head: words("roadmap", "update"), tail: words("--title", "T", "n1")},
	{head: words("roadmap", "move"), tail: words("--position", "1", "n1")},
	{head: words("roadmap", "delete"), tail: words("n1")},
	{head: words("presence"), tail: words("p1")},
	{head: words("build"), tail: words("p1")},
	{head: words("expand"), tail: words("n1")},
	{head: words("build-confirm"), tail: words("-tree", "{}", "p1")},
	{head: words("expand-confirm"), tail: words("-children", "[]", "n1")},
	{head: words("repo"), tail: words("set", "p1", "https://example.com/org/repo")},
	{head: words("sync"), tail: words("p1")},
	{head: words("guide"), tail: words("p1")},
	{head: words("commit"), tail: words("--name", "N", "--audience", "decision", "p1")},
	{head: words("versions"), tail: words("p1")},
	{head: words("readiness"), tail: words("p1")},
	{head: words("view"), tail: words("p1")},
	{head: words("agent-context"), tail: words("p1")},
	{head: words("checkpoint"), tail: words("p1", "summary")},
	{head: words("propose"), tail: words("p1", "summary")},
	{head: words("revisions"), tail: words("p1")},
	{head: words("snapshot"), tail: words("p1")},
	{head: words("invite"), tail: words("invitee@example.com")},
	{head: words("invite"), tail: words("revoke", "inv1")},
	{head: words("invites")},
	{head: words("signup-mode")},
	{head: words("signup-mode"), tail: words("open")},
	{head: words("login"), tail: words("user@example.com"), stdin: "pw-for-tests\n"},
	{head: words("register"), tail: words("user@example.com"), stdin: "pw-for-tests\n"},
	{head: words("forgot-password"), tail: words("user@example.com")},
	{head: words("reset-password"), stdin: "link-token\nnew-pw-for-tests\n"},
}

func TestEveryAPIVerbHonorsAPIFlagOverConfig(t *testing.T) {
	a, b := newRecServer(t), newRecServer(t)
	for _, v := range apiVerbs {
		t.Run(v.name(), func(t *testing.T) {
			writeClientConfig(t, clientconfig.Config{APIBase: a.URL, Token: "tok-a"})
			a.reset()
			b.reset()
			res := runCLI(t, cliEnv(), v.stdin, v.args(b.URL)...)
			if n := len(b.requests()); n == 0 {
				t.Errorf("--api %s was ignored: the server named by the flag saw no request\nexit %d\n%s", b.URL, res.code, res.output())
			}
			if n := len(a.requests()); n != 0 {
				t.Errorf("the saved login's server saw %d request(s) although --api named another\n%s", n, res.output())
			}
		})
	}
}

func TestEveryAPIVerbUsesTheSavedBaseWithoutFlag(t *testing.T) {
	a := newRecServer(t)
	for _, v := range apiVerbs {
		t.Run(v.name(), func(t *testing.T) {
			writeClientConfig(t, clientconfig.Config{APIBase: a.URL, Token: "tok-a"})
			a.reset()
			res := runCLI(t, cliEnv(), v.stdin, v.args("")...)
			if len(a.requests()) == 0 {
				t.Errorf("the saved api_base was not used (the verb went to the hosted default)\nexit %d\n%s", res.code, res.output())
			}
		})
	}
}

// The three sources of a base, in order: --api beats INSIDEOUT_API beats the
// saved login. The verbs span the three client-construction styles.
func TestAPIBasePrecedenceFlagEnvConfig(t *testing.T) {
	for _, v := range []apiVerb{
		{head: words("whoami")},
		{head: words("guide"), tail: words("p1")},
		{head: words("roadmap", "list"), tail: words("p1")},
	} {
		t.Run(v.name(), func(t *testing.T) {
			saved, env, flagged := newRecServer(t), newRecServer(t), newRecServer(t)
			writeClientConfig(t, clientconfig.Config{APIBase: saved.URL})
			runCLI(t, cliEnv("INSIDEOUT_API="+env.URL), "", v.args(flagged.URL)...)
			if len(flagged.requests()) == 0 || len(env.requests()) != 0 || len(saved.requests()) != 0 {
				t.Errorf("flag must win: flagged=%d env=%d saved=%d", len(flagged.requests()), len(env.requests()), len(saved.requests()))
			}
			flagged.reset()
			runCLI(t, cliEnv("INSIDEOUT_API="+env.URL), "", v.args("")...)
			if len(env.requests()) == 0 || len(saved.requests()) != 0 {
				t.Errorf("env must beat the saved login: env=%d saved=%d", len(env.requests()), len(saved.requests()))
			}
		})
	}
}
