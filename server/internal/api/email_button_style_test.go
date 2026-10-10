package api

import (
	"strings"
	"testing"
)

// FULL-06: DESIGN.md's Cobalt-Neo lock supersedes the old green rounded
// button. The email button is the first InsideOut-branded artifact a new
// user sees, so it must be cobalt-500 #3B5BFF with squared 0px corners.
func TestSimpleEmailHTMLButtonIsCobaltSquared(t *testing.T) {
	html := simpleEmailHTML("Sign in to InsideOut", "intro", "http://app.test/magic-link?token=x", "15 minutes")
	if !strings.Contains(html, "background:#3B5BFF") {
		t.Errorf("button background is not cobalt-500 #3B5BFF:\n%s", html)
	}
	if !strings.Contains(html, "border-radius:0px") {
		t.Errorf("button is not squared (border-radius:0px):\n%s", html)
	}
	if strings.Contains(html, "#1f6f54") || strings.Contains(html, "border-radius:8px") {
		t.Errorf("pre-Cobalt green/rounded styles remain:\n%s", html)
	}
}
