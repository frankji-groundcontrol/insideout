# 2026-08-24 — Login form action hierarchy

Design-QA follow-up ([record](../design-qa/2026-08-24-login-form-action-stack.md)):
after auth parity added the code entry, the login form showed four
visually identical stacked text buttons under the primary 登录 (magic
link, code, forgot password, register toggle) — "so ugly".

## What changed

`AuthForm` password mode now separates the action classes:

- The two sign-in alternates (magic link, code) are compact
  side-by-side `OutlinedButton`s under a hairline labeled divider
  (`login.otherWays` — "More ways to sign in" / 其他登录方式), replacing
  two full-width text buttons.
- 忘记密码？ and the register toggle moved to a space-between footer
  row, so recovery/navigation links no longer masquerade as method
  switches.
- The alternate modes (magic/code, pre- and post-send) keep the single
  "Use password instead" back action; the register form is unchanged.

l10n: `login.magicLink`/`login.codeLink` replaced by the short
`login.magicLinkShort`/`login.codeLinkShort` (邮件链接 / 验证码,
"Email link" / "Email code") plus the divider label; both catalogs
updated.

## Verification

`flutter analyze` (no new issues, the 22 pre-existing lints unchanged)
and `flutter test` 48/48 — the two auth-form mode tests updated for the
new labels and asserting the divider appears in password mode and
disappears in the alternate modes.

[Design-QA record](../design-qa/2026-08-24-login-form-action-stack.md).
