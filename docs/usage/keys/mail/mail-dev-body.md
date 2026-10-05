# mail-dev-body

`INSIDEOUT_MAIL_DEV_BODY` — dump rendered mail bodies to server logs
(dev only). Config companion: blank = default **off**; the file
exists so the kind is always findable.

## 1. Where the `.key` file is

`hosts/<host>/keys/mail/.secrets/mail-dev-body.key` (manifest
`mail/dev.yaml`).

## 2. How it is propagated

`.env` → `dev.sh` export → the mailer's dev branch; deployed value
in the provider's variable store.

## 3. How to set it

Exactly `1` turns body logging on; blank or any other value keeps it
off. Never enable where logs are shared — bodies contain login
links.

## 4. How to test it

`scripts/tests/keys/mail/test_mail-dev-body.py` (exists + only `1`
or blank). Live with `1`: trigger a login email and see the rendered
body in server logs.

## 5. How it is used

Read once at boot (`MailDevBody`); when on, the mailer logs each
rendered body in addition to sending via the
[mail-api-key](mail-api-key.md) provider.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 3
links here. Contract row in [environment.md](../../environment.md).
