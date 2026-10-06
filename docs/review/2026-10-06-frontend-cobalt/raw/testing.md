## Testing review — client suite

Commands run from `<repo>/client` on branch `main` (f185e94), read-only.

### `flutter analyze --no-fatal-infos` — exact tail

```
Analyzing client...

   info • Don't use 'BuildContext's across async gaps, guarded by an unrelated 'mounted' check. Guard a 'State.context' use with a 'mounted' check on the State, and other BuildContext use with a 'mounted' check on the BuildContext • lib/features/workspace/workspace_page.dart:368:28 • use_build_context_synchronously
   info • Unnecessary use of multiple underscores. Try using '_' • test/prd_actions_test.dart:77:24 • unnecessary_underscores
   info • Unnecessary use of multiple underscores. Try using '_' • test/prd_actions_test.dart:81:28 • unnecessary_underscores
   info • Unnecessary use of multiple underscores. Try using '_' • test/prd_desk_audience_test.dart:49:53 • unnecessary_underscores
   info • Use the null-aware marker '?' rather than a null check via an 'if'. Try using '?' • test/project_page_test.dart:71:7 • use_null_aware_elements

29 issues found. (ran in 3.0s)
```

Zero errors; all 29 remaining issues are infos (23 in `lib/` — mostly
`use_build_context_synchronously` in the PRD/project/workspace pages — and 4
inside `test/` itself). The build gate is clean but the tail shows test files
are not held to the analyzer either.

### `flutter test` — exact tail

```
00:07 +85: <repo>/client/test/auth_form_magic_test.dart: login form switches to magic-link mode and back
00:08 +87: <repo>/client/test/versions_page_test.dart: versions page shows gaps and performs a carrying commit
00:08 +89: <repo>/client/test/versions_page_test.dart: versions page shows gaps and performs a carrying commit
00:08 +92: <repo>/client/test/versions_page_test.dart: versions page shows gaps and performs a carrying commit
00:08 +93: <repo>/client/test/versions_page_test.dart: versions page shows gaps and performs a carrying commit
00:08 +94: All tests passed!
```

94/94 tests pass in ~9s of test time (164s wall including pub). The suite is
green and stays green fast enough to run on every change — the problem is not
health, it is eyesight.

### What the collection actually is

34 files, ~2960 lines. Behaviorally strong where it is strong:
`api_token_refresh_test.dart` binds a real `HttpServer` on 127.0.0.1 so the
genuine Dio interceptor stack crosses real wire I/O — 401 → refresh → retry,
failed refresh clears the session, retried 401 does not loop, and concurrent
401s share one refresh. `sse_test.dart` (215 lines, the deepest file) covers
the SSE buffer parser, delta frames, error frames, and a five-case refresh
matrix for the coach stream. `coach_panel_test.dart` drives a finding frame
through `applyCoachFrame` and asserts both critic findings render.
`roadmap_page_test.dart`, `roadmap_presence_test.dart`, and
`roadmap_canvas_test.dart` cover the server-truth progress buckets, streamed
snapshot replacement, reconnect-after-drop, and canvas/minimap toggling.
`auth_form_magic_test.dart` covers the magic-link and code modes.
`product_dock_test.dart` covers the dock's unfinished-step routing. Token
refresh, coach SSE, roadmap canvas presence (chips), dock navigation, and
auth form modes are all mapped to real tests — this part of the advertised
behavior is genuinely covered.

The weak tier is honestly labeled: `feature_wiring_test.dart`'s own header
admits it is a "LINT-TIER BACKSTOP only … they prove presence, not behavior"
— 37 substring greps — and `leftover_scan_test.dart` greps `lib/` for retired
Ink & Seal hexes. Together they are ~2 of 34 files but they pad the count
that reads as "94 tests."

### The blind spot that mattered: chrome geometry

Both coordinator-confirmed visual defects sailed through this suite, and after
reading every test file I can say precisely why.

**Auth modal header collision** (shot 02: the cobalt dot + "InsideOut"
wordmark render above the modal card, colliding with the hero buttons). The
wordmark lives outside the card by construction — `auth_door.dart:50-70`
puts dot, wordmark, and greeting in the same Column *above* the card at
line 72. The only test that renders landing + auth is `widget_test.dart`,
which sets `Size(1200, 5000)`: at 5000px tall the centered modal is nowhere
near the hero, and `expect(find.text('InsideOut'), findsWidgets)` (line 31)
is an unbounded plural that passes with both wordmarks present.
`auth_form_magic_test.dart` pumps `AuthDoor` as the entire `home:` — the
modal over an empty page. No golden test exists anywhere in the suite (grep
for `matchesGoldenFile`: zero hits). Nothing, widget or golden, could have
failed.

**Dock overlapping the dashboard card** (shot 04: the floating pill dock
clips the chapter card's body text at 1440x900). The real shell is
`app.dart:71` `extendBody: true` plus `app.dart:135` `bottomNavigationBar:
const ProductDock()` — the body extends *behind* the dock, so every page must
reserve bottom space. `product_dock_test.dart:17` mounts the dock as a
`bottomNavigationBar` of a bare `Scaffold(body: Text(label))` with no
`extendBody` and a body that has nothing to overlap. The harness removes the
defect by construction. And there is no `dashboard_page_test.dart` at all —
the signed-in home surface is untested, including its raw error branch
(`dashboard_page.dart:98` `Text('${snap.error}')`, which violates the
documented `localizedError` standard and has no test to catch it;
`versions_page_test.dart` likewise never pumps the error branch at
`versions_page.dart:73`).

Other coverage gaps, mapped to the advertised behaviors: the presence fake
declares `onCursor` but never fires it, so roadmap cursors are untested;
profile, settings, revisions, forgot/reset/magic-link, and the export page
widget have no tests (export has only the pure-function
`export_format_test.dart`); no widget test runs in the shipped default zh-CN
locale (all hardcode `'en-US'`); and every realistic surface is exactly
1400x900 — no mobile-class viewport exists, which is a risky posture heading
into a full chrome rewrite.

Could widget/golden tests have caught the two defects? Yes, specifically:
the modal defect needs a realistic-viewport golden or a card-containment
geometry assertion (a plain widget test would not throw — the layout is a
Stack/Column, not a flex overflow); the dock defect needs an `AppScaffold`-
level test with `extendBody: true` asserting the dock rect does not intersect
visible content. Neither requires new infrastructure — only pumping what the
app actually mounts, at the size it actually runs.

### The three tests that would have caught real bugs

1. **Golden test: LandingPage + AuthDoor at 1440x900, zh-CN default, light
   and dark** — catches the confirmed wordmark-outside-the-card collision
   (and lands the pivot's zero-radius chrome with a baseline from day one).
2. **Widget test: AppScaffold with `extendBody: true` and a content-filled
   body at 1440x900, asserting the ProductDock rect does not intersect the
   last visible content rect** — catches the confirmed dock-clips-card defect
   and permanently encodes the scaffold/page bottom-padding contract the
   pivot's sticky top nav will renegotiate anyway.
3. **dashboard_page_test.dart error path: API fake throws, assert the rendered
   text resolves through the `error.*` l10n keys and `find.textContaining(
   'Exception')` finds nothing** — catches the raw `'${snap.error}'` on the
   home surface (and the identical branch in versions_page_test) and locks
   the `localizedError` standard where users actually land after login.

### Hygiene notes

`sse_test.dart:48-59` and `roadmap_presence_test.dart:39-59` are
tab-indented (mirroring the `sse.dart:96-115` drift noted elsewhere);
analyzer infos fire inside `prd_actions_test.dart`, `prd_desk_audience_test.
dart`, and `project_page_test.dart`. `cobalt_test.dart` bakes current cobalt
hexes as expectations and `leftover_scan_test.dart` greps literal hexes —
both will detonate en masse when the neo-brutalist pivot swaps the palette,
converting a design change into a red suite with no mapping from old law to
new. Rewriting those two as role-based checks (forbid raw `Color(0x` outside
the theme directory; assert geometry/role tokens instead of hexes) before
the pivot starts is the cheapest insurance in this entire report.
