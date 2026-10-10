# Client changes — review-3 remediation (waves 2, 3, 4)

Commits: wave 2 `c89faea..ce8e173`, wave 3 `ce8e173..b056e30`,
wave 4 `b056e30..4464330`. Register rows each change closes are named.

## Coach panel and stage flow

- **The stage-confirm door exists** (USER-01, wave 1 client half,
  `ea9cd6c`): confirm/decline appear in the coach panel only while a stage is
  proposed, a proposed stage reads as proposed, and decline keeps the
  confirmed stage.
- **The interrupted marker is localized** (USER-08, integrator closure,
  `bce47c4`): the client strips the stable `[interrupted]` token and renders
  `coach.interruptedSuffix` in the reader's locale (en/zh keys).
- **An offline coach stream badges the panel** (`c620a78`) instead of leaving
  the reader guessing; **an absent `mailEnabled` reads as unknown, never as
  mail-on** (PM-04 code half, `673aa3d`).
- **Real masthead copy** (USER-06, `d3ed160`) replaces placeholder strapline
  text; the fact snackbar says *recorded fact* in both locales (FEAT-07,
  `11f22bf`).

## Capture and navigation

- **Capture with zero workspaces lands on the new ideas wall** (USER-09,
  `4b1af8d`), the wall pulses, and dock steps already on screen disable
  (FULL-04, `4d3a255`).
- **One snapshot flow for both pages** (CQ-03, integrator closure,
  `bce47c4`): `showSnapshotRevision` serves the PRD page and the revisions
  page, whose `onCreated` reloads the list (the bare `_reload` assignment
  never rebuilt the FutureBuilder).
- **First visit seeds the locale from the browser's language** (USER-03
  client half, `bc1f4ee`); the export page title is a neutral *Export*
  (FULL-07, `72d82dc`); one chevron action tile serves the field and
  workspace pages (CQ-06, `a59f0dd`).

## Polish and layout law

- **The auth door's lockup gap is tokenized** (CQ-01, `d733ea2`) and pinned
  by a spacing scan; loads that outlive their pages no longer write to dead
  controllers (REL-04, `69b93c3`).

## Validation leg (wave 4)

- **The Evidence screen keeps a validated leaf's observation**
  (`b1e5010`); recording an observation and confirming validation round-trips
  through the new API surface (`d79679a`).

## Verifier-visible gates

`flutter test` — 456 tests passing (up from 421 at the review base);
`flutter analyze` at the 14-info baseline; `dart format` clean
([verification.md](verification.md)).
