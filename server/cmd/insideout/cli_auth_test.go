package main

import "testing"

// Emailed flows paste as URLs; the CLI must accept both shapes.
func TestTokenFromLink(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://app.example.com/magic-link?token=abc123", "abc123"},
		{"https://app.example.com/reset-password?token=abc123&x=1", "abc123"},
		{"abc123", "abc123"},
		{"  abc123  ", "abc123"},
		{"https://app.example.com/magic-link?token=", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := tokenFromLink(tc.in); got != tc.want {
			t.Errorf("tokenFromLink(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
