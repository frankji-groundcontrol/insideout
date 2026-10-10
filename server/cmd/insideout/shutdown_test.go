package main

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// shutdownRig is the real runServe on a loopback port, plus a user, project
// and conversation to drive its streams. shutdown does to runServe's context
// what SIGTERM does in production; returned closes when runServe returns.
type shutdownRig struct {
	st       *store.Store
	base     string
	token    string
	userID   uuid.UUID
	project  uuid.UUID
	convID   uuid.UUID
	shutdown context.CancelFunc
	returned chan struct{}
}

// startShutdownRig serves the real handler stack. A non-empty llmBaseURL
// points the coach at that provider (Anthropic Messages wire format);
// otherwise the offline template coach is used. DB-gated: every handler
// checks membership against real RLS.
func startShutdownRig(t *testing.T, llmBaseURL string) *shutdownRig {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	if owner := os.Getenv("DATABASE_OWNER_URL"); owner != "" {
		if err := st.AttachMigratePool(ctx, owner); err != nil {
			t.Fatalf("attach owner pool: %v", err)
		}
	}

	user, err := st.CreateUser(ctx, "shutdown-"+uuid.NewString()+"@test.local", "x", "shutdown-tester")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, user.ID, "Shutdown Rig", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, user.ID, ws.ID, "Shutdown Rig", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	idea, err := st.CreateIdea(ctx, user.ID, ws.ID, "Idea", "content")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	_, conv, err := st.ConvertIdea(ctx, user.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pick a port: %v", err)
	}
	addr := l.Addr().String()
	l.Close()
	cfg := &config.Config{Addr: addr, DatabaseURL: url, JWTSecret: "a-32-byte-or-longer-test-secret!", AccessTTL: time.Hour}
	if llmBaseURL != "" {
		cfg.AIAuthToken, cfg.AIBaseURL, cfg.AIModel, cfg.AISchema = "test-key", llmBaseURL, "test-model", "messages"
	}
	token, err := auth.NewTokenIssuer(cfg.JWTSecret, cfg.AccessTTL).MintAccessToken(user.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}

	runCtx, shutdown := context.WithCancel(ctx)
	rig := &shutdownRig{st: st, base: "http://" + addr, token: token, userID: user.ID, project: proj.ID,
		convID: conv.ID, shutdown: shutdown, returned: make(chan struct{})}
	t.Cleanup(shutdown) // a failed test must not leave the server up
	go func() {
		defer close(rig.returned)
		runServe(runCtx, slog.New(slog.NewTextHandler(io.Discard, nil)), st, cfg)
	}()

	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		if resp, err := http.Get(rig.base + "/healthz"); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return rig
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("runServe never started serving")
		}
	}
}

// open starts an authenticated request whose response is an event stream and
// returns its reader; cleanup closes the body.
func (r *shutdownRig) open(t *testing.T, method, path, body string) *bufio.Reader {
	t.Helper()
	req, err := http.NewRequest(method, r.base+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Accept", "text/event-stream")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("%s %s: status %d, want 200; body %s", method, path, resp.StatusCode, b)
	}
	return bufio.NewReader(resp.Body)
}

// waitForLine reads the stream until a line equals want.
func waitForLine(t *testing.T, br *bufio.Reader, want string) {
	t.Helper()
	found := make(chan error, 1)
	go func() {
		for {
			line, err := br.ReadString('\n')
			if strings.TrimSpace(line) == want {
				found <- nil
				return
			}
			if err != nil {
				found <- err
				return
			}
		}
	}()
	select {
	case err := <-found:
		if err != nil {
			t.Fatalf("stream ended before a %q line: %v", want, err)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("no %q line within 10s", want)
	}
}

// REL-09: runServe must register the stream-closing hook, so an open canvas
// or progress tab does not hold the shutdown window. Nothing else may end
// the stream here: the only other thing that cuts a request is the drain,
// which is far longer than this test waits.
func TestRunServe_ShutdownEndsAnOpenProgressStream(t *testing.T) {
	rig := startShutdownRig(t, "")
	br := rig.open(t, http.MethodGet, "/api/v1/projects/"+rig.project.String()+"/roadmap/progress/stream", "")
	waitForLine(t, br, "event: progress")

	rig.shutdown()
	select {
	case <-rig.returned:
	case <-time.After(5 * time.Second):
		t.Fatal("runServe still serving 5s after shutdown began: an open progress stream holds the window")
	}
}
