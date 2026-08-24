# 2026-08-24 — Login form action stack

## Thread

User posted a screenshot of the login form (zh locale, AuthDoor modal)
after the auth-parity slice added the code-login entry, and said:

> so ugly

The screenshot showed four visually identical stacked red text buttons
below the primary 登录 button — 给我发登录链接 / 给我发验证码 /
忘记密码？ / 还没有账号？ — four different kinds of action (two
mutually-exclusive method switches, one recovery link, one navigation
link) rendered as one undifferentiated wall.

## Decision

Give each action class its own weight instead of stacking full-width
text buttons:

- The two sign-in alternates (magic link, code) become compact
  side-by-side secondary buttons under a labeled divider
  (`login.otherWays`: "More ways to sign in" / 其他登录方式).
- 忘记密码？ and the register toggle move to a space-between footer row.

Fix:
[changelog](../changelogs/2026-08-24-login-form-layout.md);
implementation in
[auth_form.dart](../../client/lib/features/auth/auth_form.dart) and both
l10n catalogs.

## Follow-up (same day)

> I think a prompt modal would be better, transparency is not apparent

The `AuthDoor` overlay used a translucent radial-gradient scrim
(8%–88% alpha scaffold color) over the landing, so the busy page bled
through unevenly and the panel floated without separation. Decision:
treat it as a real prompt modal — solid dim barrier (`Colors.black54`,
tap-to-dismiss) and an elevated solid-surface dialog panel. Same
[changelog](../changelogs/2026-08-24-login-form-layout.md); implementation
in [auth_door.dart](../../client/lib/features/auth/auth_door.dart).
