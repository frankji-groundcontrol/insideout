package agent

import (
	"errors"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/google/uuid"
)

// preflight is the first gate an LLM call passes, before any work: the per-user
// rate limit, then the circuit breaker. On refusal the response is already
// written as plain JSON (the preserved 429/503 contract, see
// docs/plans/2026-07-20-go-rewrite/03-agents.md §4) and it returns false.
func (c *Coach) preflight(w http.ResponseWriter, r *http.Request, userID uuid.UUID) bool {
	ctx := r.Context()

	rl, err := c.store.CheckRateLimit(ctx, userID)
	if err != nil {
		c.log.Error("check rate limit", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return false
	}
	if !rl.Allowed {
		httpx.WriteError(w, http.StatusTooManyRequests, "Rate limit exceeded", "APP_THROTTLE", map[string]interface{}{
			"retry_after_seconds": rl.RetryAfterSeconds, "current_count": rl.CurrentCount, "max_requests": rl.MaxRequests,
		})
		return false
	}

	circuit, err := c.store.CheckCircuit(ctx)
	if err != nil {
		c.log.Error("check circuit", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return false
	}
	if !circuit.Allowed {
		httpx.WriteError(w, http.StatusServiceUnavailable, "AI service temporarily unavailable", "CIRCUIT_OPEN", map[string]interface{}{
			"retry_after_seconds": circuit.RetryAfterSeconds, "circuit_state": circuit.State,
		})
		return false
	}
	return true
}

// acquire takes one of the global concurrent-provider-call permits, answering
// 503 when none frees up within queueWait.
func (c *Coach) acquire(w http.ResponseWriter) (release func(), waited, ok bool) {
	release, waited, ok = c.dispatch.acquirePermit()
	if !ok {
		httpx.WriteError(w, http.StatusServiceUnavailable, "AI service is busy, try again shortly", "CIRCUIT_OPEN", map[string]interface{}{
			"retry_after_seconds": int(c.queueWait.Seconds()),
		})
	}
	return release, waited, ok
}

// Admit gates an LLM call that is not a coach turn (the roadmap planner) with
// what a coach turn passes: the per-user rate limit, the circuit breaker and a
// dispatch permit. It also opens an ai_runs row, the limiter's counting source:
// without one the call would pass the check yet never count toward it. On
// refusal the response is already written and ok is false; otherwise the caller
// must call done once with the call's error (nil on success).
//
// ponytail: a caller that panics before done leaks its permit. The planners do
// not panic; defer done in the caller if that ever changes.
func (c *Coach) Admit(w http.ResponseWriter, r *http.Request, workspaceID, userID uuid.UUID, label string) (done func(error), ok bool) {
	if !c.preflight(w, r, userID) {
		return nil, false
	}
	release, _, ok := c.acquire(w)
	if !ok {
		return nil, false
	}
	run, err := c.store.CreateAIRun(r.Context(), workspaceID, nil, userID, label)
	if err != nil {
		release()
		c.log.Error("create ai run", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return nil, false
	}
	return func(callErr error) {
		defer release()
		ctx, cancel := detachedContext(r.Context())
		defer cancel()
		if callErr == nil {
			_ = c.store.MarkAIRunSucceeded(ctx, userID, run.ID, "")
			_ = c.store.RecordCircuitResult(ctx, true)
			return
		}
		_ = c.store.MarkAIRunFailed(ctx, userID, run.ID, callErr.Error())
		// Only a provider-health signal feeds the breaker; a plan the model
		// botched, or a client that went away, is not provider load.
		if errors.Is(callErr, ErrProviderRateLimited) || errors.Is(callErr, ErrProviderTransient) || errors.Is(callErr, ErrUpstreamStall) {
			_ = c.store.RecordCircuitResult(ctx, false)
		}
	}, true
}
