# Web walk, second pass: both themes, the keyboard, and two more defects

The block-craft plan's verification item 4 asks for a web walk at 1440x900 and
390x844 in both themes: "buttons cast and press, modals share one chrome, no
round spinner anywhere, errors localized in a banner, content column
constrained on wide"
([plan](../../plans/2026-10-07-frontend-block-craft/2026-10-07-frontend-block-craft.md)).
The [first pass](web-walk-and-phone-layouts.md) found the phone-layout defects.
This one added the dark theme, the keyboard and the pressed state, and found two
more.

## What was walked

Same rig as the first pass: the release web bundle, a locally built Go server on
a scratch PostgreSQL 18 with the demo data, gstack's headless Chromium, and
Flutter's accessibility tree switched on through `flt-semantics-placeholder`.

| Viewport | Theme | Pages |
| --- | --- | --- |
| 1440x900 | light | dashboard, workspace, project, roadmap (Progress, Bands, Canvas), PRD, Versions, Build MVP states |
| 1440x900 | dark | dashboard, workspace, roadmap Canvas, PRD (loading state), Versions; the keyboard pass below |
| 390x844 | light | dock destinations, roadmap, PRD, Versions, workspace |
| 390x844 | dark | dashboard, workspace, roadmap, PRD, Versions, profile |

## What it found

| Finding | Record |
| --- | --- |
| Tab reaches the custom blocks and the ring shows, but Enter does nothing in a browser (Space works) | [BUG-028](../../issues/2026-10-10-bug-028-enter-did-not-activate-focused-blocks-on-the-web.md) |
| The Versions eyebrow read "Commit 3 of 2" | [BUG-029](../../issues/2026-10-10-bug-029-versions-eyebrow-counted-revisions.md) |

Both were fixed with a red test first and re-checked in the browser: Enter on
the 想法 and 设置 rows goes to Ideas and Settings and on 编辑 opens the edit
dialog; the Versions page reads 第 2 次提交，共 2 次.

## Item 4, bullet by bullet

- **Buttons cast and press:** the cast shadow shows on every block in both
  themes. The pressed state was driven with synthetic pointer events on the
  Flutter glass pane: held down, the floating New project button sits 2px right
  and down of where it rests; released, it returns.
- **Modals share one chrome:** Add goal in light and dark (screenshots) shows a
  header with a close block, a divider, the body and a footer bar; Edit
  workspace and New project open with the same close block, fields and
  cancel/confirm buttons in the accessibility tree.
- **No round spinner anywhere:** the PRD page's loading state is the square
  block with its label; the only `CircularProgressIndicator` in `client/lib` is
  the boot splash's, which the plan protects (first-frame gate).
- **Errors localized in a banner:** a wrong sign-in showed the Chinese message
  (邮箱或密码不正确) while the language was Chinese.
- **Content column constrained on wide:** at 1440px the dashboard, roadmap
  Canvas and Versions content sits in a centered column, not edge to edge.

Console: no application error, only the headless browser's WebGL notice and the
`401` on `/me` that starts the token-refresh flow.

## Not covered

The pressed state was seen on the floating New project button; the other blocks share
the same press code (`PressableBlock`, `BlockButton`) and the widget tests
assert it for each. Screenshots stay local under `.gstack/browse-reports/`
(ignored by git).

## Operator notes

None. Nothing here is deployed.

*Superseded 2026-10-11: deployed — the 2026-10-11 owner deploy shipped the
bundle ([record](../2026-10-11-owner-deploy-and-mail-truth.md)).*
