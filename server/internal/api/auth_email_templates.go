package api

import (
	"fmt"
	"strings"
	"time"
)

// codeEmailHTML renders the code email — the digits get display prominence
// a link doesn't need.
func codeEmailHTML(code string, ttlMinutes int) string {
	return fmt.Sprintf(`<!doctype html>
<html><body style="font-family:system-ui,-apple-system,sans-serif;line-height:1.6;color:#1a1a1a;">
<h2 style="margin-bottom:4px;">Your sign-in code</h2>
<p>Enter this code to sign in to InsideOut:</p>
<p style="font-size:32px;font-weight:700;letter-spacing:8px;margin:16px 0;">%s</p>
<p style="color:#666666;font-size:13px;">This code works once and expires in %d minutes. If you didn't request it, you can ignore this email.</p>
</body></html>`, code, ttlMinutes)
}

// simpleEmailHTML renders the minimal inline template both auth emails
// share — no template files, same inline-string approach as docmind.
// expires is the pre-rendered lifetime phrase (e.g. "15 minutes",
// "7 day(s)") so callers cannot feed one unit into the other's template.
func simpleEmailHTML(heading, intro, link, expires string) string {
	return fmt.Sprintf(`<!doctype html>
<html><body style="font-family:system-ui,-apple-system,sans-serif;line-height:1.6;color:#1a1a1a;">
<h2 style="margin-bottom:4px;">%s</h2>
<p>%s</p>
<p><a href="%s" style="display:inline-block;padding:12px 24px;background:#3B5BFF;color:#ffffff;text-decoration:none;border-radius:0px;">Continue</a></p>
<p style="color:#666666;font-size:13px;">This link works once and expires in %s. If you didn't request it, you can ignore this email.</p>
</body></html>`, heading, intro, link, expires)
}

// linkExpiryPhrase phrases a link's lifetime for the shared email footer:
// whole minutes under a day — the magic-link and reset TTLs are
// minute-scale and their intros already say "15/60 minutes" — and
// humanTTL's day/hour units from a day up, where a raw minute count
// (10080) reads as a far shorter deadline than it is.
func linkExpiryPhrase(ttl time.Duration) string {
	if ttl < 24*time.Hour {
		return fmt.Sprintf("%d minutes", int(ttl.Minutes()))
	}
	return strings.TrimPrefix(humanTTL(ttl), "in ")
}
