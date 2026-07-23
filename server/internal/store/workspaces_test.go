package store

import (
	"regexp"
	"testing"
)

var sixDigitCode = regexp.MustCompile(`^\d{6}$`)

func TestGenerateInviteCode_IsSixDigits(t *testing.T) {
	for i := 0; i < 50; i++ {
		code, err := generateInviteCode()
		if err != nil {
			t.Fatalf("generateInviteCode: %v", err)
		}
		if !sixDigitCode.MatchString(code) {
			t.Fatalf("generateInviteCode() = %q, want 6 digits", code)
		}
	}
}
