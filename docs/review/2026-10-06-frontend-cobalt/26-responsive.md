# 26 · Responsive

**Lens:** Narrow-width behavior, breakpoints, mobile layout, fixed-width panes
**Score:** **3 / 10**
**Bottom line:** The dock overlap is not a padding bug: ProductDock's Center child expands to the full bounded height the Scaffold gives a bottomNavigationBar, so the pill renders at the vertical center of the viewport on every signed-in page (verified against shots 04, 06, 09, 11, 16, 19, 20). Beyond that, the web index.html is missing the viewport meta tag, the coach panel and roadmap canvas reserve no bottom space for the dock, several dialogs/sections use fixed pixel sizes that overflow at 390px, and mobile coverage exists for only 2 of 13 routes. Desktop >=900px code discipline (Expanded, Wrap, breakpoints) is good, but shipped responsive behavior is broken at every width.

Reviewed branch `main` at `f185e94`; a skeptic pass re-checked every Critical/High finding — corrections marked [Correction]. Screenshot evidence lives in `./shots/`.

---

## Scope

App chrome (`client/lib/app.dart`, `client/lib/chrome/product_dock.dart`), every page-level layout file (dashboard, workspace, project, ideas, PRD, roadmap, versions stack/commit dialog, coach panel, field card), the web host file (`client/web/index.html`, `client/lib/main.dart`), the local Flutter SDK sources for `Scaffold` and `RenderPositionedBox` layout behavior, the pre-built code map, and the capture index. Shots 04, 06, 09, 11, 16, 19, 20, and 21 were personally inspected at full resolution; the remaining desktop shots are corroborated by the capture index. Mobile coverage at 390px exists only for dashboard ([19](shots/19-dashboard-narrow.png)), ideas wall ([20](shots/20-ideas-wall-narrow.png)), and the resize anomaly ([21](shots/21-narrow-resize-gutter.png)) — a gap that is itself a finding below.

## What is good

- Single consistent breakpoint: `MediaQuery.sizeOf(context).width >= 900` at `dashboard_page.dart:134`, `prd_page.dart:153`, `assembly_step.dart:33`, and narrow branches genuinely exist — dashboard cards stack at `dashboard_page.dart:171-175`, the PRD split becomes a stacked ListView at `prd_page.dart:165-169`.
- Button groups use `Wrap`, not fixed Rows: `project_page.dart:162-180` and `198-234` cannot overflow horizontally at 1024px.
- Ideas-wall grid math is self-consistent with its ListView's 56px horizontal padding: `ideas_page.dart:239` computes `(width - 88) / 3`, `(width - 72) / 2`, `width - 56`, exactly filling the content column at the 1100/760 breakpoints.
- `FieldCard` clips (`field_card.dart:31`, `clipBehavior: Clip.antiAlias`), so the idea poster's deliberately off-edge decorative circle (`ideas_page.dart:292-302`) is safely contained rather than bleeding (clean in [20](shots/20-ideas-wall-narrow.png)).
- `SafeArea` is used for the dock (`product_dock.dart:26`), the top-right utilities (`app.dart:93`), and the auth modal (`auth_door.dart:41`); display type is fluid via a clamp (`cobalt.dart:97`).
- Desktop splits prefer flex over fixed widths: PRD uses `Expanded(flex: 3)`/`Expanded(flex: 2)` (`prd_page.dart:161-162`); dashboard cards use `Expanded` (`dashboard_page.dart:165-167`).

## Findings, worst first

### dock-renders-at-viewport-center — Critical

The lead-reported defect — "at 1440x900 the bottom dock overlays dashboard body content" — is real but misdiagnosed by its symptom. The dock does not sit at the bottom at all: the pill is centered at approximately (720, 440) of a 1440x900 viewport in [04](shots/07-project-dark.png), at (390, 825) of 780x1688 in the narrow [19](shots/19-dashboard-narrow.png)/[20](shots/20-ideas-wall-narrow.png), and mid-screen again after resize in [21](shots/21-narrow-resize-gutter.png) — in every case the vertical midpoint of the screen, with a large empty band below. The cause is a three-part chain: `AppScaffold` passes the dock as `bottomNavigationBar` on a `Scaffold` with `extendBody: true`; `ProductDock.build` wraps the pill in `Center`; and the Scaffold's layout delegate lays the bottomNavigationBar out with width-tightened, height-bounded constraints. `Center` only shrink-wraps when `maxHeight` is infinite; here it is finite, so the `Center` expands to the full viewport height and vertically centers the pill inside it. The pill therefore lands at the viewport's vertical center on every signed-in route, and every page's `bottom: 140` padding is compensating for a dock that was never at the bottom.

Evidence: `client/lib/chrome/product_dock.dart:26-28` (wired at `client/lib/app.dart:135`, `extendBody` at `client/lib/app.dart:71`)

```
return SafeArea(
  minimum: const EdgeInsets.fromLTRB(16, 0, 16, 16),
  child: Center(
    child: Material(
```

**Fix:** Replace `Center` with `Align(alignment: Alignment.bottomCenter, heightFactor: 1.0)` in `ProductDock.build` (heightFactor restores shrink-wrap), or delete `Center` entirely; then re-shoot all pages since every `bottom: 140` assumption re-anchors against a truly bottom dock.

**Verdict:** CONFIRMED

### no-dock-clearance-coach-and-canvas — High

With `extendBody: true`, body content paints behind the dock, but the coach panel and the roadmap canvas view reserve no space for it — the coach composer's bottom padding is only 24 and the roadmap canvas `Column` has no bottom inset — unlike the ListView pages that hardcode `bottom: 140`. Today this is masked by the floating (mid-screen) dock; the moment the `Center` bug is fixed, the coach chat input row and the canvas bottom row slide under the dock. These must land in the same change set as the dock fix.

Evidence: `client/lib/app.dart:71` + `client/lib/features/prd/coach_panel.dart:205` + `client/lib/features/project/roadmap_page.dart:226` (dashboard reserves 140 at `dashboard_page.dart:137`, for comparison)

```
padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
```

**Fix:** Give CoachPanel's root Column and the roadmap canvas Column a bottom padding equal to `ProductDock.height + 16 + MediaQuery.paddingOf(context).bottom`, or drop `extendBody` and let the Scaffold inset the body above the dock automatically.

**Verdict:** CONFIRMED

### missing-viewport-meta — High

`client/web/index.html` has no `<meta name="viewport">` tag at all — only description and apple meta tags — so on a real phone the page renders at the browser's ~980px fallback layout width instead of the device width. Every 390px assumption, the breakpoint, and all touch targets are defeated on actual mobile hardware; the 390px shots look correct only because CDP emulation forces device metrics. This also plausibly contributes to the resize gutter in [21](shots/21-narrow-resize-gutter.png).

Evidence: `client/web/index.html:19-36`

```
<meta name="description" content="Track the work. Refine the ideas. Ship better PRDs.">
...
<meta name="mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-status-bar-style" content="black">
<meta name="apple-mobile-web-app-title" content="insideout">
```

**Fix:** Add `<meta name="viewport" content="width=device-width, initial-scale=1.0">` to `client/web/index.html` head and rebuild.

**Verdict:** CONFIRMED

### window-resize-canvas-stuck — Medium

Shrinking the browser window after boot leaves the Flutter canvas stuck at an intermediate width with a white gutter on the right (~245px column in [21](shots/21-narrow-resize-gutter.png)); dispatching a resize event does not recover it. Booting directly at 390px is fine, so this is a resize-handling path, not a layout path.

Evidence: `docs/review/2026-10-06-frontend-cobalt/shots/21-narrow-resize-gutter.png`

**Fix:** Add the viewport meta (previous finding), then verify Flutter's engine resize handling against a `window.resizeTo` pass in `scripts/sim/web_driver.py`; if it persists, file against the engine with a minimal repro since `main.dart` has no custom sizing code.

### top-utility-row-no-width-budget — Medium

The top-right utilities `Row` is `Positioned` with `mainAxisSize: min` and no width constraint, stacking page actions plus four icon buttons. That is fine for the bare four icons (~180px), but the roadmap page injects a three-segment `SegmentedButton` as actions, making the row ~400-460px wide — which cannot fit a 390px viewport — and no narrow roadmap shot exists to prove it survives.

Evidence: `client/lib/app.dart:90-130`

**Fix:** Constrain the utility Row (e.g., `Flexible` + `Wrap`, or move page actions next to the painted title), and add a 390px capture of `/projects/:id/roadmap` to the sim suite to gate it.

### commit-dialog-fixed-width-420 — Medium

The commit dialog hardcodes a content width of 420px (`versions_commit_dialog.dart:52`), exceeding the ~310px available inside a Material `AlertDialog` (default 40px horizontal insets) on a 390px viewport — a guaranteed horizontal overflow at the single most important first-run action (first commit).

Evidence: `client/lib/features/prd/versions_commit_dialog.dart:51-53`

```
width: 420
```

**Fix:** Replace with `width: min(420, MediaQuery.sizeOf(context).width - 96)` or drop the `SizedBox` width entirely and let the dialog size to its insets.

### versions-stack-fixed-height-580 — Medium

The commit stack pins a 580px-high `Stack` whose front `FieldCard` holds an unbounded `Column` of readiness gaps: a long gap list overflows 580px with no scroll, and the fixed block dominates or is scroll-chopped on short viewports.

Evidence: `client/lib/features/prd/versions_stack.dart:60-63`

```
height: 580
```

**Fix:** Make `_FrontCard`'s Column scrollable (wrap the gaps list in `Expanded` + `SingleChildScrollView` inside a `LayoutBuilder`) or size the stack from the viewport (`min(580, availableHeight)`) instead of a constant.

### prd-narrow-fixed-heights-640-520 — Medium

Below 900px the PRD page stacks the desk and coach inside fixed 640px/520px `SizedBox`es, decoupled from both content height and viewport height — content taller than the box clips, shorter leaves dead space, and neither adapts to a short phone.

Evidence: `client/lib/features/prd/prd_page.dart:165-169`

**Fix:** Let `_prdColumn` keep its own `ListView` with natural height (remove the `SizedBox` heights) or use `LayoutBuilder`-driven fractions of the viewport.

### dead-dock-height-constant-and-magic-140s — Low

`ProductDock` declares a dead constant that no file references, while six pages independently hardcode `bottom: 140` as the dock clearance — two disconnected guesses at the same number, so a dock resize silently desynchronizes every page.

Evidence: `client/lib/chrome/product_dock.dart:15`

```
static const height = 88.0;
```

**Fix:** Reference `ProductDock.height + 16 + MediaQuery.paddingOf(context).bottom` from a shared scaffold/page helper, or drop `extendBody` and remove all manual clearance.

### paint-title-right-padding-88-vs-icons-180 — Low

The painted title reserves only 88px on the right, but the top-right utility Row (four icon buttons plus page actions) measures ~180-460px, so long workspace/project names slide beneath the icons, which paint on top.

Evidence: `client/lib/app.dart:76-86`

**Fix:** Reserve the measured utility width (or use a `Row` with `Spacer` between title and utilities) instead of the 88 constant; cap the title with `maxLines: 1, overflow: TextOverflow.ellipsis`.

### narrow-shot-coverage-gaps — Low

Mobile evidence exists only for dashboard ([19](shots/19-dashboard-narrow.png)), ideas wall ([20](shots/20-ideas-wall-narrow.png)), and the resize anomaly ([21](shots/21-narrow-resize-gutter.png)); roadmap, workspace, project, versions, PRD, and profile were never captured at 390px — leaving the highest-risk surfaces (SegmentedButton top bar, fixed 580/640 sections, 420px dialog) unverified at narrow width. This gap is itself a finding.

Evidence: `docs/review/2026-10-06-frontend-cobalt/shots/`

**Fix:** Extend `scripts/sim/web_driver.py`'s narrow pass to walk every dock route (workspace, project, roadmap, prd desk, versions, profile) at 390x844.

### fixed-240-side-panes — Low

The roadmap band/progress views pin a 240px leading pane; safe at 1024px, but it consumes 72% of a 390px content column, squeezing node titles to ~80px.

Evidence: `client/lib/features/project/roadmap_bands.dart:120` + `client/lib/features/project/roadmap_progress_view.dart:81`

**Fix:** Scale the pane with a clamp (e.g., `max(150, width * 0.25)`) or stack the band label above the cards below 600px.

## Recommendations

- Fix the dock root cause first (`Align` swap in `ProductDock.build`), and re-shoot pages 04, 06, 09, 11, 16, 19, 20 immediately — every "overlapping dock" symptom becomes a measurable bottom-clearance question once the pill truly anchors at the bottom.
- Land the viewport meta tag in `client/web/index.html` in the same change set; it is one line and unblocks all real-device behavior.
- Ship the coach-panel and roadmap-canvas bottom clearance together with the dock fix, not after — they are latently broken today and only masked by the floating pill.
- Add 390px captures for the remaining routes (roadmap, workspace, project, versions, PRD, profile) to the sim suite so the fixed-width panes, dialog widths, and utility-row budget are gated by evidence.
- Replace the magic `bottom: 140` / dead `height = 88.0` pair with one shared clearance helper once the dock's true geometry is settled.

## What this role would do first

Swap `Center` for `Align(alignment: Alignment.bottomCenter, heightFactor: 1.0)` in `ProductDock.build` (`client/lib/chrome/product_dock.dart:28`) — one line — add the `<meta name="viewport">` tag to `client/web/index.html` in the same hour, then re-shoot pages 04, 06, 09, 11, 16, 19, and 20. That single change converts every "overlapping dock" symptom from a mystery into a measurable bottom-clearance question, at which point the coach/canvas clearance findings (`coach_panel.dart:205`, `roadmap_page.dart:226`) become a small, well-defined cleanup rather than a forensic exercise.
