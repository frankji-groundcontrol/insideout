package api

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// R1-12: firstLine cut at 120 bytes, which lands inside a multibyte character
// for any non-ASCII commit message or PR title (PostgreSQL then rejects the
// invalid UTF-8 detail), and returned a first line of any length whenever the
// text had a newline. It caps at 120 characters, and only ever between them.
func TestFirstLineCapsAtWholeCharacters(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"short and plain", "fix: the thing", "fix: the thing"},
		{"body after the first line", "subject\n\nbody text", "subject"},
		{"empty", "", ""},
		{"exactly the cap", strings.Repeat("x", 120), strings.Repeat("x", 120)},
		{"ascii over the cap, no newline", strings.Repeat("x", 200), strings.Repeat("x", 120)},
		{"ascii over the cap, then a body", strings.Repeat("x", 200) + "\nrest", strings.Repeat("x", 120)},
		{"cjk over the cap, no newline", "a" + strings.Repeat("日", 100), "a" + strings.Repeat("日", 100)},
		{"cjk well over the cap", strings.Repeat("日", 200), strings.Repeat("日", 120)},
		{"cjk over the cap, then a body", strings.Repeat("日", 200) + "\nrest", strings.Repeat("日", 120)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := firstLine(tc.in)
			if !utf8.ValidString(got) {
				t.Errorf("firstLine = %q: not valid UTF-8, a character was cut in two", got)
			}
			if n := utf8.RuneCountInString(got); n > 120 {
				t.Errorf("firstLine returned %d characters, want at most 120", n)
			}
			if got != tc.want {
				t.Errorf("firstLine = %d characters %q, want %d characters %q", utf8.RuneCountInString(got), got, utf8.RuneCountInString(tc.want), tc.want)
			}
		})
	}
}
