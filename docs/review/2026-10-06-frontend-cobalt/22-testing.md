# 22 · Testing

**Lens:** Test coverage, harness fidelity, golden/widget test gaps
**Score:** **6.5 / 10**
**Bottom line:** The suite is honestly built and green (94/94, 0 analyze errors) with genuinely excellent API/SSE/auth-mechanics coverage, but it is structurally blind to chrome and layout: no golden tests exist, the only landing test inflates its surface to 1200x5000, the dock test mounts ProductDock without the real extendBody:true scaffold, and whole signed-in surfaces (dashboard, profile, settings, revisions, export widget) have zero tests — which is exactly why both confirmed visual defects shipped green.

Reviewed branch `main` at `f185e94`; a skeptic pass re-checked every Critical/High finding — corrections marked [Correction]. Screenshot evidence lives in `./shots/`.

---

## Scope

The full client test suite under `client/test/` (34 files, ~2,960 lines) on branch `main` at `f185e94`, plus every file it exercises under `client/lib/`. Commands run from `client/`, read-only: `flutter analyze --no-fatal-infos` (0 errors, 29 infos — 4 of them inside `test/` itself) and `flutter test` (94/94 passing, ~9s of test time). Every test file was read, and the two coordinator-confirmed visual defects (`shots/02-login-modal.png`, `shots/04-dashboard-light.png`) were traced back through the suite to explain why each sailed through green.

## What is good

- `client/test/api_token_refresh_test.dart` — binds a real `HttpServer` on loopback so the genuine Dio interceptor stack crosses real wire I/O: 401 → refresh → retry, failed refresh clears the session, retried 401 does not loop, and concurrent 401s share a single refresh.
- `client/test/sse_test.dart` — the deepest file (215 lines): buffer splitting, delta frames, error frames, and a five-case USER-10 refresh matrix pinning no-infinite-loop semantics for the coach stream.
- `client/test/roadmap_page_test.dart` + `roadmap_presence_test.dart` + `roadmap_canvas_test.dart` — server-truth progress buckets, streamed snapshot replacement, reconnect-after-drop with capped backoff, and canvas/minimap toggling; the advertised SSE/progress behaviors are the best-covered area of the client.
- `client/test/routes_test.dart:23-47` pins the exact route table set, so any added/removed surface fails a test and forces conscious maintenance.
- `client/test/errors_l10n_test.dart:13` pins the status→l10n-key mapping with an identity translator, and `client/test/l10n_test.dart` pins zh/en key parity — the error-localization contract is one of the few documented standards that is actually test-guarded.

## Findings

### test-dock-harness-hides-overlap — High

The dock test mounts ProductDock in a bare Scaffold without `extendBody` and with a one-line `Text` body, so the real overlay geometry that clips the dashboard card is structurally impossible to observe. In production the body extends *behind* the dock (`client/lib/app.dart:71` `extendBody: true`, `client/lib/app.dart:135` `bottomNavigationBar`), and the confirmed defect is the floating dock clipping the chapter card's body text at 1440x900 (`shots/04-dashboard-light.png`); the harness lays the body out above the dock over a `Text` with nothing to overlap — the defect class can never fail a test.

**Evidence:** client/test/product_dock_test.dart:17

```
Widget page(String label) => Scaffold(body: Text(label), bottomNavigationBar: const ProductDock()); — vs the real shell client/lib/app.dart:71 `extendBody: true,` with client/lib/app.dart:135 `bottomNavigationBar: signedIn ? const ProductDock() : null,`
```

**Fix:** Mount ProductDock through AppScaffold (or replicate extendBody:true plus a realistic body) and assert the dock rect does not intersect the last visible content rect at 1400x900; keep the existing nav-logic tests but stop treating them as chrome coverage.

**Verdict:** CONFIRMED — product_dock_test.dart:17 mounts a bare Scaffold without extendBody/bottomNavigationBar, so the harness structurally cannot expose the mid-screen overlap it is meant to guard.

### test-no-goldens-modal-collision — High

The only landing+auth widget test inflates the surface to 1200x5000 and the suite contains zero golden tests, so the confirmed auth-modal header collision is invisible to CI. At a real 1440x900 viewport the cobalt dot and "InsideOut" wordmark render above/outside the modal card and collide with the landing hero buttons (`shots/02-login-modal.png`); at 5000px tall the centered modal lands far below the hero and the suite stays green forever. Grep for `matchesGoldenFile` across `client/test` returns no matches.

**Evidence:** client/test/widget_test.dart:12

```
await tester.binding.setSurfaceSize(const Size(1200, 5000)); — grep for matchesGoldenFile across client/test returns no matches.
```

**Fix:** Add golden tests (or explicit geometry assertions: the wordmark rect must sit inside the modal card rect) for LandingPage + AuthDoor at 1440x900 in both themes and the zh-CN default locale.

**Verdict:** CONFIRMED — widget_test.dart:12 setSurfaceSize(Size(1200, 5000)) inflates the viewport past every breakpoint; zero golden tests exist anywhere in the repo.

### test-dashboard-zero-coverage — Medium

The dashboard — the signed-in home surface — has no test file at all, and its raw error rendering that violates the documented `localizedError` standard ships unguarded. Any API failure on the home page shows the user a raw exception string — the exact anti-pattern `localizedError(e, l10n.t)` exists to prevent — and no test will ever flag it. Profile, settings, revisions, and the export page widget have the same zero-coverage status (export has only the pure-function `export_format_test.dart`).

**Evidence:** client/lib/features/dashboard/dashboard_page.dart:98

```
if (snap.hasError) return Center(child: Text('${snap.error}')); — client/test/ has no dashboard* file (34 files listed; none import dashboard_page.dart).
```

**Fix:** Write dashboard_page_test.dart with a failing API fake and assert the rendered text resolves through the l10n error keys (and that `find.textContaining('Exception')` finds nothing); add the same error-path assertion to versions_page_test.dart, whose happy path is covered while its raw-error branch (`client/lib/features/versions/versions_page.dart:73`) is not.

### test-cursor-callback-never-fired — Medium

The presence-stream fake accepts the `onCursor` callback but never invokes it, so multiplayer cursor rendering — an advertised roadmap-canvas behavior — has no behavioral coverage. A regression in cursor drawing (position translation, avatar rendering) ships silently; only the presence-chip reconnect path is tested.

**Evidence:** client/test/roadmap_presence_test.dart:45

```
Future<void> presenceStream(
    String projectId,
    String clientId,
    void Function(List<PresenceEntry> entries) onChange, {
    void Function(String sessionId, String name, double x, double y)? onCursor}) async { — The body only calls onChange([...]); onCursor is never fired in any test.
```

**Fix:** Extend the fake to call `onCursor('other', 'mover', x, y)` and assert a cursor widget renders at the expected local position inside RoadmapCanvas.

### test-loose-plural-matcher — Medium

The widget test asserts `findsWidgets` (plural, unbounded) for 'InsideOut' after opening the auth door — an assertion that passes in the buggy two-wordmark state the screenshot confirmed. The modal wordmark escaping the card increases the wordmark count and the loose plural matcher keeps passing; the assertion documents presence, not placement.

**Evidence:** client/test/widget_test.dart:31

```
expect(find.text('InsideOut'), findsWidgets);
```

**Fix:** At a realistic surface size, assert the modal wordmark's rect is contained within the modal card's rect (`tester.getRect` on both), or replace with a golden.

### test-grep-tests-inflate-count — Medium

Thirty-seven substring greps over source files are counted among the 94 passing tests, inflating the apparent coverage of behavior the suite does not exercise. The file is honestly labeled ("LINT-TIER BACKSTOP only … they prove presence, not behavior"), but a refactor that keeps a verb string while breaking its call site (wrong arguments, dead branch) keeps the grep green while the feature is broken; readers see "94 tests" and over-trust it.

**Evidence:** client/test/feature_wiring_test.dart:12

```
this file is a LINT-TIER BACKSTOP only — substring assertions
// over source files ... they
// prove presence, not behavior.
```

**Fix:** Keep the file (it is honestly labeled) but report widget/behavioral test counts separately in docs, and add behavioral tests for any verb that currently only has a grep (e.g. settings page mutations have no widget test).

### test-single-viewport-only — Medium

No widget test runs at a mobile-class viewport — every realistic surface is exactly 1400x900 except the 1200x5000 outlier — so the layout is only verified at one desktop size. With the neo-brutalist pivot rewriting chrome and component geometry, any small-viewport breakage (overflow, dock covering entire width) ships untested; the pivot's press-the-block motion changes make this the worst possible moment to have a single-viewport suite.

**Evidence:** client/test/widget_test.dart:12

```
grep setSurfaceSize across client/test: 17 hits, 16 of them "const Size(1400, 900)" (e.g. roadmap_page_test.dart:66, product_dock_test.dart:48); zero hits at 390x844 or any width under 1200.
```

**Fix:** Add a 390x844 variant for the top five surfaces (landing, dashboard, workspace, prd desk, roadmap) and assert no overflow exception via `tester.takeException()`.

### test-zh-cn-never-widget-tested — Low

Every widget test hardcodes locale 'en-US'; the shipped default zh-CN UI — the locale in which both confirmed defects were captured — is never widget-tested. CJK line-breaking and longer translated strings cause overflow or collision only in zh-CN (as `shots/02-login-modal.png` shows) and nothing catches it.

**Evidence:** client/test/auth_form_magic_test.dart:17

```
ChangeNotifierProvider(create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US')), — identical pattern in 20+ files; only appearance_test.dart and l10n_test.dart touch zh-CN, at unit level.
```

**Fix:** Parameterize one representative test per top page to run in zh-CN, or fold it into the golden matrix from test-no-goldens-modal-collision.

### test-files-share-format-drift — Low

Test files share the lib's formatting drift — tab-indented blocks inside `sse_test.dart` and `roadmap_presence_test.dart`, plus analyzer infos raised inside `test/` (`prd_actions_test.dart`, `prd_desk_audience_test.dart`, `project_page_test.dart`). Mixed-indentation diffs churn reviews and signal that test code is held to a lower bar than lib code; infos in test/ ride along in every analyze report, training people to ignore analyzer output.

**Evidence:** client/test/sse_test.dart:48

```
	test('readProgressSse applies a progress frame as the assembled payload', () { — analyze tail: "info • Unnecessary use of multiple underscores ... • test/prd_actions_test.dart:77:24 • unnecessary_underscores" and "info • Use the null-aware marker ... • test/project_page_test.dart:71:7".
```

**Fix:** Run `dart format` over `test/` (matches the `sse.dart:96-115` cleanup already owed) and fix the four test-side analyzer infos.

### test-literal-hex-guard-brittle — Low

The brand-leftover test greps for the retired Ink & Seal palette by literal hex strings, a brittle pattern that will need a rewrite when the cobalt→neo-brutalist pivot changes the token vocabulary. During the pivot, cobalt hex literals become the *new* forbidden set, but the test's literal-list shape encourages the same whack-a-mole; `client/test/cobalt_test.dart:8-12` similarly bakes current hexes as expectations and will fail en masse the moment the pivot lands.

**Evidence:** client/test/leftover_scan_test.dart:15

```
forbidden = <String>[ ... '0xFFC8402F', '0xFFE4EAE4', '0xFF161A17', 'celadon', 'vermilion', ]
```

**Fix:** When the pivot starts, replace literal-hex greps with a role-based check (e.g. forbid direct `Color(0x` in lib/features, require all colors via Cobalt roles) so the test guards the law, not the palette.

## Recommendations

- Add a scaffold-level test that pumps AppScaffold (`extendBody: true`) with realistic content at 1440x900 and asserts the ProductDock rect does not intersect visible page content — this is the test that catches the confirmed dock-overlap defect class.
- Add golden tests for landing + auth door, dashboard, and workspace at 1440x900 and 390x844 in both themes, starting with the default zh-CN locale; commit goldens to the repo.
- Write dashboard_page_test.dart with an error-path case that pins `localizedError` on the home surface, and extend versions_page_test.dart with the same assertion for its raw-error branch.
- Rework product_dock_test.dart to mount the dock the way app.dart does (through AppScaffold), keeping the existing navigation assertions as routing tests.
- Extend the roadmap presence fake to fire `onCursor` and assert cursor rendering at translated coordinates on RoadmapCanvas.
- Add one mobile-viewport (390x844) variant per top-five page with a `takeException()` overflow assertion, and run one widget test per page in zh-CN.
- Report behavioral/widget test counts separately from lint-tier greps in docs so "94 tests" is not read as behavior coverage.

## What this role would do first

Add a 1440x900 golden test — or, faster to land, a card-containment geometry assertion using `tester.getRect` — for `LandingPage` + `AuthDoor` (`client/lib/features/auth/auth_door.dart`) in the default zh-CN locale, as a new `client/test/landing_auth_golden_test.dart`. It is the cheapest test that would have caught the confirmed modal-header collision in `shots/02-login-modal.png`, and it lands the neo-brutalist pivot's zero-radius chrome with a baseline from day one.
