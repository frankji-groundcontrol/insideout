package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// This file is the provider error taxonomy: classifying a failed HTTP
// response or stream read into the sentinels in llm.go, and the retry
// policy that follows from each — see
// docs/plans/2026-07-21-prd-agent-harness/plan.md §5.2.

// idleStreamTimeout bounds how long the stream may go silent (no SSE
// bytes at all) before it's canceled — extended-thinking responses have
// long pauses, so this is generous, not a per-token deadline.
const idleStreamTimeout = 90 * time.Second

// streamChat retries transient failures once — 429 (honoring Retry-After,
// capped) and 5xx/connect refusal (fixed short backoff) — with jitter.
// Every other classified error (context length, config, refusal,
// context.Canceled) is never retried here.
func (a *AnthropicStreamer) streamChat(ctx context.Context, system string, msgs []Message, tools []Tool, forceTool string, onDelta func(string)) (Turn, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		turn, err := a.doStreamChat(ctx, system, msgs, tools, forceTool, onDelta)
		if err == nil {
			return turn, nil
		}
		lastErr = err

		var wait time.Duration
		switch {
		case errors.Is(err, ErrProviderRateLimited):
			wait = retryAfterFrom(err)
		case errors.Is(err, ErrProviderTransient):
			wait = 2 * time.Second
		default:
			return Turn{}, err // not retryable
		}
		if attempt == 1 {
			break
		}
		wait += time.Duration(rand.Int64N(int64(500 * time.Millisecond)))
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return Turn{}, ctx.Err()
		}
	}
	return Turn{}, lastErr
}

// idleWatchdog cancels the request if tick() isn't called for at least
// timeout — used to bound how long an SSE stream may go silent (a hung
// upstream would otherwise pin the connection, and the user's stream,
// forever, since resp.Body.Read never returns on its own).
func idleWatchdog(cancel context.CancelFunc, timeout time.Duration) (tick func(), stop func()) {
	reset := make(chan struct{}, 1)
	done := make(chan struct{})
	go func() {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		for {
			select {
			case <-reset:
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(timeout)
			case <-timer.C:
				cancel()
				return
			case <-done:
				return
			}
		}
	}()
	tick = func() {
		select {
		case reset <- struct{}{}:
		default:
		}
	}
	stop = func() { close(done) }
	return tick, stop
}

func retryAfterFrom(err error) time.Duration {
	var withRetryAfter interface{ RetryAfter() time.Duration }
	if errors.As(err, &withRetryAfter) {
		if d := withRetryAfter.RetryAfter(); d > 0 && d <= 20*time.Second {
			return d
		}
	}
	return 20 * time.Second
}

// rateLimitError carries the provider's Retry-After so streamChat can
// honor it without anthropicHTTPError needing to know about retries.
type rateLimitError struct {
	msg        string
	retryAfter time.Duration
}

func (e *rateLimitError) Error() string             { return e.msg }
func (e *rateLimitError) Unwrap() error             { return ErrProviderRateLimited }
func (e *rateLimitError) RetryAfter() time.Duration { return e.retryAfter }

// anthropicHTTPError classifies a non-200 response into one of the
// sentinels in llm.go. This is the taxonomy's single decision point for
// HTTP-level failures.
func anthropicHTTPError(resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var parsed anthropicErrorBody
	msg := string(raw)
	if json.Unmarshal(raw, &parsed) == nil && parsed.Error.Message != "" {
		msg = fmt.Sprintf("%s: %s", parsed.Error.Type, parsed.Error.Message)
	}
	full := fmt.Sprintf("agent: anthropic returned %s: %s", resp.Status, msg)

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		var wait time.Duration
		if secs, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && secs > 0 {
			wait = time.Duration(secs) * time.Second
		}
		return &rateLimitError{msg: full, retryAfter: wait}
	case resp.StatusCode == 529 || resp.StatusCode >= 500:
		return fmt.Errorf("%w: %s", ErrProviderTransient, full)
	case resp.StatusCode == http.StatusBadRequest && strings.Contains(strings.ToLower(msg), "context"):
		return fmt.Errorf("%w: %s", ErrContextLength, full)
	case resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized ||
		resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrProviderConfig, full)
	default:
		return errors.New(full)
	}
}

// telemetryErrorClass gives a short, stable label for InsertAIRunEvent —
// metadata only, never the error's own text (which could embed prompt
// content echoed back by the provider).
func telemetryErrorClass(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrProviderRateLimited):
		return "rate_limited"
	case errors.Is(err, ErrProviderTransient):
		return "transient"
	case errors.Is(err, ErrContextLength):
		return "context_length"
	case errors.Is(err, ErrProviderConfig):
		return "config"
	case errors.Is(err, ErrContentRefusal):
		return "refusal"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "canceled"
	default:
		return "unknown"
	}
}

// classifyStreamReadErr distinguishes a caller-initiated cancellation
// (client disconnect or the idle watchdog) from a genuine transient read
// failure, since coach.go's error taxonomy treats them very differently
// (no circuit failure vs. one).
func classifyStreamReadErr(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return fmt.Errorf("%w: read anthropic stream: %s", ErrProviderTransient, err.Error())
}
