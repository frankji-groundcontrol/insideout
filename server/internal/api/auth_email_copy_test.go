package api

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// FULL-08: copy pins for all four auth email templates (invitation, magic
// link, password reset, login code), captured from the dev-mode sender the
// way a hosted provider would see them. Template regressions like FULL-01
// sailed through every gate because nothing asserted the copy; these pin
// subject, link shape, and the text/HTML expiry agreement per template.

// oneTimeTokenRe matches the base64url one-time tokens the link emails
// carry (43 chars, unpadded).
var oneTimeTokenRe = regexp.MustCompile(`^[A-Za-z0-9_-]{32,}$`)

// footerExpiry pulls the lifetime phrase out of an HTML template's
// "… works once and expires in …." footer line (the intros say "expires"
// too, but never end the phrase with a period on that line).
var footerExpiry = regexp.MustCompile(`works once and expires in ([^.\n]+)\.`)

// wantLinkToken fails unless the mailed text carries a standalone link
// line prefix + token, and returns the token.
func wantLinkToken(t *testing.T, mail sentMail, prefix string) string {
	t.Helper()
	token := tokenFrom(t, mail, prefix)
	if !oneTimeTokenRe.MatchString(token) {
		t.Fatalf("emailed link token %q does not look like a one-time token (prefix %q)", token, prefix)
	}
	return token
}

// wantExpiryAgreement fails unless the HTML footer's lifetime phrase also
// appears in the text body — the invariant FULL-01 broke.
func wantExpiryAgreement(t *testing.T, what, text, html string) {
	t.Helper()
	m := footerExpiry.FindStringSubmatch(html)
	if m == nil {
		t.Fatalf("%s: HTML has no \"expires in …\" footer:\n%s", what, html)
	}
	if !strings.Contains(text, m[1]) {
		t.Fatalf("%s: HTML footer says %q but the text body never states that lifetime:\ntext: %s\nhtml: %s",
			what, m[1], text, html)
	}
}

func TestMagicLinkEmailCopy(t *testing.T) {
	m := newMailWorld(t)
	email := "copy-magic-" + uuid.NewString() + "@test.local"
	if _, err := m.store.CreateUser(context.Background(), email, "x", "copy-magic"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if w := postJSON(m.Server, "/api/v1/auth/magic-link", `{"email":"`+email+`"}`); w.Code != http.StatusOK {
		t.Fatalf("magic-link request: status %d, body %s", w.Code, w.Body.String())
	}
	mail := m.lastMail(t)
	if mail.Subject != "Your InsideOut sign-in link" {
		t.Errorf("subject = %q, want %q", mail.Subject, "Your InsideOut sign-in link")
	}
	if mail.To != email {
		t.Errorf("mailed to %q, want %q", mail.To, email)
	}
	if !strings.Contains(mail.Text, "valid for 15 minutes") {
		t.Errorf("text intro lost the 15-minute lifetime:\n%s", mail.Text)
	}
	wantLinkToken(t, mail, "http://app.test/magic-link?token=")
	wantExpiryAgreement(t, "magic link", mail.Text,
		simpleEmailHTML("Sign in to InsideOut", "Sign in to InsideOut with this link (valid for 15 minutes):",
			"http://app.test/magic-link?token=x", linkExpiryPhrase(magicLinkTTL)))
}

func TestPasswordResetEmailCopy(t *testing.T) {
	m := newMailWorld(t)
	email := "copy-reset-" + uuid.NewString() + "@test.local"
	if _, err := m.store.CreateUser(context.Background(), email, "x", "copy-reset"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if w := postJSON(m.Server, "/api/v1/auth/forgot-password", `{"email":"`+email+`"}`); w.Code != http.StatusOK {
		t.Fatalf("forgot-password request: status %d, body %s", w.Code, w.Body.String())
	}
	mail := m.lastMail(t)
	if mail.Subject != "Reset your InsideOut password" {
		t.Errorf("subject = %q, want %q", mail.Subject, "Reset your InsideOut password")
	}
	if mail.To != email {
		t.Errorf("mailed to %q, want %q", mail.To, email)
	}
	if !strings.Contains(mail.Text, "valid for 60 minutes") {
		t.Errorf("text intro lost the 60-minute lifetime:\n%s", mail.Text)
	}
	wantLinkToken(t, mail, "http://app.test/reset-password?token=")
	wantExpiryAgreement(t, "password reset", mail.Text,
		simpleEmailHTML("Reset your InsideOut password", "Set a new InsideOut password with this link (valid for 60 minutes):",
			"http://app.test/reset-password?token=x", linkExpiryPhrase(passwordResetTTL)))
}

func TestLoginCodeEmailCopy(t *testing.T) {
	m := newMailWorld(t)
	email := "copy-code-" + uuid.NewString() + "@test.local"
	if _, err := m.store.CreateUser(context.Background(), email, "x", "copy-code"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if w := postJSON(m.Server, "/api/v1/auth/login-code", `{"email":"`+email+`"}`); w.Code != http.StatusOK {
		t.Fatalf("login-code request: status %d, body %s", w.Code, w.Body.String())
	}
	mail := m.lastMail(t)
	if mail.Subject != "Your InsideOut sign-in code" {
		t.Errorf("subject = %q, want %q", mail.Subject, "Your InsideOut sign-in code")
	}
	if mail.To != email {
		t.Errorf("mailed to %q, want %q", mail.To, email)
	}
	match := loginCodeInText.FindStringSubmatch(mail.Text)
	if match == nil {
		t.Fatalf("text lost the 6-digit code:\n%s", mail.Text)
	}
	wantExpiryAgreement(t, "login code", mail.Text, codeEmailHTML(match[1], int(loginCodeTTL.Minutes())))
}

func TestInvitationEmailCopy(t *testing.T) {
	m := newMailWorld(t)
	email := "copy-invitee-" + uuid.NewString() + "@test.local"
	if w := postJSONBearer(m.Server, m.admin.token, "/api/v1/admin/invitations", `{"email":"`+email+`"}`); w.Code != http.StatusCreated {
		t.Fatalf("create invitation: status %d, body %s", w.Code, w.Body.String())
	}
	mail := m.lastMail(t)
	if mail.Subject != "Your InsideOut invitation" {
		t.Errorf("subject = %q, want %q", mail.Subject, "Your InsideOut invitation")
	}
	if mail.To != email {
		t.Errorf("mailed to %q, want %q", mail.To, email)
	}
	code := wantLinkToken(t, mail, "http://app.test/register?invite=")
	// Both bodies of the default 7-day invitation state the same lifetime
	// (FULL-01's agreement, pinned here end to end through the route).
	_, text, html := invitationEmail("http://app.test/register?invite="+code, defaultInviteTTL)
	if !strings.Contains(mail.Text, "expires in 7 day(s)") {
		t.Errorf("mailed text lost the day-scale lifetime:\n%s", mail.Text)
	}
	wantExpiryAgreement(t, "invitation", text, html)
}
