## Fullstack review — client behavior, endpoint coverage, and env contract

Scope: `client/lib/router.dart`, all feature pages at action level, `api/client.dart`
+ `api/requests.dart`, `chrome/product_dock.dart` + `chrome/loop_focus.dart`, the
`main.dart` → `app.dart` boot chain, `pubspec.yaml`, and
`docs/usage/local-development.md`. I traced every named control: the six dock items,
idea capture and convert-to-PRD, the commit dialog, the export download path,
invite-code join, and the admin signup-mode toggle. Verdict: the API surface is
complete and the configuration contract is honest, but the client has a systemic
silent-failure pattern on write actions and a dock that sometimes navigates to the
wrong project.

### What checks out

**Endpoint coverage is complete.** Every affordance I traced maps to a real
`ApiClient` method: dock items resolve to `/workspace/:id/ideas`, `/prd/:id`,
`/prd/:id/versions`, `/projects/:id/roadmap`, and the project page (the Evidence
surface); idea capture → `createIdea`, convert → `convertIdea` (which returns the
new `prdId` and navigates to it); commit dialog → `commitPrd` with the readiness
carry list; export → `exportPrd` with format/audience query params; join →
`joinWorkspace`; admin → `setSignupMode`/`createInvitation`/`revokeInvitation`. I
found no place where the UI asks for something the client cannot send. The
reference-design "UI asks, API never sends" hole does not exist here.

**The env contract is coherent.** `session/session.dart:7-10` reads
`String.fromEnvironment('API_BASE', defaultValue: 'http://127.0.0.1:8080/api/v1')`;
`local-development.md:45,121` documents the identical `--dart-define`; the
Dockerfile bakes `ARG API_BASE=/api/v1` for same-origin nginx. Code, docs, and
deploy agree.

**Token refresh is the best code in the client.** Single-flight refresh, a `/auth/*`
carve-out so bad credentials surface untouched, and interceptor-level resolve-retry
(`api/client.dart:47-83`). The SSE paths even re-read the rotated token on retry.
The admin signup-mode toggle works end to end and re-reads its state after writing.

### What fails

**Every primary write action can fail silently (FULL-01, High).** The pattern
repeats across the loop's spine: `ideas_page._captureNow`, idea convert,
`versions_page._commitDialog`, dashboard create/join, `prd_page._save`, repo
save/sync on the project page — all `await api.…` inside button callbacks with no
try/catch. A 409, 422, or dropped connection produces an unhandled async error in
the logs and *nothing* on screen: no SnackBar, and `_reload()` never runs. The
worst instance is the version commit: the user taps 提交版本, the server rejects the
commit, and the UI offers no evidence either way. `project_page._decide` already
demonstrates the correct pattern (catch → `localizedError` → SnackBar); the fix is
to extract it once and apply it everywhere.

**The dock can open the wrong workspace's project (FULL-02, High).** `ProductDock._go`
resolves the project as `focus.projectId ?? _projectOnRoute(context)`
(product_dock.dart:91), and `IdeasPage._seedProject` early-returns whenever any
projectId is already set (ideas_page.dart:38) — nothing clears or re-keys it when
the user crosses workspaces. From workspace B's ideas wall, dock "Roadmap" routes
to workspace A's roadmap. The dock is the product's primary navigation; a
navigation control that crosses workspace boundaries on a labeled button fails the
"does each button do what it says" test outright.

**The dock's step memory is boot-scoped (FULL-03, Medium).** `LoopFocus` is a fresh
`ChangeNotifierProvider` per boot (main.dart:41) with no persistence and no
path-parameter seeding on the PRD pages. Refresh `/prd/:id` and the dock correctly
highlights 版本 — then tapping it falls through `stay()` to `/dashboard`, because
`focus.prdId` is null and `_projectOnRoute` only matches `/projects/`. The
highlighted button ejects the user from the page it represents. Relatedly, 记下 and
想法 are two dock buttons with one identical destination, and the dashboard's
"Capture idea" silently chooses `list.first` of the user's workspaces.

**Export polish (FULL-04/FULL-05).** The markdown download branch discards the
stub's documented false return while the print branch beside it handles the same
contract — on the shipped iOS/Android targets the Download button is a silent
no-op. The page's primary action is also labeled `common.save` ("Save") for what is
really "generate a preview," which misstates the action.

Smaller items: the audience view issues eight identical `getPrd` calls per load
(FULL-06); three feature files are over the ~350-line rule with no dated modularity
issue filed — `project_page.dart` at 548 is the largest file in the client after
the two already-filed API files (FULL-07); the commit dialog closes silently on an
empty name (FULL-08); the project page headlines itself "Evidence" while loading
(FULL-09); and a failed readiness fetch before commit discards the carry list
(FULL-10).

### Pivot note

The reference shell is a sticky top-nav with *no bottom nav*; today's chrome is a
floating bottom dock that the coordinator confirmed visually clips page content
(and my look at `shots/11-versions-dark.png` shows the dock parked on top of the
commit card's gap list at 1440×900). Killing the dock in the pivot retires FULL-02
and FULL-03 wholesale — the strongest argument for taking the reference layout
literally rather than restyling the dock. Nothing else in these findings is
design-coupled: the guarded-action helper, the export fallback, and the audience
view fetch hoist survive the re-skin unchanged.

### Cheapest user-visible fix

One shared guarded-action wrapper, applied to the six unguarded write actions
first: idea capture, convert, version commit, workspace join/create, and PRD save.
It is roughly sixty lines, needs no design input, and converts an entire class of
silent failures into translated, visible feedback on the exact paths the product
demo walks.
