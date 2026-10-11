# Client fidelity fixes

The block-craft fidelity review of 2026-10-08 left a list of medium and low
findings ([record](../../issues/2026-10-08-block-craft-fidelity-followups.md)),
and the Nuxt-era canvas records left two items with Flutter equivalents
([record](../../issues/2026-07-26-roadmap-canvas-adversarial-followups.md)).
Each one below is a test that fails against the unfixed code, then the fix.
The 390px defects the web walk found are in
[web-walk-and-phone-layouts.md](web-walk-and-phone-layouts.md), and the
Build MVP bug is [BUG-022](../../issues/2026-10-10-bug-022-build-mvp-wrote-nothing.md).

## Theme and surfaces

| Finding | Fix | Commit |
| --- | --- | --- |
| Four stock Material popup menus (rounded, blurred, ripples) | One `popupMenuTheme` (square, 2px border, no elevation, no ripple); the PRD desk keeps only its trigger face | `fd7e59e` |
| `showDatePicker` rendered round Material chrome | One `datePickerTheme`: flat 2px dialog, square day and year cells, 2px cobalt "today" outline | `ff34e00` |
| Idea poster and version peeks filled whole surfaces with cobalt-950 (reserved for status chips) | Existing tokens: the poster fades raised to sunken, the peeks are sunken. Light is unchanged; dark drops the blue slab | `47f9734` |
| Bands view's Next band and Progress view's needs-deadline band did the same; Progress items were bare containers | Sunken surface for both bands; Progress items use `FieldCard` like Bands | `de817fe` |

## Workspace

| Finding | Fix | Commit |
| --- | --- | --- |
| Quick-capture was a stock flat FAB with no name | `QuickCaptureFab`: a cast-and-press block on `PressableBlock`; "New project" is both tooltip and screen-reader name | `9cf00cb` |
| Pull-to-refresh painted Material's round spinner | `RefreshIndicator.noSpinner` with `BlockRefresh` overlaying the square loader; the gesture is unchanged | `f2a77e4` |
| A worker suspected a board shorter than the screen could not be pulled | Measured: it can (a primary `ListView` is always scrollable). A test now pins it | `eac9573` |

## Coach

| Finding | Fix | Commit |
| --- | --- | --- |
| API failures collapsed to the generic error, and the countdown copy was picked by looking for the English word "unavailable" | The exception object is kept, so `localizedError` maps status and code; `isCoachUnavailable` reads the `CIRCUIT_OPEN` code. This also fixes the second 503 wording the server sends | `c2fd281` |
| A backoff failure landing after the user left threw `setState() after dispose()` and left a timer running | Mounted guard before the state change and the countdown | `0fd2547` |

## PRD desk

| Finding | Fix | Commit |
| --- | --- | --- |
| The desk and the revisions table printed raw ISO timestamps | `timeAgo`: relative for two weeks ("3d ago" / "3 天前"), a plain date after that | `c435fd8` |

## Project and roadmap

| Finding | Fix | Commit |
| --- | --- | --- |
| Minimap marks were hardcoded English; column titles at 9px and the cursor label at 10px, under the 12px floor | `roadmap.minimap.status.*` keys and the theme's `bodySmall`; the row scrolls horizontally on a phone and stays centered while it fits | `53e509d` |
| The Evidence timeline printed the enum (`agent_proposal`) and raw ISO ages | Localized kinds (two agent kinds added to the key family) and `timeAgo` | `ad7a688` |
| After clearing a deadline the button rendered empty | A cleared deadline reads like unset ("Set deadline") | `44e7003` |
| Deleting a node silently deleted its whole branch (`parent_id` cascades) | The confirm states "This deletes N nodes" (1 + descendants) when the node has any; a lone node keeps its wording | `e8a06ff` |
| The Evidence flash and the minimap jump ignored the OS reduce-motion setting | Both consult `inkReduceMotion`: the flash lands instantly, the jump is a `jumpTo` | `8865ce5` |
| Arrow keys scroll on web only; on iOS and Android they move focus, so a keyboard user could not scroll a tall band | The canvas binds ArrowUp/ArrowDown to `ScrollIntent` (PageUp/PageDown already worked and are now pinned) | `bb12aa1` |
| No test covered the Evidence write flows | `project_flows_test.dart`: edit and delete project, post, edit and remove an update, repo save and sync, the roadmap button | `d2321aa` |

## How it was verified

- **Red first.** Each commit message records how the new test failed against
  the unfixed code (for example twelve coach cases, six date-picker cases, the
  exact `setState() after dispose()` assertion). Where a fix is a token swap
  that leaves light mode untouched, the light cases pass by design as a guard.
- **Gate:** `flutter pub get --enforce-lockfile`, `flutter analyze
  --no-fatal-infos` (baseline of 16 infos, unchanged), `dart format
  --set-exit-if-changed`, and `flutter test`; CI's `client (flutter)` job
  repeats it with a web build.
- **Real browser:** the project, roadmap (Progress, Bands, Canvas), PRD and
  Versions pages were walked at 1440x900 and 390x844 against a local server
  ([web walk](web-walk-and-phone-layouts.md)).

## Reviewed and kept

- The boot splash keeps its round spinner: the block-craft record named it a
  plan-protected exception, and it is the full-screen gate that holds the
  router until the first frame and the fonts are ready
  ([boot loading page plan](../../plans/2026-08-25-boot-loading-page.md)).
- Delete buttons use the Cobalt danger variant, which is blue by design.

## Operator notes

None. Nothing here is deployed; the hosted bundle predates these changes.

*Superseded 2026-10-11: the hosted bundle now carries these changes — the
2026-10-11 owner deploy shipped both services
([record](../2026-10-11-owner-deploy-and-mail-truth.md)).*
