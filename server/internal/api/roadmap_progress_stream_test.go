package api

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// TestRoadmapProgressStream_StatusChangeEmitsEvent opens the progress SSE
// channel, then moves a node pending → in_progress. The stream must emit a
// progress frame carrying the assembled payload with that new status.
// DB-gated like prd_to_progress_test.go: the node has to land in real PostgreSQL.
func TestRoadmapProgressStream_StatusChangeEmitsEvent(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	st, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	ctx := context.Background()

	admin, err := st.CreateUser(ctx, "progress-sse-"+uuid.NewString()+"@test.local", "x", "progress-sse")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Progress SSE", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, admin.ID, ws.ID, "Progress SSE", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	node, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, nil, "Ship the stream", "")
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	tokens := auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour)
	token, err := tokens.MintAccessToken(admin.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	srv := NewServer(st, tokens, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/projects/"+proj.ID.String()+"/roadmap/progress/stream", nil)
	if err != nil {
		t.Fatalf("build stream request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("stream status %d, want 200; body %s", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type = %q, want text/event-stream", ct)
	}

	br := bufio.NewReader(resp.Body)
	// Connecting emits the current snapshot first, so the following status
	// change is a later frame and not the one already on the wire.
	if _, err := readProgressFrame(t, br); err != nil {
		t.Fatalf("initial progress frame: %v", err)
	}

	patch(t, ts, token, "/api/v1/roadmap/"+node.ID.String(), `{"status":"in_progress"}`, http.StatusOK)

	frame, err := readProgressFrame(t, br)
	if err != nil {
		t.Fatalf("progress frame after status change: %v", err)
	}
	if !strings.Contains(frame, "event: progress") {
		t.Fatalf("frame = %q, want event: progress", frame)
	}
	if !strings.Contains(frame, node.ID.String()) || !strings.Contains(frame, `"status":"in_progress"`) {
		t.Fatalf("frame = %q, want the node id and its new status", frame)
	}
}

func readProgressFrame(t *testing.T, br *bufio.Reader) (string, error) {
	t.Helper()
	type result struct {
		frame string
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		f, err := readFrame(br)
		ch <- result{f, err}
	}()
	select {
	case got := <-ch:
		return got.frame, got.err
	case <-time.After(5 * time.Second):
		t.Fatal("progress frame never arrived")
		return "", nil
	}
}
