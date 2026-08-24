# 2026-08-25 — Mail delivery enabled on production

The operator-owned step from the
[email-auth slice](2026-08-24-email-auth.md) is done: the user set the
Resend key in the local `.env`; it was propagated into the Railway
`server` environment (`INSIDEOUT_MAIL_API_KEY` + `INSIDEOUT_MAIL_FROM`,
values never printed — sourced in a subshell like `dev.sh` does) and the
service redeployed (plus the `app` restart for the nginx re-resolve).

Verification: a live register + code request shows **no** "mail dev
mode" log lines and a real Resend API round trip
(`mail: resend returned 422` on an `@example.com` probe — Resend
rejecting the recipient under sandbox rules, which proves the key is in
use). `INSIDEOUT_MAIL_FROM` is a custom sender, so delivery to any
address works once that sender's domain is verified in the Resend
account (until verified, Resend delivers only to the account's own
email). The real-inbox confirmation is the user's next test from the
web app.
