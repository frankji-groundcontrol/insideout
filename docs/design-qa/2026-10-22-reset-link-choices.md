# 2026-10-22 — Reset link choices

## Comment

> Users can use the reset link on the web, but we should also allow them
> to reset on the CLI. For example, a user clicks the CLI and can reset
> their password, or copy-paste the reset code onto the CLI so they can
> use the reset code to reset their password on the CLI.
>
> So the process will be: the user has two options on the web link they
> click open. The user can either:
>
> 1. Reset their password there on the web.
> 2. Get a reset code so they can reset on the CLI.

## Decision

The reset page starts with those two choices. Reset on the page is the
existing form. The CLI choice shows the one-time code and the command
`insideout reset-password`. Copying the code does not use it up. Setting
the password on either side does.

## Files

- [reset_password_page.dart](../../client/lib/features/auth/reset_password_page.dart)
- [en_account.dart](../../client/lib/l10n/en_account.dart)
- [zh_account.dart](../../client/lib/l10n/zh_account.dart)
- [changelog](../changelogs/2026-10-22-reset-code-on-the-page.md)
