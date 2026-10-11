# BUG-026 — a committed version's card hid what the version recorded

## Symptom

After committing a version, its card on the Versions page showed the name, the
change summary, three diff counts and who committed it. Three things the commit
records were shown nowhere in the app:

- **The open items it carried.** The commit dialog says "{n} open item(s) will
  be carried as unresolved", the server stores them, and the API, CLI and MCP
  return them. [PRODUCT.md](../../PRODUCT.md) says a Commit "records a name,
  primary audience, change summary, unresolved validation items, and a Diff".
- **Which sections changed.** The card gave "added 1, changed 1, removed 0" and
  no names.
- **The decision note.** [PRODUCT.md](../../PRODUCT.md) asks for a Decision
  Log with a Commit. The server stores a `decisionNote`, but the commit
  dialog never asked for one (the 2026-10-04
  [review](../review/2026-10-04-multirole-2/01-findings-register.md) filed
  that as FEAT-02), so a note could only come from
  the CLI or MCP, and nothing showed it.

## Root cause

The first Flutter versions page (`17754bc`, 2026-08-21) listed all three under
each commit. The Cobalt restyle (`cc139a8`, 2026-09-18) rebuilt the page as the
fanned commit stack and carried none of them over. Their strings stayed in both
locale tables with no reader, and no test pinned any of the three, so the loss
was silent.

The 2026-10-05 planning-flow changelog says "the commit screen renders the
section diff, carried gaps, and decision note instead of three counts". At that
commit (`0216f25`) the client parsed `diffSections` and `decisionNote` but the
page rendered neither, so the sentence was not true of the app until this fix.

Found 2026-10-10 by following up a web-walk observation (carried items are
stored as English sentences): looking for where they are shown found nothing,
and a scan for locale keys that no code reads listed the other two.

## Fix

| Step | Commit |
| --- | --- |
| Carried items: red test, then the fix. Each stored gap (`<section>: <reason> (carried as open question)`) shows as the section label plus the reason in the reader's language; an item a caller named shows as stored | `e793dc3`, `e7f6a65` |
| The commit's details block moves into `CommitDetails` so the stack stays under the line budget (343 to 310 lines) | `cc6df46` |
| Red test: section lines (en, zh), decision note, and the dialog's note field | `31fb183` |
| Fix: one line per changed section under the counts (`· Background & Problem Statement — changed`), the decision note under its own heading, and an optional Decision note field in the commit dialog handed to `commitPrd(note:)` | `850345f` |
| Order: the real build showed the card scrolling with the note below eight section lines, so the card now reads summary, note, diff, carried items. Red test first (note at y=453, first section line at y=381) | `a2d436b`, `5f154c8` |
| Guard: a test that fails when a locale key has no reader in `lib/`, then deletion of the 31 keys that had none (62 lines, no insertions) | `82187c9`, `5309572` |
| Server: a real-PostgreSQL API test that a commit lists back its note and a diff naming each changed section as `{change: …}`; nothing covered either before | `def318b` |

## Verification

- Widget tests: `versions_commit_card_test.dart` (carried items en and zh,
  section lines en and zh, note, their order, and a commit that recorded none of
  them shows none) and `versions_page_test.dart` (the dialog sends name, summary
  and note). Full suite 267 tests, analyze at its baseline of 16 infos.
- Mutations: making the note stop reaching `commitPrd`, or stop rendering, fails
  the dialog test and both card tests; making `carriedItem` return the stored
  string fails the en and zh carried tests; making the store write an empty note
  fails the server test.
- Release build, real server, real PostgreSQL 18 (1440x900, zh): the dialog
  shows the new "决策备注（可选）" field; a commit with a note comes back with its
  summary, note and `diff.sections` (`goals`/`background` as `changed`) in
  `GET /prds/{id}/commits`; the card shows the summary, the note, the counts,
  one line per changed section and the carried items in Chinese; no console
  error (the only messages are the headless browser's WebGL notice).
- The first version of the section test was red for the wrong reason: its
  fixture gave each section as a bare string, the server writes
  `{change, bytes}` (`prdcommit.Diff`), so the page failed to parse and the test
  failed on a page error. It showed up when the fix still failed. The fixture now
  follows the server's output, and the test helper first asserts that the commit
  loaded, so a red means "not shown"
  ([learning](../learning/2026-10-10-a-restyle-leaves-its-strings-behind.md)).

## Status

**Fixed 2026-10-10.**
