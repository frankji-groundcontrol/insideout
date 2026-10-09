# Replacing a platform widget drops the affordances no test asserts

## What was learned

When a design-system pass swaps framework primitives for hand-built ones,
it silently discards the **non-visual affordances the framework gave for
free** — focus traversal, keyboard activation, a focus indicator,
semantics roles, hover cursors, tooltip plumbing. None of them are
visible in a screenshot, none of them are implied by the design decree,
and a green gate cannot see an affordance that no test asserts. The
result is a full-fidelity visual pass that is a regression for anyone not
using a mouse.

The decree itself was not the problem. "0px corners, 2px borders, hard
cast shadows, 100ms press" describes *paint*. Material's `FilledButton`
supplied paint **and** focus; the replacement block supplied only paint,
and nothing in the specification mentioned what had been lost.

## Evidence

The [block-craft pass](../plans/2026-10-07-frontend-block-craft/2026-10-07-frontend-block-craft.md)
replaced Material buttons across every surface with `BlockButton`,
`BlockIconButton`, and `PressableBlock`. All three were `MouseRegion` +
`GestureDetector` only:

```
$ grep -c 'Focus\|FocusNode\|Shortcuts\|Actions' \
    lib/theme/block_button.dart lib/theme/block_chrome.dart \
    lib/widgets/pressable_block.dart
lib/theme/block_button.dart:0
lib/theme/block_chrome.dart:0
lib/widgets/pressable_block.dart:0
```

At that moment the gate was fully green — `flutter analyze` 0 errors /
0 warnings, **160/160 tests passed**, `dart format` clean, and
`cobalt_test.dart` unmodified. Every test passed because not one of them
pressed Tab.

`DESIGN.md` had mandated "Focus is a cobalt ring" under Components →
Buttons since before the pass; the sentence survived into the amended
file while no code implemented it. The gap was found by an adversarial
**read-only fidelity review** that grepped for `Focus` in the new
primitives — not by any gate, and not by the sweep agents that wrote
them. Filed as
[issue 2026-10-08-block-focus-ring-keyboard](../issues/2026-10-08-block-focus-ring-keyboard.md),
then fixed by `client/lib/theme/block_focus.dart`
([changelog](../changelogs/2026-10-08-block-focus-ring.md)): a
`FocusableActionDetector` with an `ActivateIntent` action, plus a ring
painted by a **foreground painter outside the child's bounds** — an
outer border would have grown every control by 4px and broken the
measured fixed heights the primitive tests assert.

Two details made the fix safe rather than another round of drift: a
disabled block is *not wrapped at all*, so it leaves Tab traversal
instead of being focusable-but-inert; and the ring's cost to layout is
zero, which a test asserts directly by comparing `getSize` focused vs
unfocused.

## Scope

Any replacement of a platform or framework component with a custom one:
buttons, fields, dialogs, tabs, menus, chips, sliders. Also any
"pixel-exact port" of a reference design, because a reference app's CSS
or widget tree rarely carries over the host framework's accessibility
behaviour — the reference Flutter app in this port shares the same gap,
so copying it faithfully copied the defect.

## When to apply it again

Before replacing a framework primitive, **enumerate what it gives for
free** and write the list down: focus traversal, activation keys, focus
indicator, `Semantics` role and label, hover cursor, tooltip, disabled
semantics, text-input and autofill plumbing. Then assert each item in
the replacement's test, in the same commit that introduces it. The list
is short and mechanical; the regression it prevents is app-wide.

Add the affordance assertions to the primitive test, not to page tests —
one `block_focus_test.dart` covering the primitive protects every call
site, whereas per-page coverage would have to be repeated dozens of
times and will be skipped under deadline.

And treat "the design doc already mandates it" as no protection at all.
The mandate existed for two plans; only a grep for `Focus` in the
implementation found the hole. When a review is the only thing that can
catch a class of defect, that class needs a test.
