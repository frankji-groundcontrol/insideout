# Mail key (Resend)

`INSIDEOUT_MAIL_API_KEY` — transactional auth email (magic-link login,
one-time codes, password reset) via Resend.

## 1. Where the `.key` file is

`hosts/<host>/keys/mail/.secrets/mail-api-key.key` (manifest
`mail/resend.yaml`).

## 2. How it is propagated

`.env` → `dev.sh` export → the mailer at send time; deployed value in
the provider's variable store. Two non-secret companions are part of
the contract: `INSIDEOUT_PUBLIC_URL` (**required** the moment a key is
set, fail-fast, so emailed links never point at the localhost default)
and `INSIDEOUT_MAIL_FROM` (a Resend-verified sender; the sandbox
default only delivers to the account's own address).

## 3. How to set it

From the Resend dashboard. **Empty is a valid choice**: mail dev mode
logs recipient + subject instead of sending (full bodies only with
`INSIDEOUT_MAIL_DEV_BODY=1`, local only). Rotation note: this key was
printed into a local session transcript once (2026-10-04; nothing
committed or sent) — rotate it if transcripts ever leave the machine.

## 4. How to test it

Request a magic link from the login screen: delivery to a real inbox
proves key + sender + public URL together. HTTP 422 means the
*recipient* was rejected, not a bad key; a 5xx/absent mail with the
key set means configuration (sender verification, public URL).

## 5. How it is used

Switches auth mail from log-only dev mode to actually sending. All
five auth ways that email (magic link, code login, password reset,
invitations) ride it.

## 6. Installation guide

This file is the detail (no SETENV step — the key is optional and its
empty-state is dev mode). Contract rows (`INSIDEOUT_MAIL_*`,
`INSIDEOUT_PUBLIC_URL`) in [environment.md](../../environment.md).
