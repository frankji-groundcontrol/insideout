package agent

import (
	"context"
	"errors"
	"testing"
)

func TestTelemetryErrorClass(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"rate limited", ErrProviderRateLimited, "rate_limited"},
		{"wrapped rate limited", errors.New("wrap: " + ErrProviderRateLimited.Error()), "unknown"}, // not errors.Is-wrapped -> unknown, by design
		{"transient", ErrProviderTransient, "transient"},
		{"context length", ErrContextLength, "context_length"},
		{"config", ErrProviderConfig, "config"},
		{"refusal", ErrContentRefusal, "refusal"},
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
