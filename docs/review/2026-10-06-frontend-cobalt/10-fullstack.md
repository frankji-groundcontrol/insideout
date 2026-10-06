# 10 · Fullstack

**Lens:** End-to-end wiring, API contracts, auth and boot flow, env/deploy parity
**Score:** **6.5 / 10**
**Bottom line:** Endpoint inventory is complete — every dock item and page action traces to a real ApiClient method, and the API_BASE env contract is coherent across code, docs, and Dockerfile. But the client ships a systemic silent-failure pattern on every primary write action (no try/catch, no SnackBar, no reload on error) and the bottom dock — the app's main navigation — can route to the wrong workspace's project and bounces to the dashboard after a refresh. Both are squarely fullstack defects on documented paths, plus a file-size-rule gap the repo's own contract says must be filed.

Reviewed branch `main` at `f185e94`; a skeptic pass re-checked every Critical/High finding — corrections marked [Correction]. Screenshot evidence lives in `./shots/`.

---

## Scope

`client/lib/router.dart`, all feature pages at action level, `api/client.dart` + `api/requests.dart`, `chrome/product_dock.dart` + `chrome/loop_focus.dart`, the `main.dart` → `app.dart` boot chain, `pubspec.yaml`, and `docs/usage/local-development.md`. Every named control was traced: the six dock items, idea capture and convert-to-PRD, the commit dialog, the export download path, invite-code join, and the admin signup-mode toggle. Verdict of the pass: the API surface is complete and the configuration contract is honest, but the client has a systemic silent-failure pattern on write actions and a dock that sometimes navigates to the wrong project.

## What is good

- **Token refresh is architecturally sound** — `client/lib/api/client.dart:77-83`: single-flight refresh, a `/auth/*` carve-out so terminal 401s surface untouched (`client/lib/api/client.dart:51-56`), and resolve-retry inside `onResponse` (`client/lib/api/client.dart:66-70`). This is the hard part of an API client done right; the SSE paths even re-read the rotated token on retry.
- **The env contract is coherent end to end** — `client/lib/session/session.dart:7-10` reads `String.fromEnvironment('API_BASE', defaultValue: 'http://127.0.0.1:8080/api/v1')`, matching `docs/usage/local-development.md:45,121` (`--dart-define=API_BASE`) and `client/Dockerfile` (`ARG API_BASE=/api/v1` for same-origin nginx). No drift between code, docs, and deploy.
- **Export download has a real web implementation** — `client/lib/api/download_web.dart` implements an object-URL anchor click with a deferred revoke for the new-tab path, behind a documented conditional-import stub; the print branch even honors the documented false-return fallback (`client/lib/features/prd/export_page.dart:89-92`).
- **Endpoint coverage is complete** — every affordance traced maps to a real `ApiClient` method: dock items resolve to `/workspace/:id/ideas`, `/prd/:id`, `/prd/:id/versions`, `/projects/:id/roadmap`, and the project page; idea capture → `createIdea`, convert → `convertIdea` (which returns the new `prdId` and navigates to it); commit → `commitPrd` with the readiness carry list; export → `exportPrd` with format/audience query params; join → `joinWorkspace`; admin → `setSignupMode`/`createInvitation`/`revokeInvitation`. No place where the UI asks for something the client cannot send.
- **The proposal-decision flow is honest about server authority** — `client/lib/features/project/project_page.dart:105-126` posts the decision, then applies accepted `add_node` items with parent-title matching; `_decide` wraps everything in `localizedError` → SnackBar (`client/lib/features/project/project_page.dart:89-99`) — the error-handling pattern most of the app lacks.
- **The admin signup-mode toggle works end to end** — `client/lib/features/profile/admin_section.dart`: ChoiceChip → `setSignupMode` (PUT /admin/settings) → `_load()` re-reads via authConfig and the ledger refreshes; the section is gated on `user.isAdmin` (`client/lib/features/profile/profile_page.dart:81`).

## Findings, worst first

### full-write-actions-fail-silently — High

Every primary write action awaits the API with no error handling, so a failed capture, convert, commit, join, or save fails with zero user feedback and no list reload. The pattern repeats across the loop's spine: `ideas_page._captureNow`, idea convert, `versions_page._commitDialog`, dashboard create/join, `prd_page._save`, repo save/sync on the project page — all `await api.…` inside button callbacks with no try/catch. The worst instance is the version commit: the user taps 提交版本, the server rejects the commit, and the UI offers no evidence either way.

**Evidence:** `client/lib/features/workspace/ideas_page.dart:63`

```
Future<void> _captureNow() async {
  final title = _capture.text.trim();
  if (title.isEmpty) return;
  await context.read<Session>().api.createIdea(widget.workspaceId, title, '');
  _capture.clear();
  await _reload();
}
```

— same shape at `ideas_page.dart:83, 110, 134`; `versions_page.dart:57-63` `await … api.commitPrd(...)`; `dashboard_page.dart:71-72` `await … api.joinWorkspace(code.text.trim())`; `prd_page.dart:81-86` `await … api.updatePrd(...)`.

A 409, 422, or dropped connection produces an unhandled async error in the logs and *nothing* on screen: no SnackBar, no dialog, and `_reload()` never runs. On the versions page the user believes the PRD was committed; on the ideas wall the captured idea simply never appears.

**Fix:** Add one guarded-action helper (try/catch → `ScaffoldMessenger` SnackBar with `localizedError`, success → reload) and route the six unguarded call sites through it, mirroring the pattern `project_page.dart` `_decide` already uses.

**Verdict:** CONFIRMED

### full-dock-crosses-workspaces — High

Dock Roadmap/Evidence prefer a stale `LoopFocus` projectId over the project on the current route, so tapping them from a second workspace's ideas page opens the first workspace's project. `ProductDock._go` resolves the project as `focus.projectId ?? _projectOnRoute(context)`, and `IdeasPage._seedProject` early-returns whenever any projectId is already set — nothing clears or re-keys it when the user crosses workspaces. The dock is the product's primary navigation; a navigation control that crosses workspace boundaries on a labeled button fails the "does each button do what it says" test outright.

**Evidence:** `client/lib/chrome/product_dock.dart:91`

```
final project = focus.projectId ?? _projectOnRoute(context);
```

— `ideas_page.dart:38`: `if (focus.projectId != null) return;` — `product_dock.dart:123-125` `case DockItem.roadmap: if (project != null) { context.go('/projects/$project/roadmap');`

**Fix:** Make dock navigation route-first — when the current path matches `^/projects/`, always use that id; otherwise fall back to `LoopFocus`. Also key the seeded project per workspace (or overwrite when the ideas page belongs to a different workspace than focus.projectId's origin).

**Verdict:** CONFIRMED

### full-dock-memory-boot-scoped — Medium

`LoopFocus` ids live only in memory, so after a browser refresh the highlighted dock item navigates to `/dashboard` instead of the page it represents, and Capture/Ideas are two buttons with one destination. `LoopFocus` is a fresh `ChangeNotifierProvider` per boot (`main.dart:41`) with no persistence and no path-parameter seeding on the PRD pages. Refresh `/prd/:id` and the dock correctly highlights 版本 — then tapping it falls through `stay()` to `/dashboard`, because `focus.prdId` is null and `_projectOnRoute` only matches `/projects/`. The highlighted button ejects the user from the page it represents. Separately, 记下 and 想法 are two dock buttons with one identical destination, and the dashboard's "Capture idea" silently picks `list.first` of the user's workspaces with no chooser.

**Evidence:** `client/lib/chrome/product_dock.dart:109`

```
case DockItem.version:
  final prd = focus.prdId;
  if (prd != null) { context.go('/prd/$prd'); } else { stay(); }
```

— `void stay() { if (project != null) { context.go('/projects/$project'); } else { context.go('/dashboard'); } }` — `main.dart:41` `ChangeNotifierProvider(create: (_) => LoopFocus())` — `DockItem.capture` and `DockItem.ideas` both `context.go('/workspace/$ws/ideas')`.

**Fix:** Seed focus ids from path parameters on every gated page's initState (cheap) or persist `LoopFocus` across boots; collapse capture+ideas into one dock item that focuses the capture field on arrival.

### full-download-stub-contract-ignored — Medium

The markdown download branch ignores the documented false-return of `downloadTextFile`, so on iOS/Android builds the Download button is a silent no-op while the print branch next to it handles the same contract correctly. The repo ships `ios/` and `android/` targets and the product docs describe a Flutter web + iOS + Android client.

**Evidence:** `client/lib/features/prd/export_page.dart:80`

```
FilledButton.tonal(
  onPressed: () => downloadTextFile('prd-${widget.prdId}.md', body!, mime: 'text/markdown'),
  child: Text(l10n.t('common.download')),
)
```

— `download_stub.dart`: `bool downloadTextFile(String filename, String text, {String mime = 'text/plain'}) => false;` — while the print branch does it right: `if (!openTextInNewTab(body!, mime: 'text/html')) { downloadTextFile('prd-${widget.prdId}.html', body!, mime: 'text/html'); }`

On a mobile build the conditional import resolves to the stub, `downloadTextFile` returns false, and the button discards the result — the user taps Download and nothing happens, with no fallback.

**Fix:** Check the return value on the markdown branch exactly like the print branch (fall back to `openTextInNewTab` or an error hint).

### full-line-budget-issues-missing — Medium

Three feature files exceed the ~350-line rule with no dated modularity issue filed, while the repo's own AGENTS.md says such files "get a dated modularity issue in docs/issues/". The largest, `project_page.dart` at 548 lines, mixes GitHub sync, evidence, updates, and proposals with no recorded target structure or bounded fix prompt, so nothing schedules its split while the documented rule claims coverage.

**Evidence:** `client/lib/features/project/project_page.dart:1`

```
wc -l: project_page.dart 548, workspace_page.dart 371, ideas_page.dart 370
```

— `client.dart` 633 and `models.dart` 539 have issues filed (`docs/issues/2026-09-18-flutter-apiclient-line-budget.md` and `docs/issues/2026-10-04-models-dart-line-budget.md`); `docs/issues/README.md:103-108` lists the two Dart issues only.

**Fix:** File one dated issue covering the three feature files with target splits (project_page → github_section + updates_section + evidence_section widgets), or split them now; add the rows to `docs/issues/README.md`.

### full-save-label-misleading — Low

The export page's primary action button is labeled "Save" though it generates a preview — the button does not say what it does. A user on `/prd/:id/export` hesitates: "Save" suggests persisting to the server; the action actually fetches rendered text into a preview.

**Evidence:** `client/lib/features/prd/export_page.dart:72`

```
FilledButton(onPressed: _run, child: Text(l10n.t('common.save'))),
```

— `l10n/en.dart:5` `'common.save': 'Save',`

**Fix:** Add an export-specific key (e.g. `'prd.exportRun': 'Generate export'`) and use it here.

### full-audience-view-request-waste — Low

The audience view fires one identical GET /prds/:id per PRD section — 8 redundant requests per page load — because each `_SectionBody` fetches the whole PRD itself (`prdSectionKeys` has 8 entries). Every audience-view load issues 8 byte-identical GETs against /prds/:id (plus the view projection call) — 9 requests where 2 suffice, multiplying server load for the most shareable page in the product.

**Evidence:** `client/lib/features/prd/audience_view_page.dart:145`

```
_SectionBody(prdId: widget.prdId, sectionKey: key), ... @override void initState() { super.initState(); _future = context.read<Session>().api.getPrd(widget.prdId); }
```

**Fix:** Hoist a single `getPrd` future into `_AudienceViewPageState` and pass the section string down to `_SectionBody` (which becomes stateless).

### full-commit-empty-name-swallowed — Low

Committing with an empty name silently closes the dialog and does nothing — no validation, no hint that the name is required. A user opens 提交版本, picks an audience, leaves the name blank, taps the commit button: the dialog disappears with no commit and no explanation.

**Evidence:** `client/lib/features/prd/versions_page.dart:56`

```
if (result == null || result.name.isEmpty) return;
```

**Fix:** Validate in `VersionsCommitDialog` (disable the confirm button or show a field error when `_name` is empty) instead of swallowing the result upstream.

### full-project-title-evidence-fallback — Low

While the project loads, the page title falls back to `l10n.t('evidence.title')`, so the Evidence screen briefly headlines itself "Evidence — Evidence". On a slow connection the project page renders the section eyebrow EVIDENCE with the literal word "Evidence" as the project headline until `getProject` resolves — a flash of wrong content on the dock's Evidence destination.

**Evidence:** `client/lib/features/project/project_page.dart:135`

```
final title = snap.data?['title'] as String? ?? l10n.t('evidence.title');
```

**Fix:** Fall back to an empty string (or a muted placeholder) and render the headline only once data exists.

### full-commit-carry-swallowed — Low

A failed readiness fetch before the commit dialog is swallowed and the commit proceeds with an empty unresolved/carry list, silently discarding the readiness gaps from the commit record. A transient network blip between page load and commit makes the readiness future unusable; the user commits and the record loses the unresolved-decisions list that `carryIntoCommit` was designed to preserve.

**Evidence:** `client/lib/features/prd/versions_page.dart:47`

```
try {
  carry = (await _readiness).audiences[audience]?.carryIntoCommit ?? const [];
} catch (_) {
  carry = const [];
}
```

**Fix:** Surface the readiness error and abort the dialog (or retry once) instead of committing with empty carry.

## Recommendations

- Add one guarded-action helper (catch → SnackBar with `localizedError`; success → reload) and route the unguarded write actions through it, starting with idea capture, idea convert, version commit, and workspace join.
- Make dock navigation route-first — derive the project from a `^/projects/` path before consulting `LoopFocus`, and seed the focused project per workspace.
- Seed `LoopFocus` ids from path parameters on every gated page (or persist them) so dock items work after refresh, and collapse the Capture and Ideas dock buttons into one.
- Honor the download stub contract on the markdown branch and rename the export run button to a generate/export label.
- Hoist the per-section `getPrd` calls in `AudienceViewPage` into one parent-level fetch.
- File dated modularity issues (or split) for `project_page.dart`, `workspace_page.dart`, and `ideas_page.dart`, and index them in `docs/issues/README.md`.
- Validate the version name inside `VersionsCommitDialog` instead of silently dropping an empty commit.

One pivot note from the raw review: the reference shell is a sticky top-nav with *no bottom nav*; today's chrome is a floating bottom dock that visually clips page content (`shots/11-versions-dark.png` shows the dock parked on top of the commit card's gap list at 1440×900). Killing the dock in the pivot retires the dock findings wholesale — the strongest argument for taking the reference layout literally. Nothing else here is design-coupled: the guarded-action helper, the export fallback, and the audience-view fetch hoist survive the re-skin unchanged.

## What this role would do first

Add one shared guarded-action wrapper (~60 lines, no design input) in `client/lib/features/` and apply it to the six unguarded write actions — `ideas_page.dart:63,83,110,134`, `versions_page.dart:57`, `dashboard_page.dart:71-72`, `prd_page.dart:81-86` — so every primary loop action reports failure via `localizedError` → SnackBar instead of failing silently.
