package agent

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// recordTelemetry writes one ai_run_events row — metadata only (model,
// tokens, latency, error class), never prompt/response content (plan
// §10.6). Best-effort: a telemetry write failure never fails the turn.
func (c *Coach) recordTelemetry(ctx context.Context, runID uuid.UUID, role string, started time.Time, usage Usage, callErr error) {
	latencyMs := int(time.Since(started).Milliseconds())
	eventType := role
	if class := telemetryErrorClass(callErr); class != "" {
		eventType = role + "_error_" + class
	}
	inputTokens, outputTokens := usage.InputTokens, usage.OutputTokens
	// Detached so a telemetry write survives the client leaving, but bounded so
	// a wedged write can't hang forever (F11).
	persistCtx, cancel := detachedContext(ctx)
	defer cancel()
	if err := c.store.InsertAIRunEvent(persistCtx, runID, eventType, c.streamer.Model(), &inputTokens, &outputTokens, &latencyMs, nil, nil); err != nil {
		c.log.Error("insert ai run event", "error", err)
	}
}
