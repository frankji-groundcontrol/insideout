package main

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"golang.org/x/term"
)

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

// swapStdin points the shared prompt reader at a scripted input and
// returns a restore func (tests in a package run sequentially).
func swapStdin(script string) func() {
	old := stdin
	stdin = bufio.NewReader(strings.NewReader(script))
	return func() { stdin = old }
}

// readSecret on a pipe must keep reading through the shared buffered
// reader — both the single piped secret and the back-to-back pair that
// once exposed the per-call-reader buffering trap (2026-08-24).
func TestReadSecretPiped(t *testing.T) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		t.Skip("stdin is a terminal; the non-TTY path is not exercised here")
	}
	restore := swapStdin("link-token\nnewpassword\n")
	defer restore()

	link, err := readSecret("reset link or token: ")
	if err != nil {
		t.Fatalf("readSecret(first) returned error: %v", err)
	}
	if link != "link-token" {
		t.Errorf("readSecret(first) = %q, want %q", link, "link-token")
	}

	pw, err := readSecret("new password: ")
	if err != nil {
		t.Fatalf("readSecret(second) returned error: %v", err)
	}
	if pw != "newpassword" {
		t.Errorf("readSecret(second) = %q, want %q (second answer lost to buffer-ahead?)", pw, "newpassword")
	}
}

// On a real terminal the hidden path must not demand an extra Enter
// per secret: term.ReadPassword consumes the trailing newline itself,
// so any read after it blocks for another Enter — and echoes it,
// echo already being restored. Needs a pty; normal `go test` skips:
//
//	go test -c -o /tmp/insideout-auth.test ./cmd/insideout
//	printf 'pw1\npw2\nhello\n\n' | script -q /dev/null /tmp/insideout-auth.test -test.v -test.run TestReadSecretTerminal
func TestReadSecretTerminal(t *testing.T) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		t.Skip("stdin is not a terminal; drive with a pty (see comment)")
	}
	s1, err := readSecret("s1: ")
	if err != nil || s1 != "pw1" {
		t.Fatalf("first readSecret = %q, %v; want pw1", s1, err)
	}
	s2, err := readSecret("s2: ")
	if err != nil || s2 != "pw2" {
		t.Fatalf("second readSecret = %q, %v; want pw2 (extra Enter swallowed it?)", s2, err)
	}
	if next := prompt("next: "); next != "hello" {
		t.Fatalf("prompt after secrets = %q, want hello", next)
	}
}
