# Keys — mail module

Per-key reference for the `mail/` module of the key store. Each file
documents one key kind with the six parts (where the `.key` file is,
propagation, setting, testing, use, and the installation-guide
pointer). The module table, shared flow, and per-host state live in
the [keys README](../README.md).

## Guided delivery flow

Use the [setup tutorial](../../key-setup-tutorial.md#5-configure-mail-when-auth-email-must-leave-the-server)
from the [installation guide](../../installation.md#your-own-server). The
[testing tutorial](../../key-testing-tutorial.md#6-prove-mail-delivery) uses
the admin provider probe and a disposable or operator-owned recipient. An auth
endpoint returning `200` is an anti-enumeration contract response; it is not
delivery evidence. The dev-body switch is local-only and must never be enabled
on a shared deployment.

- [mail-api-key.md](mail-api-key.md) — `INSIDEOUT_MAIL_API_KEY`
  (manifest `mail/resend.yaml`)
- [mail-from.md](mail-from.md) — `INSIDEOUT_MAIL_FROM`
  (`mail/resend.yaml`)
- [public-url.md](public-url.md) — `INSIDEOUT_PUBLIC_URL`
  (`mail/resend.yaml`)
- [mail-dev-body.md](mail-dev-body.md) — `INSIDEOUT_MAIL_DEV_BODY`
  (`mail/dev.yaml`)
