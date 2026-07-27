package api

import (
	"strings"
	"testing"
)

// TestValidateUpdateContent covers the F10 field bound: a project update's
// content is trimmed, must be non-empty, and is capped by rune count. The
// multibyte cases prove the cap counts runes, not bytes — a 5000-CJK-rune
// note is 15 KB on the wire yet must pass, while 5001 runes must not.
func TestValidateUpdateContent(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr string
	}{
		{"trims surrounding whitespace", "  hello  ", "hello", ""},
		{"empty rejected", "", "", "content is required"},
		{"whitespace-only rejected", "   \n\t ", "", "content is required"},
		{"exactly at the rune cap passes", strings.Repeat("a", maxUpdateContentRunes), strings.Repeat("a", maxUpdateContentRunes), ""},
		{"one rune over the cap rejected", strings.Repeat("a", maxUpdateContentRunes+1), "", "content too long"},
		{"multibyte at the cap passes (rune not byte count)", strings.Repeat("世", maxUpdateContentRunes), strings.Repeat("世", maxUpdateContentRunes), ""},
		{"multibyte one over the cap rejected", strings.Repeat("世", maxUpdateContentRunes+1), "", "content too long"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, errMsg := validateUpdateContent(tc.in)
			if errMsg != tc.wantErr {
				t.Fatalf("errMsg = %q, want %q", errMsg, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("content = %q, want %q", got, tc.want)
			}
		})
	}
}
