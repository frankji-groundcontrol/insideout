package agent

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// failTurn classifies loopErr per the taxonomy (§5.2) and reports it —
// SSE error code, whether it counts against the circuit breaker — then
// persists the terminal ai_runs state. Cleanup runs on a context
// detached from the (possibly already-canceled) request context, same
// reasoning as the HandleMessage defer.
func (c *Coach) failTurn(ctx context.Context, userID uuid.UUID, runID uuid.UUID, loopErr error, sse *sseWriter) {
	cleanupCtx, cancel := detachedContext(ctx)
	defer cancel()
	_ = c.store.MarkAIRunFailed(cleanupCtx, userID, runID, loopErr.Error())

	switch {
	case errors.Is(loopErr, ErrProviderRateLimited):
		_ = c.store.RecordCircuitResult(cleanupCtx, false)
		sse.errorEvent("AI provider is rate limited", "ANTHROPIC_RATE_LIMIT")
	case errors.Is(loopErr, ErrProviderTransient):
		_ = c.store.RecordCircuitResult(cleanupCtx, false)
		c.log.Error("agent loop: provider transient error", "error", loopErr)
		sse.errorEvent("internal error", "")
	case errors.Is(loopErr, ErrContextLength):
		sse.errorEvent("conversation too long — snapshot and start a fresh session", "CONTEXT_LENGTH")
	case errors.Is(loopErr, ErrProviderConfig):
		// The provider's own text (up to 4 KiB of its response body) stays in
		// the log and ai_runs; the wire carries a fixed message like every
		// other failure class.
		c.log.Error("agent loop: provider configuration error", "error", loopErr)
		sse.errorEvent("AI provider is misconfigured - contact the operator", "PROVIDER_CONFIG")
	case errors.Is(loopErr, ErrContentRefusal):
		sse.errorEvent("the model declined to respond", "CONTENT_REFUSAL")
	case errors.Is(loopErr, ErrUpstreamStall):
		// The provider went silent and the idle watchdog killed the stream
		// while the client was still connected — a hung upstream, not a
		// disconnect. Counts against the circuit breaker, and the still-there
		// user gets an honest error instead of a silently dead bubble.
		_ = c.store.RecordCircuitResult(cleanupCtx, false)
		c.log.Error("agent loop: upstream stream stalled (idle watchdog)", "error", loopErr)
		sse.errorEvent("AI provider stopped responding — try again", "UPSTREAM_STALL")
	case errors.Is(loopErr, context.Canceled):
		// Client disconnected. No circuit result — routine, not provider health.
	default:
		_ = c.store.RecordCircuitResult(cleanupCtx, false)
		c.log.Error("agent loop", "error", loopErr)
		sse.errorEvent("internal error", "")
	}
}
