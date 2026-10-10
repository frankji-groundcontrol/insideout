package agent

import (
	"errors"
	"strings"
	"testing"
)

// responsesErrorStream is one Responses API "error" event. Unlike Anthropic's
// nested error.type, its code and message sit at the top level of the event.
func responsesErrorStream(code, msg string) string {
	return "event: error\ndata: {\"type\":\"error\",\"code\":\"" + code + "\",\"message\":\"" + msg + "\"}\n"
}

// A mid-stream error from a Responses-API provider used to go through the
// Anthropic classifier: its codes matched none of Anthropic's error types, so a
// rate limit or a server error was neither retried nor counted as a provider
// failure, and the text blamed anthropic. They must classify like their
// Anthropic counterparts (rate_limit_error, api_error) and name the provider
// that answered; a code nobody mapped stays unclassified.
func TestParseResponsesStream_ErrorEventTaxonomy(t *testing.T) {
	sentinels := []error{ErrProviderRateLimited, ErrProviderTransient, ErrContextLength, ErrProviderConfig}
	cases := []struct {
		code string
		want error // nil: no sentinel at all
	}{
		{"rate_limit_exceeded", ErrProviderRateLimited},
		{"server_error", ErrProviderTransient},
		{"weird_new_code", nil},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			_, err := parseResponsesStream(strings.NewReader(responsesErrorStream(tc.code, "upstream said no")), nil, nil)
			if err == nil {
				t.Fatal("want an error for an in-stream error event")
			}
			for _, s := range sentinels {
				if got, want := errors.Is(err, s), s == tc.want; got != want {
					t.Errorf("errors.Is(err, %v) = %v, want %v; err = %q", s, got, want, err)
				}
			}
			if strings.Contains(err.Error(), "anthropic") {
				t.Errorf("err = %q blames anthropic for a Responses-API failure", err)
			}
			if want := "agent: openai-compatible responses stream error " + tc.code + ": upstream said no"; !strings.Contains(err.Error(), want) {
				t.Errorf("err = %q, want it to contain %q", err, want)
			}
		})
	}
}
