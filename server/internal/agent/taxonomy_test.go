package agent

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"
)

// fakeResponse builds an *http.Response the way anthropicHTTPError expects
// to read one: status code, optional Retry-After header, and a body
// shaped like a real Anthropic error payload
// (https://docs.anthropic.com/en/api/errors).
func fakeResponse(status int, retryAfter string, body string) *http.Response {
	h := http.Header{}
	if retryAfter != "" {
		h.Set("Retry-After", retryAfter)
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     h,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

func TestAnthropicHTTPError_Classification(t *testing.T) {
	cases := []struct {
		name      string
		resp      *http.Response
		wantErr   error
		wantRetry time.Duration
	}{
		{
			name:      "429 with Retry-After",
			resp:      fakeResponse(http.StatusTooManyRequests, "12", `{"type":"error","error":{"type":"rate_limit_error","message":"Number of request tokens has exceeded your per-minute rate limit"}}`),
			wantErr:   ErrProviderRateLimited,
			wantRetry: 12 * time.Second,
		},
		{
			name:    "429 without Retry-After",
			resp:    fakeResponse(http.StatusTooManyRequests, "", `{"type":"error","error":{"type":"rate_limit_error","message":"rate limited"}}`),
			wantErr: ErrProviderRateLimited,
		},
		{
			name:    "529 overloaded",
			resp:    fakeResponse(529, "", `{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`),
			wantErr: ErrProviderTransient,
		},
		{
			name:    "500 internal",
			resp:    fakeResponse(http.StatusInternalServerError, "", `{"type":"error","error":{"type":"api_error","message":"Internal server error"}}`),
			wantErr: ErrProviderTransient,
		},
		{
			name:    "400 context length",
			resp:    fakeResponse(http.StatusBadRequest, "", `{"type":"error","error":{"type":"invalid_request_error","message":"prompt is too long: 205000 tokens > 200000 maximum context length"}}`),
			wantErr: ErrContextLength,
		},
		{
			name:    "400 other invalid request",
			resp:    fakeResponse(http.StatusBadRequest, "", `{"type":"error","error":{"type":"invalid_request_error","message":"max_tokens: field required"}}`),
			wantErr: ErrProviderConfig,
		},
		{
			name:    "401 authentication",
			resp:    fakeResponse(http.StatusUnauthorized, "", `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`),
			wantErr: ErrProviderConfig,
		},
		{
			name:    "403 permission",
			resp:    fakeResponse(http.StatusForbidden, "", `{"type":"error","error":{"type":"permission_error","message":"Your API key does not have permission"}}`),
			wantErr: ErrProviderConfig,
		},
		{
			name:    "404 model not found",
			resp:    fakeResponse(http.StatusNotFound, "", `{"type":"error","error":{"type":"not_found_error","message":"model: claude-bogus-model not found"}}`),
			wantErr: ErrProviderConfig,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := anthropicHTTPError(tc.resp)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("classified as %v, want wrapping %v", err, tc.wantErr)
			}
			if tc.wantRetry > 0 {
				got := retryAfterFrom(err)
				if got != tc.wantRetry {
					t.Fatalf("retryAfterFrom = %v, want %v", got, tc.wantRetry)
				}
			}
		})
	}
}

func TestRetryAfterFrom_DefaultsAndCaps(t *testing.T) {
	// No Retry-After header at all -> default cap.
	err := anthropicHTTPError(fakeResponse(http.StatusTooManyRequests, "", `{}`))
	if got := retryAfterFrom(err); got != 20*time.Second {
		t.Fatalf("no Retry-After: got %v, want 20s default", got)
	}

	// A huge Retry-After is capped at 20s, not honored verbatim.
	err = anthropicHTTPError(fakeResponse(http.StatusTooManyRequests, "600", `{}`))
	if got := retryAfterFrom(err); got != 20*time.Second {
		t.Fatalf("huge Retry-After: got %v, want capped at 20s", got)
	}

	// A plain error with no RetryAfter() method still gets the default.
	if got := retryAfterFrom(errors.New("no method")); got != 20*time.Second {
		t.Fatalf("plain error: got %v, want 20s default", got)
	}
}
