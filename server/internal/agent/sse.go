package agent

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
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

// event writes one framed SSE event and returns the write error, so
// callers can abort the turn when the client is gone mid-stream.
func (s *sseWriter) event(name string, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		b = []byte(`{}`)
	}
	// A turn can idle for minutes between events, so each write gets its own
	// deadline; one armed at the start would expire before the next event.
	httpx.ArmWriteDeadline(s.w, httpx.SSEWriteTimeout)
	if _, err := fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", name, b); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}

// messageStart opens the turn. model identifies the streamer; the template
// (offline dev-mode) streamer is flagged offline so clients can disclose that
// the reply is canned. Both fields are additive — older clients ignore them.
func (s *sseWriter) messageStart(id, model string) error {
	return s.event("message_start", map[string]any{
		"id":      id,
		"model":   model,
		"offline": model == templateModelID,
	})
}
func (s *sseWriter) delta(text string) error {
	return s.event("delta", map[string]string{"text": text})
}
func (s *sseWriter) prdUpdated(section string) error {
	return s.event("prd_updated", map[string]string{"section": section})
}
func (s *sseWriter) factRecorded(f Fact) error {
	return s.event("fact_recorded", map[string]string{"id": f.ID, "kind": f.Kind, "text": f.Text, "status": f.Status})
}
func (s *sseWriter) criticFindings(findings []CriticFinding) error {
	if len(findings) == 0 {
		return nil
	}
	return s.event("critic_findings", map[string]any{"findings": findings})
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
