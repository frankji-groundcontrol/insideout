package ui

import "testing"

func TestResetTokenReadsALinkOrABareToken(t *testing.T) {
	link := "https://insideout.example/reset-password?token=abc123"
	if got := resetToken(link); got != "abc123" {
		t.Fatalf("link token = %q", got)
	}
	if got := resetToken("abc123"); got != "abc123" {
		t.Fatalf("bare token = %q", got)
	}
}
