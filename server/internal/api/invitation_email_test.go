package api

import (
	"strings"
	"testing"
	"time"
)

// FULL-01: the invitation handler passed int(ttl.Hours()) into a template
// that hardcodes "minutes", so the default 7-day invite read "expires in
// 168 minutes" in HTML while the text body said 7 day(s). Both bodies must
// state the same lifetime for a day-scale TTL.
func TestInvitationEmailExpiryCopyAgreesAboveADay(t *testing.T) {
	_, text, html := invitationEmail("http://app.test/register?invite=x", 7*24*time.Hour)
	if !strings.Contains(text, "expires in 7 day(s)") {
		t.Fatalf("text body does not state the day-scale lifetime: %q", text)
	}
	if strings.Contains(html, "minutes") {
		t.Errorf("HTML footer states minutes for a day-scale link: %s", html)
	}
	if !strings.Contains(html, "expires in 7 day(s)") {
		t.Errorf("HTML expiry copy disagrees with the text body: %s", html)
	}
}
