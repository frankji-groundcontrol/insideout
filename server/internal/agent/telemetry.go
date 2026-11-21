package agent

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/llm"
)

// recordTelemetry writes one ai_run_events row — metadata only (model,
// tokens, latency, error class), never prompt/response content (plan
// §10.6). Best-effort: a telemetry write failure never fails the turn.
func (c *Coach) recordTelemetry(ctx context.Context, runID uuid.UUID, role string, started time.Time, usage llm.Usage, callErr error) {
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

// telemetryErrorClass gives a short, stable label for InsertAIRunEvent —
// metadata only, never the error's own text (which could embed prompt
// content echoed back by the provider).
func telemetryErrorClass(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, llm.ErrProviderRateLimited):
		return "rate_limited"
	case errors.Is(err, llm.ErrProviderTransient):
		return "transient"
	case errors.Is(err, llm.ErrContextLength):
		return "context_length"
	case errors.Is(err, llm.ErrProviderConfig):
		return "config"
	case errors.Is(err, llm.ErrContentRefusal):
		return "refusal"
	case errors.Is(err, llm.ErrUpstreamStall):
		return "upstream_stall"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "canceled"
	default:
		return "unknown"
	}
}
