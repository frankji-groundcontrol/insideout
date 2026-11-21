package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/llm"
)

func TestTelemetryErrorClass(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"rate limited", llm.ErrProviderRateLimited, "rate_limited"},
		{"wrapped rate limited", errors.New("wrap: " + llm.ErrProviderRateLimited.Error()), "unknown"}, // not errors.Is-wrapped -> unknown, by design
		{"transient", llm.ErrProviderTransient, "transient"},
		{"context length", llm.ErrContextLength, "context_length"},
		{"config", llm.ErrProviderConfig, "config"},
		{"refusal", llm.ErrContentRefusal, "refusal"},
		{"canceled", context.Canceled, "canceled"},
		{"deadline", context.DeadlineExceeded, "canceled"},
		{"unknown", errors.New("boom"), "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := telemetryErrorClass(tc.err); got != tc.want {
				t.Errorf("telemetryErrorClass(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}
