# Web walk: phone layouts, copy, and Build MVP in a real browser

The block-craft wave never ran a 390px web walk
([design-QA record](../../design-qa/2026-10-07-block-craft.md)). This wave did,
and every defect it found was fixed test-first.

## What was walked

- **Build:** the release web bundle of the Flutter client, served by a static
  single-page-app host, against a locally built Go server on a scratch
  PostgreSQL 18 with the demo data (`insideout seed`).
- **Driver:** gstack's headless Chromium at 390x844 and again at 1440x900.
  Flutter's accessibility tree is off by default on the web; the walk turns it
  on by clicking the `flt-semantics-placeholder` element, then drives the page
  through that tree and a few JavaScript clicks where the snapshot tool does not
  list a control.
- **Pages:** dashboard, workspace, project (Evidence), roadmap (Progress, Bands,
  Canvas), PRD, Versions, and the dock between them.

## What it found

| Finding | Record |
| --- | --- |
| Roadmap header toggles cut off at 390px; "ROADMAP" broke mid-word | [BUG-025](../../issues/2026-10-10-bug-025-phone-layouts-and-copy-found-by-the-web-walk.md) #1 |
| Progress view answered a first goal with "No roadmap yet" | BUG-025 #2 |
| PRD page: title field squeezed, heading broke, a drag never reached the coach | BUG-025 #3 |
| Commit card gap reasons half in English in zh | BUG-025 #4 |
| "1 members", "1 nodes" | BUG-025 #5 |
| Build MVP drafted but wrote nothing (draft-then-confirm had no confirm in the UI) | [BUG-022](../../issues/2026-10-10-bug-022-build-mvp-wrote-nothing.md) |
| A committed version's card showed none of the open items, section lines or decision note it recorded | [BUG-026](../../issues/2026-10-10-bug-026-committed-versions-hid-what-they-recorded.md), [commit-card-details.md](commit-card-details.md) |

BUG-025's five defects were fixed with a red test first and re-walked in the
browser. Build MVP was exercised end to end: draft, the confirm step, the
navigation to the roadmap, and the replace dialog when the roadmap already has
nodes.

## Closed without a change

- **W3-N2** (a minified console error from the dock): not reproducible. A tap
  matrix over every dock button at 390x844, on a release build with source maps,
  produced zero console errors
  ([record](../../issues/2026-10-06-wave3-minified-console.md)).
- **1440x900:** the dashboard, workspace, project, roadmap, PRD and Versions
  pages were clean.

## Reviewed and kept

- The Versions card's gap list scrolls inside a fixed-height card.
- A `401` on `/me` in the console is the token-refresh flow.
- Delete buttons use the Cobalt danger variant, which is blue by design.
- Action buttons are full width and stacked on the 1440px project page.
- The dock's Version and Commit buttons both open `/versions`, by design.

## How it was verified

Every fix has a failing test commit before the fix commit; the widget tests
resize the view (`tester.view.physicalSize`, not `setSurfaceSize`) so
`MediaQuery` sees the phone width
([learning](../../learning/2026-10-10-test-the-width-you-ship-for.md)). The
screenshots are kept locally under `.gstack/browse-reports/` (ignored by git).

## Operator notes

None. Nothing here is deployed.
