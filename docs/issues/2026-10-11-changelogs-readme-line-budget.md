# Changelogs README exceeds the line budget

**Date:** 2026-10-11
**Kind:** structural debt (line-budget split)
**File:** [`docs/changelogs/README.md`](README.md) — 473 lines against the
~350-line budget, and it grows by roughly one entry per change record.

## Why it grew

The Records section carries a dated bullet per changelog file (130 files),
each with a one-line description. The convention text and the per-record
lines together passed the budget during the 2026-10-10/11 gap-closure and
remediation work.

## Target structure

Keep `README.md` as the convention + a table of contents by month, and move
the per-record bullets into per-month index files the README links:

- `docs/changelogs/README.md` — convention, how to write a record, and a
  short month index (one line per month file, ~60 lines total).
- `docs/changelogs/index/2026-07.md`, `index/2026-08.md`, … — the dated
  bullet lists, newest month linked first from the README.

## Bounded fix prompt

Move the existing bullets verbatim into the month files (no rewriting of
record descriptions), regenerate the README as convention + month links,
then re-run the link-graph gate (`orphans=0`, `md_broken=0`) — every
changelog file must still be linked from exactly one month index, and the
learning/issues/practice records that link `../changelogs/…` files
directly are unaffected because record files themselves do not move.

## Resolution notes (2026-10-11)

- Three moved descriptions had been rewritten during the split; on review
  they were reverted to the README's verbatim text (owner-deploy,
  frontend-block-craft, release-pipeline). The fresher facts live in those
  records' own dated correction notes, so nothing was lost.
- The old README listed 2026-07-26-backend-optimization-pass twice (a
  shorter entry in date position, a fuller one with the deferred-findings
  link appended later); the split collapses the duplicate to the fuller
  entry, keeping the file linked from exactly one month index.
- One number is aligned rather than moved verbatim: the
  2026-10-10-multirole-review-3 bullet reads "32-cluster", matching the
  typo correction now carried by the record itself (the README's "30" was
  the same typo the rv3 plan corrected). Ruling: the pure-move gate
  protects descriptions from drift, not known-wrong numbers.
- `2026-07-23-backend-version-control.md`, never listed by the old README,
  gained its first bullet in `index/2026-07.md`, so the "exactly one month
  index" acceptance now covers every record.
