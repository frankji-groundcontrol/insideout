# 2026-10-08 — Block primitives get keyboard focus and the cobalt ring

Closes the only **high** finding of the block-craft fidelity review: the
sweep had replaced the app's keyboard-focusable Material buttons with
custom blocks that were `MouseRegion` + `GestureDetector` only, so every
primary action, dock item, icon control, and pick-one choice became
unreachable by Tab and showed no focus indicator. That was a regression
against the pre-sweep baseline and against
[DESIGN.md](../../DESIGN.md), which mandates "Focus is a cobalt ring".

Filed as
[issue 2026-10-08-block-focus-ring-keyboard](../issues/2026-10-08-block-focus-ring-keyboard.md),
found by Wave 7 of the
[block-craft plan](../plans/2026-10-07-frontend-block-craft/2026-10-07-frontend-block-craft.md),
recorded in the
[design-QA thread](../design-qa/2026-10-07-block-craft.md).

## What changed

**New — `client/lib/theme/block_focus.dart`.** `BlockFocus` wraps a block
in `FocusableActionDetector` with an `ActivateIntent` action, so Tab
reaches it and Enter or Space fires the callback through the platform's
default shortcuts rather than a hand-rolled key handler. (On the web Enter
is a different intent and did nothing until 2026-10-10:
[BUG-028](../issues/2026-10-10-bug-028-enter-did-not-activate-focused-blocks-on-the-web.md).)

The ring is `CobaltFocusRing`, a `CustomPainter` used as a **foreground
painter**: it draws a 2px `brand` rect inflated `gap + borderContainer/2`
clear of the block's own ink border, zero blur. Painting outside the
child's bounds instead of adding an outer border is the load-bearing
choice — an outer border would grow every control by 4px and break the
measured fixed heights (32/40/52 for buttons, 44/40 for fields) that
`block_primitives_test.dart` asserts.

**Wired into three places, which covers the app:**

| File | Effect |
|---|---|
| `theme/block_button.dart` | `BlockButton` enabled path wraps in `BlockFocus`. The disabled path returns early unwrapped, so a disabled control leaves Tab traversal instead of being focusable-but-inert. |
| `theme/block_chrome.dart` | `BlockIconButton` wraps its body inside the existing `Tooltip`. |
| `widgets/pressable_block.dart` | `PressableBlock` wraps its `MouseRegion`, which carries `BlockChoice`, every `FieldCard` with an `onTap`, and the dock items. |

No existing choreography changed: rest `cast` (4,4) → hover `castSm`
(2,2) + translate (2,2) → active flush + translate (4,4), all over
`Cobalt.pressDuration`. Hover and focus stay visually distinct — hover
moves the block, focus rings it.

**Tests — `client/test/block_focus_test.dart`, 7 new.** Tab reaches a
`BlockButton`; Enter fires `onPressed`; Space fires `onPressed`; the
focused block paints a ring whose color equals `palette.brand` in the
dark theme and again in the light theme (separate tests, because a loop
over both reuses the `State` and leaks the focused flag); the measured
size is byte-identical focused vs unfocused and still equals
`BlockButtonSize.md.height`; a disabled `BlockButton` contains no
`BlockFocus` and paints no ring; `BlockIconButton` and `PressableBlock`
both activate from the keyboard in Tab order. The ring is found by its
painter type, so an unrelated `CustomPaint` cannot satisfy the assertion.

`client/test/block_primitives_test.dart` and `client/test/cobalt_test.dart`
are both **unmodified**.

## Verification

From `client/`, all three gates with their real exit codes:

```
$ dart format --output=none --set-exit-if-changed lib test
Formatted 135 files (0 changed) in 0.30 seconds.        exit 0

$ flutter analyze --no-fatal-infos
16 issues found. (ran in 1.1s)                           exit 0
  (16 infos, 0 errors, 0 warnings — all pre-existing style
   infos in lib/api/client.dart, lib/api/requests.dart,
   test/project_page_test.dart, test/sse_test.dart)

$ flutter test
00:09 +167: All tests passed!                            exit 0
```

160 → **167** tests. `git diff --stat HEAD -- client/test/cobalt_test.dart`
is empty, confirming the frozen token assertions were not touched.

**Not run:** the browser web walk. Focus behaviour is asserted by widget
tests here, but the real Tab order across a signed-in page and the ring's
appearance on a live canvas still belong to the block-craft plan's
outstanding verification item 4.

The regression class is recorded as a reusable lesson:
[replacing a platform widget drops the affordances no test asserts](../learning/2026-10-08-custom-primitives-drop-unasserted-affordances.md).
The gate was green at 160/160 while the hole was open, because no test
pressed Tab — an adversarial read-only review found it by grepping the
new primitives for `Focus`.

## Operator notes

Nothing to deploy on its own — this is part of the same uncommitted
`client/` slice as the
[2026-10-07 block-craft changelog](2026-10-07-frontend-block-craft.md).
The worktree also still carries the app-icon slice, so stage one task at
a time; do not use `git add .`. Ship `client/` changes with
`railway up --service app`.
