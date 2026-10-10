# 2026-10-10 — A VM test cannot show what the web's key map does

## Context

The 2026-10-08 focus-ring fix gave the custom blocks Tab traversal and Enter and
Space activation, with seven widget tests that pressed both keys
([record](../issues/2026-10-08-block-focus-ring-keyboard.md)). They passed, and so
did the other 260. In a real browser two days later Space opened a focused row
and Enter did nothing ([BUG-028](../issues/2026-10-10-bug-028-enter-did-not-activate-focused-blocks-on-the-web.md)).

Flutter's default key map is chosen per platform. `flutter test` runs on the VM,
so every key test used the VM map, where Enter is `ActivateIntent`. On the web
Enter is `ButtonActivateIntent`, and the control bound only `ActivateIntent`.

## Lesson

A green key test says the control works under the map the test ran with, not
under the one the product ships with. For a control that handles key intents:

- Run the test under the shipping platform's map. In a widget test, pass the
  web table to `MaterialApp(shortcuts: ...)` explicitly (the VM cannot be made to
  report `kIsWeb`); keep the test red against the control before the fix.
- Press the keys once in a real browser: Tab, Space, Enter takes two minutes and
  finds this whole class.
- When a record says a keyboard path works, say on which platform it was seen.

The earlier note on [affordances no test asserts](2026-10-08-custom-primitives-drop-unasserted-affordances.md)
is the other half: there the tests never pressed Tab; here they pressed the key
under the wrong table.

## When to apply

- Any `FocusableActionDetector`, `Actions` or `Shortcuts` that handles
  `ActivateIntent` or another intent the default shortcuts bind differently by
  platform.
- Any new custom control that replaces a Material one (the Material buttons
  already handle both intents).
- A web-first app whose test suite only ever runs on the VM.
