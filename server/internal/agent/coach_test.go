package agent

import (
	"testing"
	"time"
)

func TestNew_AIConcurrencyLimits_FromEnv(t *testing.T) {
	t.Setenv("INSIDEOUT_AI_MAX_CONCURRENT", "9")
	t.Setenv("INSIDEOUT_AI_QUEUE_WAIT", "3s")

	c := New(nil, nil, nil)
	if cap(c.dispatch.permits) != 9 {
		t.Fatalf("maxConcurrent: got %d, want 9", cap(c.dispatch.permits))
	}
	if c.dispatch.queueWait != 3*time.Second {
		t.Fatalf("queueWait: got %s, want 3s", c.dispatch.queueWait)
	}
	if c.queueWait != 3*time.Second {
		t.Fatalf("coach queueWait (503 retry hint): got %s, want 3s", c.queueWait)
	}
}

func TestNew_AIConcurrencyLimits_Fallbacks(t *testing.T) {
	// Unset, unparsable, and non-positive values all fall back to the
	// package defaults.
	for _, tc := range []struct{ name, max, wait string }{
		{"unset", "", ""},
		{"invalid", "nope", "soon"},
		{"non-positive", "0", "0s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("INSIDEOUT_AI_MAX_CONCURRENT", tc.max)
			t.Setenv("INSIDEOUT_AI_QUEUE_WAIT", tc.wait)

			c := New(nil, nil, nil)
			if cap(c.dispatch.permits) != defaultMaxConcurrent {
				t.Fatalf("maxConcurrent: got %d, want default %d", cap(c.dispatch.permits), defaultMaxConcurrent)
			}
			if c.dispatch.queueWait != defaultQueueWait {
				t.Fatalf("queueWait: got %s, want default %s", c.dispatch.queueWait, defaultQueueWait)
			}
			if c.queueWait != defaultQueueWait {
				t.Fatalf("coach queueWait (503 retry hint): got %s, want default %s", c.queueWait, defaultQueueWait)
			}
		})
	}
}
