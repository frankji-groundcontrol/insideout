# BUG-025 — phone layouts and copy defects found by the web walk

## Symptom

The block-craft verification wave never ran a 390px web walk (it says so in
[the design-QA record](../design-qa/2026-10-07-block-craft.md)). Walking the
release web build in a real browser at 390x844, then again at 1440x900,
turned up five defects, none of which any widget test had looked at:

| # | Where | What a user saw |
|---|-------|-----------------|
| 1 | Roadmap header, 390px | The three view toggles (Progress, Bands, Canvas) ran off the left edge and could not be reached; the "New item" button squeezed the title so "ROADMAP" broke mid-word. |
| 2 | Roadmap Progress view | A user's first goal was answered with "No roadmap yet. Add a top-level goal…", although the goal existed. |
| 3 | PRD page, 390px | The title field was squeezed to a sliver beside "Next: Request review"; the 52px "PRODUCT REQUIREMENTS" heading broke "REQUIREMENTS" mid-word; a drag over the PRD column never moved the page, so the coach below it could be reached only by dragging from the coach's own area. |
| 4 | Versions page, zh | The commit card's readiness gaps read "背景与问题陈述 — the why-this, why-now argument a decision maker needs": the section was translated and the reason was not. |
| 5 | Dashboard card and replace prompt | "1 members" on a workspace with one member; "…a roadmap with 1 nodes…" in the replace prompt. |

The 1440px pass of the dashboard, workspace, project (Evidence), roadmap
(Progress, Bands, Canvas), PRD, and Versions pages was clean.

## Root cause

1. The roadmap header put seven controls in one row below the narrow
   breakpoint: the four utility icons plus the three toggles in the top-right
   cluster, which is a `Stack` child. A `Stack` clips instead of raising an
   overflow error, so the left edge simply cut the toggles off (the test
   measured a toggle at x = -119.4) and no analyzer or test complained.
2. The Progress view lists only deadlined work, in-progress work without a
   deadline, and a done count. A roadmap of to-do, undated goals therefore
   comes back empty from the server, and the page equated "empty progress"
   with "no roadmap".
3. The PRD page's title row was the same at every width: the title field
   shared a `Row` with the review button, and the heading used `displaySmall`.
   Below the wide breakpoint the page stacks, and it put the PRD column in a
   fixed 640px box with its own scrolling list inside the page's list; a touch
   drag that starts over an inner scrollable is taken by it.
4. The server's gap reasons (`readiness.go`) are a fixed English table keyed by
   audience and section; the client translated the section and printed the
   reason as sent.
5. Counts were interpolated into one plural-only string
   (`workspace.members`, `prd.buildReplaceConfirm`).

All five share one cause in the process: widget tests ran at the default
800x600 surface. `tester.binding.setSurfaceSize` resizes the layout but not
`MediaQuery`, which these pages read, so narrow layouts were never exercised
([learning](../learning/2026-10-10-test-the-width-you-ship-for.md)).

## Fix

Each fix is a test commit that fails, then the fix.

| # | Test (red) | Fix |
|---|-----------|-----|
| 1 | `1d05904` `roadmap_narrow_test.dart` (toggle at left -119.4, "ROADMAP" on two lines) | `5067f3b` below `Cobalt.bpNarrow` the toggles sit with the New item button under a full-width title; the header moves to `roadmap_header.dart`, which also keeps `roadmap_page.dart` under the line budget |
| 2 | `92f18a4` `roadmap_progress_empty_test.dart` | `c467f55` no goals keeps "No roadmap yet"; goals with nothing scheduled say "Nothing is scheduled yet…" (en, zh); blank until the goals are known so the wrong sentence never flashes |
| 3 | `9061ba3` `prd_narrow_test.dart` (field width 0, heading size equal to the wide one, coach header still at y = 704 after the drag) | `6aab2cc` stack the field and button, `headlineLarge` heading; `05e8b34` the column lays out at full height inside the page list (`PrdDeskColumn(embedded: true)`), so one drag scrolls through the sections and on to the coach (704 to 268) |
| 4 | `d2304d9` `versions_gap_l10n_test.dart` | `56ca03b` twelve `versions.gap.*` keys per locale, looked up by audience and section; an unknown pair keeps the server's sentence |
| 5 | `733036f` dashboard and `prd_actions` tests | `410f16d` `workspace.membersOne` and `prd.buildReplaceConfirmOne`, chosen when the count is one |

## Status

**Fixed 2026-10-10.** Re-walked in the browser after each fix: the 390px
roadmap and PRD pages hold their controls and headings, the first-goal
Progress view says what is true, the zh commit card is in Chinese, and the
singular reads right. The dock tap matrix at 390x844 held no console error,
which closes [W3-N2](2026-10-06-wave3-minified-console.md). Recorded in the
[web walk changelog](../changelogs/2026-10-10-close-open-gaps/web-walk-and-phone-layouts.md).

Not changed, on purpose: the stacked PRD layout's coach block is a fixed 640px
box, which does not block use. The English text the server stores for a commit's
carried items is translated where the commit card shows it, which also turned up
a worse problem: the card showed no carried items at all
([BUG-026](2026-10-10-bug-026-committed-versions-hid-what-they-recorded.md)).
