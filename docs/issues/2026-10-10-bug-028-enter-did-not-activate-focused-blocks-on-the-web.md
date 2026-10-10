# BUG-028 — Enter did not activate a focused block in a browser

## Symptom

In the release web build, Tab reached the custom blocks and drew the cobalt ring
and Space opened the focused row (想法 to Ideas, 设置 to Settings), but Enter did
nothing, on both rows tried. Every widget test pressed Enter on the same blocks
and passed. Found by the keyboard pass of the 2026-10-10 web walk (real
Chromium, 1440x900, dark theme).

## Root cause

Flutter's default key map, `WidgetsApp.defaultShortcuts`, differs by platform.
Space is `ActivateIntent` everywhere. Enter is `ActivateIntent` on the VM and on
desktop and mobile, but on the web it is `ButtonActivateIntent` (the framework's
own comment: "on the web, enter activates buttons, but not other controls").

`BlockFocus` (`client/lib/theme/block_focus.dart`) carries `BlockButton`,
`BlockIconButton` and `PressableBlock`, and through them `BlockChoice`, every
`FieldCard` tap and the dock. It bound only `ActivateIntent`, so on the web an
Enter key press found no action and was dropped. The tests all run on the VM,
where Enter is `ActivateIntent`.

The 2026-10-08 focus-ring record says "Tab reaches it and Enter or Space fires
the callback". That held on every platform but the one the product ships to.
Introduced with the block component layer (`185ed46`).

## Fix

| Step | Commit |
| --- | --- |
| Red test: the four focusable blocks (button, icon button, choice, row) under the web key map. Space fires each once, Enter fires none | `3c8cc9e` |
| Fix: `BlockFocus` runs the same activation for `ButtonActivateIntent` as for `ActivateIntent` | `c2574cb` |

## Verification

- `block_focus_test.dart`: `Enter activates every focusable block under the web
  key map`. The test replaces `MaterialApp.shortcuts` with the web map (the
  default map plus Enter and NumpadEnter as `ButtonActivateIntent`), focuses each
  block, presses Space and then Enter, and expects two hits each. Red for that
  reason: `at location [1] is 'icon' instead of 'button'` (Space fired, Enter did
  not).
- Mutation: removing the `ButtonActivateIntent` action brings back the same red.
- Real browser, release build, real server: Tab to 想法, Enter, the URL becomes
  `/workspace/<id>/ideas`; Tab to 设置, Enter, `/workspace/<id>/settings`; Tab
  to 编辑 (a `BlockButton`), Enter, the edit dialog opens.

`BlockFocus` is the only place in `client/lib` that binds an activation intent
(`grep ActivateIntent`); the other keyboard code is the dialog's Escape binding
and the canvas's scroll keys, and the Material controls are the framework's own.

## Status

**Fixed 2026-10-10.** The lesson, that a VM test cannot show what a platform's
key map does, is in
[the learning note](../learning/2026-10-10-vm-tests-do-not-run-the-web-key-map.md).
