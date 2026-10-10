package api

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// openSSE opens an authenticated event stream and returns a frame reader plus
// a cancel that ends it from the client side. The caller reads frames with
// readProgressFrame; cleanup closes the body.
func openSSE(t *testing.T, ts *httptest.Server, token, path string) (*bufio.Reader, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+path, nil)
	if err != nil {
		t.Fatalf("build stream request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("open %s: status %d, want 200; body %s", path, resp.StatusCode, body)
	}
	return bufio.NewReader(resp.Body), cancel
}

// REL-09 / REL-16: http.Server.Shutdown never cancels handler contexts, and
// the passive streams (roadmap progress, canvas presence) select only on the
// request context, so one connected tab made every deploy burn the whole
// shutdown window. The server must end them when shutdown starts, and the
// presence stream must leave like it does on a client disconnect. DB-gated:
// both handlers check project membership against real RLS.
func TestShutdown_EndsOpenProgressAndPresenceStreams(t *testing.T) {
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
	user, err := st.CreateUser(ctx, "shutdown-"+uuid.NewString()+"@test.local", "x", "shutdown-tester")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, user.ID, "Shutdown Streams", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, user.ID, ws.ID, "Shutdown Streams", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	tokens := auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour)
	token, err := tokens.MintAccessToken(user.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}

	srv := NewServer(st, tokens, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
	ts := httptest.NewUnstartedServer(srv.Handler())
	ts.Config.RegisterOnShutdown(srv.CloseStreams) // what runServe registers
	ts.Start()
	t.Cleanup(ts.Close)

	base := "/api/v1/projects/" + proj.ID.String()
	progress, _ := openSSE(t, ts, token, base+"/roadmap/progress/stream")
	presence, _ := openSSE(t, ts, token, base+"/presence/stream?client=tab-1")
	for name, br := range map[string]*bufio.Reader{"progress": progress, "presence": presence} {
		if _, err := readProgressFrame(t, br); err != nil {
			t.Fatalf("%s stream: first frame: %v", name, err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	start := time.Now()
	err = ts.Config.Shutdown(shutdownCtx)
	if took := time.Since(start); err != nil || took > 3*time.Second {
		t.Fatalf("Shutdown with a progress and a presence stream open = %v after %s, want nil within 3s", err, took.Round(time.Millisecond))
	}
	if left := srv.presence.List(proj.ID.String()); len(left) != 0 {
		t.Fatalf("presence still lists %d viewer(s) after shutdown, want the stream to leave", len(left))
	}
}
