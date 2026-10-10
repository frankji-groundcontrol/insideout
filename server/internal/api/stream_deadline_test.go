package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
)

// deadlineWriter is the response writer under a stream in these tests. Like a
// real connection it takes a write deadline, and it notes every write that runs
// while none is armed: a deadline counts only while it is still ahead and no
// further out than httpx.SSEWriteTimeout, and a flush uses it up, so a deadline
// armed once at the top of a stream does not cover the writes that follow.
type deadlineWriter struct {
	mu     sync.Mutex
	header http.Header
	armed  bool
	frame  strings.Builder
	frames chan string // one entry per flush; frames nobody waits for are dropped
	bare   []string    // first line of every write that ran with no deadline armed
}

func newDeadlineWriter() *deadlineWriter {
	return &deadlineWriter{header: http.Header{}, frames: make(chan string, 1024)}
}

func (d *deadlineWriter) Header() http.Header { return d.header }
func (d *deadlineWriter) WriteHeader(int)     {}

func (d *deadlineWriter) SetWriteDeadline(t time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	left := time.Until(t)
	d.armed = left > 0 && left <= httpx.SSEWriteTimeout
	return nil
}

func (d *deadlineWriter) Write(p []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.armed {
		d.bare = append(d.bare, strings.SplitN(string(p), "\n", 2)[0])
	}
	d.frame.Write(p)
	return len(p), nil
}

func (d *deadlineWriter) Flush() {
	d.mu.Lock()
	frame := d.frame.String()
	d.frame.Reset()
	d.armed = false
	d.mu.Unlock()
	select {
	case d.frames <- frame:
	default:
	}
}

// await reads frames until each wanted prefix has been written once.
func (d *deadlineWriter) await(t *testing.T, want ...string) {
	t.Helper()
	pending := map[string]bool{}
	for _, p := range want {
		pending[p] = true
	}
	timeout := time.After(5 * time.Second)
	for len(pending) > 0 {
		select {
		case f := <-d.frames:
			for p := range pending {
				if strings.HasPrefix(f, p) {
					delete(pending, p)
				}
			}
		case <-timeout:
			t.Fatalf("stream never wrote a frame starting with %v", pending)
		}
	}
}

func (d *deadlineWriter) requireEveryWriteBounded(t *testing.T) {
	t.Helper()
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.bare) > 0 {
		t.Fatalf("%d stream write(s) ran with no write deadline armed, so a client that stopped reading would pin the handler: %q", len(d.bare), d.bare)
	}
}

// serveStream runs a GET through the full handler chain into a deadlineWriter
// until stop is called (the cleanup calls it too) and returns once the handler
// has returned.
func serveStream(t *testing.T, h http.Handler, token, path string) (*deadlineWriter, func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, path, nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+token)
	w := newDeadlineWriter()
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.ServeHTTP(w, req)
	}()
	stop := func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("stream handler still running 5s after its request was cancelled")
		}
	}
	t.Cleanup(stop)
	return w, stop
}

// REL-15: the stream handlers wrote to the connection with no deadline, so a
// client that stops reading (a black-holed network, a suspended laptop) pinned
// the handler goroutine until the kernel gave up. Every write the presence and
// roadmap-progress streams make must be armed with its own bounded deadline.
// DB-gated: both handlers check project membership against real RLS.
func TestStreams_ArmAWriteDeadlineBeforeEveryWrite(t *testing.T) {
	fx := newStreamFixture(t)
	old := streamHeartbeat
	streamHeartbeat = 50 * time.Millisecond // reach the ping and its refresh without waiting 10s
	t.Cleanup(func() { streamHeartbeat = old })
	h := fx.srv.Handler()
	base := "/api/v1/projects/" + fx.proj.ID.String()

	t.Run("presence", func(t *testing.T) {
		w, stop := serveStream(t, h, fx.tokenA, base+"/presence/stream?client=tab-1")
		w.await(t, "event: presence", ": ping") // the connect snapshot, then a heartbeat ping
		// The other member moves their pointer: a stream does not echo its own viewer's.
		post := httptest.NewRequest(http.MethodPost, base+"/cursor?client=tab-1", strings.NewReader(`{"x":1,"y":2}`))
		post.Header.Set("Authorization", "Bearer "+fx.tokenB)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, post)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("post cursor: status %d, want 204", rec.Code)
		}
		w.await(t, "event: cursor")
		stop()
		w.requireEveryWriteBounded(t)
	})

	t.Run("progress", func(t *testing.T) {
		w, stop := serveStream(t, h, fx.tokenA, base+"/roadmap/progress/stream")
		w.await(t, "event: progress", ": ping") // the connect snapshot, then a heartbeat ping
		fx.srv.progress.Publish(fx.proj.ID.String())
		w.await(t, "event: progress") // a later change
		stop()
		w.requireEveryWriteBounded(t)
	})
}
