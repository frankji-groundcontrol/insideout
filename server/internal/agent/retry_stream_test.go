package agent

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// TestRetryStreamChat_NoRetryAfterDelta: once a delta has been emitted,
// a transient failure must surface immediately — retrying would replay
// text the client has already seen.
func TestRetryStreamChat_NoRetryAfterDelta(t *testing.T) {
	streamed := 0
	calls := 0
	do := func(ctx context.Context) (Turn, error) {
		calls++
		streamed++ // simulate the delta reaching the client
		return Turn{}, fmt.Errorf("%w: stream died mid-turn", ErrProviderTransient)
	}

	_, err := retryStreamChat(context.Background(), func() bool { return streamed == 0 }, do)
	if !errors.Is(err, ErrProviderTransient) {
		t.Fatalf("err = %v, want wrapping ErrProviderTransient", err)
	}
	if calls != 1 {
		t.Fatalf("do called %d times, want exactly 1 (no retry once a delta went out)", calls)
	}
}

// TestRetryStreamChat_RetryWhenNothingStreamed: a transient failure with
// zero deltas emitted is retried once and may succeed.
func TestRetryStreamChat_RetryWhenNothingStreamed(t *testing.T) {
	streamed := 0
	calls := 0
	do := func(ctx context.Context) (Turn, error) {
		calls++
		if calls == 1 {
			// A 1ms Retry-After is honored verbatim (under the 20s cap),
			// keeping the backoff negligible; the jitter still applies.
			return Turn{}, &rateLimitError{msg: "rate limited", retryAfter: time.Millisecond}
		}
		return Turn{Text: "ok"}, nil
	}

	turn, err := retryStreamChat(context.Background(), func() bool { return streamed == 0 }, do)
	if err != nil {
		t.Fatalf("err = %v, want nil after successful retry", err)
	}
	if turn.Text != "ok" {
		t.Fatalf("turn.Text = %q, want %q", turn.Text, "ok")
	}
	if calls != 2 {
		t.Fatalf("do called %d times, want 2 (one retry after the rate limit)", calls)
	}
}
