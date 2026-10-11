# Review-register sweep: the client

The client half of the [review-register sweep](review-sweep-server.md). Every
fix was written test-first: the widget or wire test failed on the old code for
the stated reason and was committed red, then the fix turned it green, and each
fix was mutated once to prove the test goes red again. Client gates after each
commit: `flutter analyze --no-fatal-infos` at the 16-info baseline,
`dart format --set-exit-if-changed lib test`, `flutter test`.

## Coach and desk

- **A PRD with no conversation had no way to start one** (`r2/USER-20`). The
  desk said "no conversation" and stopped. A PRD's author or an admin now sees a
  **Start coaching** block that calls the new `POST /api/v1/prds/{id}/conversation`
  (`201` for a new one, `200` when one already exists, `404`, `403`). The server
  side, `EnsureConversationForPrd`, takes a per-(PRD, user) advisory lock, so two
  taps cannot create two conversations. A plain member keeps the sentence.
- **A member was offered Approve and Reject** (`r1/FEAT-06`). The server refused
  the call; the buttons were enabled anyway. The page loads the viewer's role and
  disables the two decisions for a member (an unknown role fails open, so a
  failed role read never locks an admin out).
- **A blank section card gave no hint** (`r3/prd-section-cards-blank-no-placeholder`).
  Each of the eight section editors now shows a hint (`prd.sectionHint.<key>`, en
  and zh); the title field has none.
- **Critic findings showed only the issue** (`r3/feat-critic-findings-incomplete`).
  A finding now shows its severity, the review text and the suggestion, and the
  list clears when the stage changes (not on every send: the server keeps the
  list across turns that do not critique, on purpose).
- **A late stream frame no longer throws** (`r3/rel-coach-setstate-after-dispose`):
  the delta handler checks `mounted` like the other handlers.
- **The "fact recorded" snackbar printed the whole payload**
  (`r3/feat-coach-error-contract-partial`). It now shows the fact's text, and
  nothing for a missing or empty text; the test fixture uses the shape the server
  sends.

## Roadmap and project

- **Leaving a roadmap page now ends its streams** (`r3/rel-sse-streams-never-cancelled`).
  The progress and presence connections stayed open after the page was gone. A
  stop signal now cancels them (`CancelToken` natively, `AbortController` on the
  web) through one helper in `client_stream.dart`. Proven with a loopback HTTP
  server that sees the client leave; the web path was checked by building the
  release bundle, not by a browser test.
- **One retry loop, one timer set** (`r3/rel-shared-retry-timer`,
  `r3/cq-shared-retry-timer`). The two near-identical reconnect loops are one
  `_retrying` helper, and dispose cancels every pending backoff timer (a failing
  pair of streams used to leave an orphaned 500 ms timer behind).
- **The Bands view filed finished work under Now** (`r3/feat-bands-done-in-now`,
  `r2/FS-05`). With no progress reading, a done node landed in "Now"; and after a
  reload the Bands tab ignored the progress reading the Progress tab used. Done
  work now falls to "Later", and both tabs read the same reading.
- **The evidence strip no longer hides its limits** (`r3/feat-evidence-overpromised`).
  A failed load shows an error banner instead of nothing, and a project with more
  than five finished leaves says "Finished work not shown here: N more".
- **Pointers carry the tab's id** (`r2/SEC-18`, client half): the cursor request
  sends `client=<id>`, matching the presence stream, so the server can tell two
  tabs of one user apart. A new test covers the pointer drawing, moving, the
  page's own pointer being skipped, and removal when the member leaves
  (`r3/test-cursor-callback-never-fired`).

## Account, workspace and ideas

- **The sign-in form says when mail is off** (`r1/USER-04`). `GET /auth/config`
  now returns `mailEnabled`; the password login form and forgot-password show a
  notice when no mail is delivered (a dev server logs the link instead), and the
  options stay, annotated rather than hidden.
- **A refused invitation says so** (`r2/USER-16`): a `403` on register shows the
  invitation message in both languages instead of "Could not create account".
- **Inviting says the teammate needs an account first** (`r2/USER-15`).
- **A workspace owner can create another workspace** (`r3/create-workspace-unreachable-nonempty`):
  the create row existed only in the empty state.
- **The Material widgets speak the app's language** (`r3/feat-material-locale-never-set`):
  both `MaterialApp`s set `locale`, `supportedLocales` and the global Material
  delegates, so a date picker or a dialog's "Cancel" reads 取消 in Chinese.
  **Cost: the release web bundle grows by 490 KB raw (+110 KB gzip, 933 KB to
  1.04 MB)**, because the delegates bundle every locale table. Accepted for the
  Chinese strings; trimming it needs a custom delegate and is not done.
- **An idea card's body is tappable** (`r3/idea-card-body-not-tappable`): a tap
  runs the same convert (or opens the converted PRD) as the button; the menu
  keeps its own tap.
- **Server errors are not "network errors"** (`r3/rel-5xx-misclassified-network`):
  a `5xx` answer is mapped to the server-error message; only a failed transport
  reads as a network problem.
- **No more `setState` after dispose** in capture, drop, edit and the admin
  revoke dialog (`r2/FS-08`): four tests, each red without its guard.
- **The board's latest-update note is capped** (`r2/FS-09`): the list answer
  carries at most 200 runes plus an ellipsis, cut on a rune boundary; the full
  note stays in the updates feed.
- **Test coverage that could not fail now can** (`r3/test-dashboard-zero-coverage`,
  `r1/FS-09`): the dashboard and versions pages got failed-load and Retry tests
  (with the raw error text swapped back in, all four go red), and the router's
  magic-link and reset-password routes now assert the token they pass on.

## Theme and landing

- **The light danger chip clears 4.5:1** (`r3/dark-status-pills-collapse`,
  `r3/light-warn-pill-borderline`): its label was cobalt-700 on cobalt-200 at
  4.49:1; it is the ink colour now (13.0:1). The dark chips still share one hue
  by the one-hue decision; [DESIGN.md](../../../DESIGN.md) records it.
- **The dock's keyboard focus is visible** (`r3/no-focus-indicators`). The brand
  ring on the brand-coloured dock measured 1.0:1. `BlockFocus` and
  `PressableBlock` take an optional `ringColor`, the dock passes its foreground
  colour (5.09:1 light, 4.79:1 dark), the current item has a 2 px outline, and
  the current item is announced as selected.
- **The Home numeral at 120 is a named exception** in DESIGN.md
  (`r3/pm-home-numeral-off-lock`), following the Home lock.
- **The landing teaser's mock commit hashes are hex** (`r3/cq-mock-hashes`,
  `r3/feat-landing-fake-hashes-dead-arm`); the diagram's two identical arms are
  one (`r3/cq-dead-if-arms`).

## Test names

- **No test cites the deleted Nuxt app as its contract** (`r1/ARCH-09`,
  `r1/TEST-06`, `r2/TEST-16`). Nine test titles now name the route or string they
  pin, and `leftover_scan_test.dart` fails if the word reappears in a test.
  The `cwd`-sensitivity half of `r2/TEST-16` did not reproduce (no invocation
  that Flutter accepts runs the scans from another directory), so nothing changed.

## Operator notes

- Nothing here needs a server step except the endpoints named above
  (`POST /api/v1/prds/{id}/conversation`, `GET /auth/config` with
  `mailEnabled`, the presence `client` query), all shipped with the server.
- The web bundle is about 110 KB gzip larger (see the Material locale item).

## Round 2: reopened findings and integrator follow-ups

- **Sub-12px text and untokenized spacing** (`r3/sub-12px-text`,
  `r3/type-spacing-unscaled`, reopened by the independent re-check).
  `BlockTag` is 12px/w700 like every other meta line; the minimap's
  six-character band titles keep their shape and gain a tooltip with the
  full title. ShadeChip, FieldCard and BlockBanner pad with `Cobalt.space*`
  tokens, and a regression test pins the token use the sibling tokens have.
- **Brand-on-dark text** (`r3/brand-on-dark-accents-fail`). The roadmap
  pressure label uses `fgBrand` (hue200 on dark) instead of raw
  `palette.brand`, clearing 4.5:1 in both themes.
- **The pivot-seam barrier colour** (`r3/pm-hardcoded-paint-at-pivot-seam`).
  The modal barrier is the named `modalBarrier` token beside
  `castShadowColor` (same value the block-craft plan pinned), and `AuthDoor`
  reads it instead of painting its own black.
- **The line budget is guarded now** (`r3/cq-line-budget-drift`,
  `r3/full-line-budget-issues-missing`). A test walks `lib/` and `test/`,
  fails on any Dart file over 350 lines, and names the dated modularity
  issue the AGENTS.md rule requires — the drift that recurred within hours
  of the sweep cannot come back silently.
- **One read gate instead of ten copies** (`r3/cq-async-branch-copypaste`).
  `readGate(snap, onRetry)` beside `BlockError` replaces the ten inline
  `hasError ? BlockError : BlockLoading` branches.
- **The dock's magic numbers and dead routes** (`r2a`). The clearance is
  `product_dock.dart`'s own constant (height + space10 + space4) instead of
  four private `140`s; deep links to `/prd/:id/revisions|view|export` light
  the dock's Version item, which was inert before.
- **Gap reasons in the reader's language** (`r3/versions-readiness-mixed-language`).
  The audience view's `_GapRow` renders `gapReason` like the rest of the
  page instead of concatenating the server's raw English reason.
- **The stale export preview** (`x/export-preview-stale-after-option-change`,
  found by the c2 worker). Changing the format or audience under an existing
  preview now clears it; a Markdown preview no longer renders under the
  Print header.
- **A blank required field says so** (`r2o`). `BlockFormField` gained
  `required`: confirming with a blank or spaces-only value keeps the modal
  open with `common.fieldRequired` (en + zh) instead of closing silently.
  Marked required only where the server rejects an empty value.
- **The coach stream closes when its panel goes**
  (`x/coach-stream-never-cancelled`, from the c3 worker's notes). `sendCoach`
  takes a `stop` future like the presence and progress streams
  (`_untilStopped`), and the panel completes it in `dispose` — the route
  that holds a model call open the longest no longer leaks it.
- **The PRD page's stage callback is guarded**
  (`x/prd-page-stage-setstate-unguarded`): a stage frame that lands after
  the member left the PRD no longer throws `setState() called after
  dispose()`.
- **The dead self-pointer filter is gone** (integrator, c3 note 9). The
  server composes presence session ids as `user:client`, so the page's
  `sessionId == _clientId` check could never match; the server-side skip of
  the viewer's own pointer is the real guard, and the page now renders
  every cursor the stream delivers, which the retargeted test pins.

## Round 2: tests that only counted

- **Substring greps became behavior tests** (`r3/test-grep-tests-inflate-count`,
  `r2b`). Role change, member removal, rename, ideas edit and workspace edit
  assert the API call and its arguments, not a source substring.
- **The phone viewport and zh finally have widget tests**
  (`r3/test-single-viewport-only`, `r3/test-zh-cn-never-widget-tested`,
  `r2c`). New files only: `phone_layout_test.dart` pumps the four heaviest
  pages at phone width, and five `*_zh_test.dart` files pump landing,
  ideas, project, revisions, audience view and export under zh — the locale
  tables now prove their own keys render.
- **Two analyzer infos and a format drift went** (`r3/test-files-share-format-drift`,
  `r2a`): the null-aware element in `project_page_test.dart`, braces in
  `sse_test.dart`. The analyze baseline drops to 14.
