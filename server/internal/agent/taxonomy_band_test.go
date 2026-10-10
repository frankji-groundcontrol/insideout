package agent

import (
	"errors"
	"net/http"
	"testing"
)

// AIDEV-03: the residual 4xx band (402 payment-required, 408, 413, 422)
// fell through providerHTTPError unsentineled, and countsAgainstCircuit
// counts every unclassified failure — so a lapsed operator credit (402 on
// every call) opened the one global breaker for all users. These statuses
// are billing/payload problems, not provider load: they must classify as
// ErrProviderConfig and stay off the breaker, like 401/403/404 already do.
func TestProviderHTTPError_Residual4xxBandIsConfigNotLoad(t *testing.T) {
	for _, status := range []int{
		http.StatusPaymentRequired,       // 402: quota/billing exhausted
		http.StatusRequestTimeout,        // 408
		http.StatusRequestEntityTooLarge, // 413
		http.StatusUnprocessableEntity,   // 422
	} {
		err := providerHTTPError("anthropic", fakeResponse(status, "", `{"error":{"type":"billing_error","message":"credit exhausted"}}`))
		if !errors.Is(err, ErrProviderConfig) {
			t.Errorf("%d classified as %v, want wrapping ErrProviderConfig", status, err)
			continue
		}
		if countsAgainstCircuit(err) {
			t.Errorf("a %d turn would count against the circuit breaker: %v", status, err)
		}
	}
}

// The same fall-through exists in-stream: an unknown Responses error code
// stayed unclassified and fed the breaker. Known load signals
// (rate_limit_exceeded, server_error) keep counting; everything else is a
// configuration problem on our side of the wire, not load.
func TestClassifyResponsesStreamError_UnknownCodeIsNotCircuitLoad(t *testing.T) {
	err := classifyResponsesStreamError("weird_new_code", "upstream said no")
	if !errors.Is(err, ErrProviderConfig) {
		t.Fatalf("err = %v, want wrapping ErrProviderConfig", err)
	}
	if countsAgainstCircuit(err) {
		t.Fatalf("an unknown in-stream code would count against the breaker: %v", err)
	}
	if err := classifyResponsesStreamError("server_error", "boom"); !countsAgainstCircuit(err) {
		t.Fatalf("server_error stopped counting as load: %v", err)
	}
	if err := classifyResponsesStreamError("rate_limit_exceeded", "slow down"); !countsAgainstCircuit(err) {
		t.Fatalf("rate_limit_exceeded stopped counting as load: %v", err)
	}
}
