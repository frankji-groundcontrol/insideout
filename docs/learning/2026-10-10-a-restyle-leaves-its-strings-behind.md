# 2026-10-10 — A restyle leaves its strings behind; make a red prove what it claims

## Context

The Cobalt restyle rebuilt the Versions page as a fanned commit stack and
dropped three things a committed version records: its carried open items, one
line per changed section, and its decision note. Nothing failed. No test
pinned them, the analyzer does not look at locale tables, and the only trace
was strings no code read, for three weeks
([BUG-026](../issues/2026-10-10-bug-026-committed-versions-hid-what-they-recorded.md)).
A changelog even claimed the screen showed them. Fixing it taught three
smaller things.

## Lesson

- **A string nobody reads is a feature that went missing.** After a restyle,
  scan for locale keys with no reader. `l10n_unread_keys_test.dart` now does it
  on every run: a key counts as read when its literal appears under `lib/`
  outside the tables, or sits under a prefix built at runtime
  (`'prd.sections.$key'`). 35 of 384 keys had no reader: four were the lost
  details (the three diff verbs and the note heading), the other 31 dead weight
  that is now deleted.
- **Build a fixture from the producer, not from the consumer's idea of it.** The
  first section test gave `diff.sections` as `{key: 'changed'}`; the server
  writes `{key: {change, bytes}}`. The page threw while parsing, the test failed
  on a page error, and it looked like a correct red until the fix still failed.
  Read the producing function (`prdcommit.Diff`) and copy its shape.
- **Make the test helper assert its precondition.** One `expect` after the pump
  (the commit's name is on screen) means a red says "not shown" and not "the
  page broke".
- **Look at the real card.** A test that finds the decision note proves it is in
  the tree, not that anyone can see it: on the release build the note sat below
  eight section lines inside a fixed-height scrolling card. Open the page at
  the width people use, put the short human text first and the long lists last,
  and keep the order in a test (`getTopLeft` of the note against the first
  section line works off-screen too).

## When to apply

- After any restyle or page rebuild that replaces a screen wholesale.
- Whenever a test fails on a fixture you wrote from memory of an API.
- Whenever content lives in a fixed-height scroll area.
