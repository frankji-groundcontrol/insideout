package agent

import (
	"errors"
	"net/http"
	"testing"
)

// failAfterHeadersWriter supports flushing but fails every Write — like a
// client that vanished after the SSE response started.
type failAfterHeadersWriter struct {
	header http.Header
}

func (f *failAfterHeadersWriter) Header() http.Header        { return f.header }
func (f *failAfterHeadersWriter) Write([]byte) (int, error)  { return 0, errors.New("broken pipe") }
func (f *failAfterHeadersWriter) WriteHeader(statusCode int) {}
func (f *failAfterHeadersWriter) Flush()                     {}

func TestSSE_Event_PropagatesWriteError(t *testing.T) {
	sw, ok := newSSEWriter(&failAfterHeadersWriter{header: http.Header{}})
	if !ok {
		t.Fatal("failAfterHeadersWriter must support flushing")
	}

	if err := sw.messageStart("m1", "claude-test"); err == nil {
		t.Fatal("messageStart must propagate the write error")
	}
	if err := sw.delta("hi"); err == nil {
		t.Fatal("delta must propagate the write error")
	}
	if err := sw.prdUpdated("problem"); err == nil {
		t.Fatal("prdUpdated must propagate the write error")
	}
	if err := sw.factRecorded(Fact{ID: "f1"}); err == nil {
		t.Fatal("factRecorded must propagate the write error")
	}
	if err := sw.criticFindings([]CriticFinding{{ID: "c1"}}); err == nil {
		t.Fatal("criticFindings must propagate the write error")
	}
}

func TestSSE_CriticFindings_EmptyIsNilError(t *testing.T) {
	sw, ok := newSSEWriter(&failAfterHeadersWriter{header: http.Header{}})
	if !ok {
		t.Fatal("failAfterHeadersWriter must support flushing")
	}
	if err := sw.criticFindings(nil); err != nil {
		t.Fatalf("criticFindings with no findings must not touch the writer, got %v", err)
	}
}
