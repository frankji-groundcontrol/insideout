package agent

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
)

// deadlineWriter is a flushing response writer that, like a real connection,
// takes a write deadline, and notes every write that runs while none is armed.
// A deadline counts only while it is still ahead and no further out than
// httpx.SSEWriteTimeout, and a flush uses it up, so a deadline armed once at
// the start of a turn does not cover the events that follow.
type deadlineWriter struct {
	header http.Header
	armed  bool
	bare   []string // first line of every write that ran with no deadline armed
}

func (d *deadlineWriter) Header() http.Header { return d.header }
func (d *deadlineWriter) WriteHeader(int)     {}

func (d *deadlineWriter) SetWriteDeadline(t time.Time) error {
	left := time.Until(t)
	d.armed = left > 0 && left <= httpx.SSEWriteTimeout
	return nil
}

func (d *deadlineWriter) Write(p []byte) (int, error) {
	if !d.armed {
		d.bare = append(d.bare, strings.SplitN(string(p), "\n", 2)[0])
	}
	return len(p), nil
}

func (d *deadlineWriter) Flush() { d.armed = false }

// REL-15: the coach stream returns its write errors, but the write itself had
// no deadline, so a client that stopped reading mid-turn pinned the handler
// (and the turn's run row) until the kernel gave up. Every event the turn
// writes must be armed with its own bounded deadline: a turn can sit minutes
// between events while the model thinks, so one deadline per stream is no use.
func TestSSEWriter_ArmsAWriteDeadlineBeforeEveryEvent(t *testing.T) {
	w := &deadlineWriter{header: http.Header{}}
	sw, ok := newSSEWriter(w)
	if !ok {
		t.Fatal("newSSEWriter refused a writer that flushes")
	}
	sw.messageStart("m1")
	sw.delta("hello")
	sw.prdUpdated("problem")
	sw.messageEnd("m1", 3)
	if len(w.bare) > 0 {
		t.Fatalf("%d coach stream write(s) ran with no write deadline armed, so a client that stopped reading would pin the turn: %q", len(w.bare), w.bare)
	}
}
