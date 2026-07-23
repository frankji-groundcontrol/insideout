package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// sseWriter frames Server-Sent Events per
// docs/plans/2026-07-20-go-rewrite/03-agents.md §4: message_start, delta,
// prd_updated, stage_changed, message_end, error.
type sseWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// newSSEWriter starts the SSE response. Returns ok=false if the
// underlying ResponseWriter doesn't support flushing (shouldn't happen
// with net/http's default server, but fails loudly rather than silently
// buffering the whole reply if it ever does).
func newSSEWriter(w http.ResponseWriter) (*sseWriter, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, false
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	sw := &sseWriter{w: w, flusher: flusher}
	return sw, true
}

func (s *sseWriter) event(name string, data any) {
	b, err := json.Marshal(data)
	if err != nil {
		b = []byte(`{}`)
	}
	fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", name, b)
	s.flusher.Flush()
}

func (s *sseWriter) messageStart(id string) { s.event("message_start", map[string]string{"id": id}) }
func (s *sseWriter) delta(text string)      { s.event("delta", map[string]string{"text": text}) }
func (s *sseWriter) prdUpdated(section string) {
	s.event("prd_updated", map[string]string{"section": section})
}
func (s *sseWriter) factRecorded(f Fact) {
	s.event("fact_recorded", map[string]string{"id": f.ID, "kind": f.Kind, "text": f.Text, "status": f.Status})
}
func (s *sseWriter) criticFindings(findings []CriticFinding) {
	if len(findings) == 0 {
		return
	}
	s.event("critic_findings", map[string]any{"findings": findings})
}
func (s *sseWriter) stageChanged(stage string) {
	s.event("stage_changed", map[string]string{"stage": stage})
}
func (s *sseWriter) messageEnd(id string, tokens int) {
	s.event("message_end", map[string]any{"id": id, "tokens": tokens})
}
func (s *sseWriter) errorEvent(message, code string) {
	s.event("error", map[string]string{"error": message, "code": code})
}
