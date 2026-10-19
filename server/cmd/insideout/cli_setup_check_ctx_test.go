package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// Review-4 QUAL-04, the residue. Every other verb runs under cliCtx, so a
// Ctrl-C aborts its request (cli_ctx_bound_test.go); `setup --check` built its
// 60 s deadline on context.Background() and asked the signed-in API with a
// client that had no context at all, so a Ctrl-C left its probes running until
// they answered or the deadline struck.
//
// These tests make the CLI context cancelled before the check starts and
// while a probe waits on a listener that is slow on purpose. They require the
// check to come back at once, to report that probe as cancelled and to say the
// run was cut short (exit 1, which runInterruptible turns into 130), never
// "all configured checks passed". No signal and no interruptGrace is involved
// in-process, and the child test sets the grace to an hour, so the forced exit
// that interruptGrace brings cannot be what ends the check.

// rvSetupChkLane is one probe of the check, wired to a slow local listener.
type rvSetupChkLane struct {
	label string   // subtest name
	line  string   // the name the probe prints: api, db or llm
	args  []string // setup arguments
	// wire points the probe at the slow listener and returns the login to
	// save, the environment to set, and a channel that fires when the
	// listener was reached. The API of every other lane answers at once.
	wire func(t *testing.T) (cfg clientconfig.Config, env map[string]string, reached <-chan struct{})
}

func rvSetupChkLanes() []rvSetupChkLane {
	return []rvSetupChkLane{
		{"api anonymous", "api", []string{"--check"}, func(t *testing.T) (clientconfig.Config, map[string]string, <-chan struct{}) {
			api, reached := holdingAPI(t)
			return clientconfig.Config{APIBase: api + "/api/v1"}, nil, reached
		}},
		{"api signed in", "api", []string{"--check"}, func(t *testing.T) (clientconfig.Config, map[string]string, <-chan struct{}) {
			api, reached := holdingAPI(t)
			return clientconfig.Config{APIBase: api + "/api/v1", Token: "tok-a"}, nil, reached
		}},
		{"db", "db", []string{"--check"}, func(t *testing.T) (clientconfig.Config, map[string]string, <-chan struct{}) {
			addr, reached := rvSetupChkSlowPG(t)
			dsn := "postgres://u:pw@" + addr + "/insideout?sslmode=disable&connect_timeout=3"
			return clientconfig.Config{APIBase: rvSetupChkFastAPI(t)}, map[string]string{"DATABASE_URL": dsn}, reached
		}},
		{"llm live", "llm", []string{"--check", "--llm"}, func(t *testing.T) (clientconfig.Config, map[string]string, <-chan struct{}) {
			provider, reached := holdingAPI(t)
			return clientconfig.Config{APIBase: rvSetupChkFastAPI(t)}, map[string]string{
				"INSIDEOUT_LLM_API_KEY":  strings.Repeat("k", 40),
				"INSIDEOUT_LLM_SCHEMA":   "openai",
				"INSIDEOUT_LLM_BASE_URL": provider,
			}, reached
		}},
	}
}

// rvSetupChkWhens are the two moments a Ctrl-C can find the check: before it
// starts, and while a probe is waiting for its target.
var rvSetupChkWhens = []struct {
	name string
	arm  func(t *testing.T, reached <-chan struct{})
}{
	{"already cancelled", func(t *testing.T, _ <-chan struct{}) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		setCommandContext(t, ctx)
	}},
	{"cancelled while waiting", cancelOnArrival},
}

// rvSetupChkFastAPI is an API that answers 401 at once: the anonymous probe's pass.
func rvSetupChkFastAPI(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/api/v1"
}

// rvSetupChkSlowPG takes every connection and never says a word, as a database
// that is up and not answering. reached fires on the first connection.
func rvSetupChkSlowPG(t *testing.T) (addr string, reached <-chan struct{}) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	in := make(chan struct{}, 1)
	var mu sync.Mutex
	var held []net.Conn
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			held = append(held, c)
			mu.Unlock()
			select {
			case in <- struct{}{}:
			default:
			}
			go io.Copy(io.Discard, c) // ends when the client leaves
		}
	}()
	t.Cleanup(func() {
		ln.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, c := range held {
			c.Close()
		}
	})
	return ln.Addr().String(), in
}

// rvSetupChkRun runs `insideout setup <args>` in process against the saved
// login cfg and the extra environment, and reports what it printed and how
// long it took. The CLI context in force is whatever the test armed.
func rvSetupChkRun(t *testing.T, cfg clientconfig.Config, env map[string]string, args []string) (code int, out string, took time.Duration) {
	t.Helper()
	pinnedNoRepo(t)
	path, err := clientconfig.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := clientconfig.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
	start := time.Now()
	out = captureStdout(t, func() { _, code = runSetupCommand(args) })
	return code, out, time.Since(start)
}

// rvSetupChkLine is the printed line of one probe, "" when it printed none.
func rvSetupChkLine(out, name string) string {
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, " "+name+" — ") {
			return l
		}
	}
	return ""
}

func TestRvSetupChkAnInterruptedCheckReturnsAtOnceAndSaysCancelled(t *testing.T) {
	for _, lane := range rvSetupChkLanes() {
		for _, when := range rvSetupChkWhens {
			t.Run(lane.label+"/"+when.name, func(t *testing.T) {
				cfg, env, reached := lane.wire(t)
				when.arm(t, reached)

				code, out, took := rvSetupChkRun(t, cfg, env, lane.args)

				if took > 2*time.Second {
					t.Errorf("the check took %v, want it to end with the cancelled context, not wait for the probe or the 60 s deadline", took)
				}
				if line := rvSetupChkLine(out, lane.line); !strings.Contains(line, "skip") || !strings.Contains(line, "cancelled") {
					t.Errorf("%s line = %q, want a skip that says the probe was cancelled", lane.line, line)
				}
				if code != 1 {
					t.Errorf("exit %d, want 1: a check cut short must not pass", code)
				}
				if strings.Contains(out, "all configured checks passed") || !strings.Contains(out, "interrupted") {
					t.Errorf("output %q, want it to say the run was interrupted, not that all checks passed", out)
				}
				if when.name == "already cancelled" {
					select {
					case <-reached:
						t.Errorf("a probe still reached its target after the context was cancelled")
					default:
					}
				}
			})
		}
	}
}

// A real Ctrl-C reaches the check. The child's interruptGrace is an hour, so
// the exit that interruptGrace forces on a command that ignores the signal
// cannot be what ends it: status 130 here means the cancelled context aborted
// the probe and the check returned by itself.
func TestRvSetupChkCtrlCEndsTheCheckByItself(t *testing.T) {
	for _, lane := range []struct{ name, token string }{{"signed in", "tok-test"}, {"anonymous", ""}} {
		t.Run(lane.name, func(t *testing.T) {
			api, reached := holdingAPI(t)
			env := []string{
				graceEnv + "=1h",
				"INSIDEOUT_API=" + api + "/api/v1",
				"INSIDEOUT_TOKEN=" + lane.token,
				"INSIDEOUT_SETUP_REPO_ROOT=" + t.TempDir(), // not a checkout: the child reads no .env
				"INSIDEOUT_ENV_FILE=" + filepath.Join(t.TempDir(), "absent.env"),
			}
			cmd, errOut := startCLIWithEnv(t, env, "", "setup", "--check")
			select {
			case <-reached:
			case <-time.After(10 * time.Second):
				t.Fatalf("the check never reached the API (stderr %q)", errOut.String())
			}
			wantInterrupted(t, cmd, errOut)
		})
	}
}
