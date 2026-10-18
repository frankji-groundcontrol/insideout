package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// Review-4 QUAL-04: a Ctrl-C used to be the default signal action, which kills
// the process wherever it stands (even between the server rotating a refresh
// token and the CLI saving the new pair). The CLI now owns a root context that
// SIGINT/SIGTERM cancel: the request in flight is aborted and the command ends
// with the conventional interrupted status, 130.

// childStderr collects a child's stderr and lets a test wait for a marker.
type childStderr struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	mark string
	seen chan struct{}
	once sync.Once
}

func (w *childStderr) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf.Write(p)
	if w.mark != "" && strings.Contains(w.buf.String(), w.mark) {
		w.once.Do(func() { close(w.seen) })
	}
	return len(p), nil
}

func (w *childStderr) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// startCLI starts main() in a child (see TestMain) with a throwaway home, a
// token that is not the developer's own, and stdin that stays open.
func startCLI(t *testing.T, mark string, args ...string) (*exec.Cmd, *childStderr) {
	t.Helper()
	home := t.TempDir()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = []string{runMainEnv + "=1", "HOME=" + home, "XDG_CONFIG_HOME=" + home + "/config", "INSIDEOUT_TOKEN=tok-test"}
	errOut := &childStderr{mark: mark, seen: make(chan struct{})}
	cmd.Stderr = errOut
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stdin.Close(); cmd.Process.Kill(); cmd.Wait() })
	return cmd, errOut
}

// wantInterrupted sends SIGINT and requires the child to end by itself with
// status 130 and say why, soon.
func wantInterrupted(t *testing.T, cmd *exec.Cmd, errOut *childStderr) {
	t.Helper()
	start := time.Now()
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	err := cmd.Wait()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 130 {
		t.Fatalf("after SIGINT the command ended with %v, want exit status 130 (stderr %q)", err, errOut.String())
	}
	if took := time.Since(start); took > 3*time.Second {
		t.Errorf("the command took %v to end after SIGINT", took)
	}
	if !strings.Contains(errOut.String(), "interrupted") {
		t.Errorf("stderr = %q, want it to say the command was interrupted", errOut.String())
	}
}

// Ctrl-C aborts the request in flight, for the verbs that sign nothing in
// (clientFromFlag) and the ones that do (runAuthCommand builds its own client).
func TestCtrlCAbortsTheRequestInFlight(t *testing.T) {
	arrived := make(chan struct{}, 1)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body) // the server notices a client that left only once the body is read
		select {
		case arrived <- struct{}{}:
		default:
		}
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
	}))
	t.Cleanup(api.Close)

	for _, args := range [][]string{
		{"whoami", "--api", api.URL},
		{"forgot-password", "--api", api.URL, "a@example.com"},
	} {
		t.Run(args[0], func(t *testing.T) {
			cmd, errOut := startCLI(t, "", args...)
			select {
			case <-arrived:
			case <-time.After(10 * time.Second):
				t.Fatalf("%v never reached the API (stderr %q)", args, errOut.String())
			}
			wantInterrupted(t, cmd, errOut)
		})
	}
}

// A command that is not in a request (here: waiting for the password) must still
// end on the first Ctrl-C, not hold the terminal until a second one.
func TestCtrlCAtAPromptEndsTheCommand(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("the API was called (%s %s) before the password was given", r.Method, r.URL.Path)
	}))
	t.Cleanup(api.Close)

	cmd, errOut := startCLI(t, "password: ", "login", "--api", api.URL, "a@example.com")
	select {
	case <-errOut.seen:
	case <-time.After(10 * time.Second):
		t.Fatalf("login never asked for the password (stderr %q)", errOut.String())
	}
	wantInterrupted(t, cmd, errOut)
}
