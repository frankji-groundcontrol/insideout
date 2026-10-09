# 2026-10-08 — Custom block primitives have no focus ring or keyboard activation

**Resolved 2026-10-08.** New
[`client/lib/theme/block_focus.dart`](../../client/lib/theme/block_focus.dart)
— `BlockFocus` wraps a block in `FocusableActionDetector` with an
`ActivateIntent` action, so Tab reaches it and Enter or Space fires the
callback. The ring is a `CobaltFocusRing` foreground painter that draws
a 2px `brand` rect inflated clear of the block's own ink border, no
blur; because it paints outside the child's bounds rather than adding an
outer border, wrapping never changes a measured control height.
Wired into `BlockButton` (enabled path only — a disabled block is not
wrapped, so it leaves Tab traversal), `BlockIconButton`, and
`PressableBlock`, which carries `BlockChoice` and every `FieldCard`
`onTap`. Seven widget tests in
[`client/test/block_focus_test.dart`](../../client/test/block_focus_test.dart);
`block_primitives_test.dart` passes unmodified. Gate: analyze 0
errors / 0 warnings (16 pre-existing infos), **167/167** tests, format
clean, `cobalt_test.dart` untouched
([changelog](../changelogs/2026-10-08-block-focus-ring.md)).

The original finding, kept below as filed.

---

The block-craft sweep replaced the app's keyboard-focusable Material
buttons with custom blocks that are `MouseRegion` + `GestureDetector`
only — no `Focus`, no `FocusNode`, no `Shortcuts`/`ActivateIntent`
anywhere in the primitive files:

- `client/lib/theme/block_button.dart` — `BlockButton` (~lines 156–177)
  and `BlockChoice`
- `client/lib/theme/block_chrome.dart` — `BlockIconButton` (~lines
  102–183)
- `client/lib/widgets/pressable_block.dart` — the motion wrapper the
  dock and `FieldCard` taps ride (~lines 49–70)

Consequence: every primary action, dock item, icon control, and
pick-one choice in the app is unreachable by Tab and shows no visible
focus indicator. This is an app-wide accessibility **regression**
against the pre-sweep baseline (Material `FilledButton` /
`OutlinedButton` / `IconButton` all had focus traversal and a focus
overlay) and against
[DESIGN.md](../../DESIGN.md), which mandates "Focus is a cobalt ring"
under Components → Buttons. Found by the Wave 7 fidelity review of the
[block-craft plan](../plans/2026-10-07-frontend-block-craft/2026-10-07-frontend-block-craft.md)
as its only high finding; the reference app shares the gap, but this
app regressed from a focusable baseline. Filed rather than fixed
inside the plan because the fix touches every primitive and needs its
own test pass, not a verification gate.

## Target behavior

- Each interactive primitive wraps its block in `Focus` (or uses
  `FocusableActionDetector`) with `canRequestFocus` tied to enabled
  state; Enter/Space activate; focus paints a cobalt ring per DESIGN.md
  (2px `brand` outline offset outside the ink border, no blur).
- The press choreography's hover state doubles as the focus visual only
  if it stays distinguishable from hover; otherwise a distinct ring.
- `MediaQuery.disableAnimationsOf` honored, semantics unchanged.

## Bounded fix prompt

"Give `BlockButton`, `BlockChoice`, `BlockIconButton`, and
`PressableBlock` keyboard focus and activation with a cobalt focus
ring, per
[docs/issues/2026-10-08-block-focus-ring-keyboard.md](2026-10-08-block-focus-ring-keyboard.md).
Add widget tests: Tab traversal reaches a `BlockButton` in a real form,
Enter/Space fire `onPressed`, and the focused block paints the ring
(finder on the ring widget, both themes). No visual change to the
rest/hover/active choreography — `block_primitives_test.dart` passes
unmodified. Gate: `cd client && flutter analyze --no-fatal-infos &&
flutter test && dart format --output=none --set-exit-if-changed lib
test`. Record in a changelog entry and close this issue."
