package agent

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// Pure, so the global ai_circuit_breaker row is never touched: only failures
// that say something about the provider's health may feed the breaker.
func TestCountsAgainstCircuit(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"rate limited", fmt.Errorf("%w: anthropic returned 429", ErrProviderRateLimited), true},
		{"transient", fmt.Errorf("%w: anthropic returned 503", ErrProviderTransient), true},
		{"upstream stall", ErrUpstreamStall, true},
		{"unclassified provider error", errors.New("agent: anthropic returned 422 Unprocessable Entity: bad field"), true},
		{"context length", fmt.Errorf("%w: prompt is too long", ErrContextLength), false},
		{"provider config", fmt.Errorf("%w: anthropic returned 401", ErrProviderConfig), false},
		{"content refusal", fmt.Errorf("%w: model declined", ErrContentRefusal), false},
		{"client disconnect", fmt.Errorf("client write failed: %w", context.Canceled), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := countsAgainstCircuit(tc.err); got != tc.want {
				t.Fatalf("countsAgainstCircuit(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
