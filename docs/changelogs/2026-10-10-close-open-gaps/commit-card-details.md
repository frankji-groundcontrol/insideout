# The commit card shows what a version recorded

A committed version records its change summary, the open items it carried, a
diff against the previous commit and, per
[PRODUCT.md](../../../PRODUCT.md), a Decision Log. The
Cobalt restyle's commit card showed the summary and three diff counts, and
nothing else
([BUG-026](../../issues/2026-10-10-bug-026-committed-versions-hid-what-they-recorded.md)).

## What changed

- **Carried open items** appear under their own heading, translated like the
  gaps they came from (section label plus the reason in the reader's language);
  an item a caller named shows as stored.
- **Section diff:** one line per changed section under the counts
  (`· Background & Problem Statement — changed`).
- **Decision note:** shown right under the summary, and the commit dialog now
  asks for it ("Decision note (optional)"), so a note no longer has to come from
  the CLI or MCP.
- **Order:** summary, note, diff counts and section lines, carried items. The
  card is a fixed height and scrolls, so the short human text comes first.
- **Locale guard:** `l10n_unread_keys_test.dart` fails when a locale key has no
  reader in `lib/`. A scan found 35 of 384 keys with none: four were the lost
  details' strings (the three diff verbs and the note heading), which the fix put
  back to use; the other 31 (old page titles, a recorded-facts panel, generic
  `common.*` and `*.notFound` strings) were deleted from all eight tables, 62
  lines and no insertions.
- **Server test:** a real-PostgreSQL API test that a commit lists back its note
  and a diff naming each changed section as `{change: …}`.

New string: `versions.noteLabel` (en, zh). No API or schema change: the server
already stored and returned all of it.

## How it was verified

- 267 Flutter tests, analyze at its baseline of 16 infos, `dart format` clean.
  Mutations (drop the note on the way to `commitPrd`, stop rendering it, make
  `carriedItem` return the stored string, make the store write an empty note)
  each fail a test.
- Release web build against the real server and PostgreSQL 18, 1440x900, in
  Chinese: commit with a summary and a note, then reload; the card shows the
  summary, the note, the counts, the changed sections and the carried items,
  and the API returns the same fields.
- Moving the details block into `CommitDetails` is a move plus scaffolding: 35
  lines, differing only by the widget class and `commit!` becoming `commit`.

## Operator notes

None. Nothing here is deployed; the hosted bundle predates it.

*Superseded 2026-10-11: the hosted bundle now carries it — the 2026-10-11
owner deploy shipped both services
([record](../2026-10-11-owner-deploy-and-mail-truth.md)).*
