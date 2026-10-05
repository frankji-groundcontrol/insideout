# mail-from

`INSIDEOUT_MAIL_FROM` — the From address of every auth email. Config
companion of [mail-api-key](mail-api-key.md).

## 1. Where the `.key` file is

`hosts/<host>/keys/mail/.secrets/mail-from.key` (manifest
`mail/resend.yaml`). Format is display-name sender:
`Name <addr@host>` — brackets are legitimate here (the placeholder
detector excludes `@`-bearing bracket tokens for exactly this reason).

## 2. How it is propagated

`.env` → `dev.sh` export → the mailer at send time; deployed value in
the provider's variable store. Catalog default is the provider's
sandbox sender, which delivers only to the account's own address.

## 3. How to set it

A provider-verified sender (verify the domain in the mail provider's
dashboard first). Display name optional; angle-bracket address
required.

## 4. How to test it

Request a magic link to a real inbox — delivery proves key + sender +
[public-url](public-url.md) together. 422 means the *recipient* was
rejected, not a bad sender.

## 5. How it is used

The From header of magic-link, code, reset, and invitation emails.

## 6. Installation guide

This file is the detail. Contract row in
[environment.md](../../environment.md).
