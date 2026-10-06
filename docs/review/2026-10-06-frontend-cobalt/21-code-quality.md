# 21 · Code Quality

**Lens:** Structure, duplication, size budget, dead code, formatting drift
**Score:** **6.5 / 10**
**Bottom line:** The client is consistent in its idioms and well-covered by tests, but it has no formatting gate (55 of 68 lib files are not dart-format-clean), one theme-widget living in a feature file that another feature imports, a six-file copy-paste of the FutureBuilder error/loading branch, and four files over the repo's own ~350-line rule with no modularity issue filed — including a 548-line project_page.dart that mixes page, dialogs, proposal logic, and evidence loading.

Reviewed branch `main` at `f185e94`; a skeptic pass re-checked every Critical/High finding — corrections marked [Correction]. Screenshot evidence lives in `./shots/`.

---

## Scope

All of `client/lib` (68 files, 10,173 lines) and `client/test` (34 files, ~2,959 lines), checked against the repo's own modularity rule (AGENTS.md: files under ~350 lines, with a dated issue owed for any chunky file that can't be split now). Every lead was verified by opening the file; none of the supplied leads were refuted, and two are bigger than reported.

## What is good

- Line-budget debt is tracked for the two worst files with concrete target structures and fix prompts: `docs/issues/2026-09-18-flutter-apiclient-line-budget.md` (`client/lib/api/client.dart` → core/product/stream split) and `docs/issues/2026-10-04-models-dart-line-budget.md` (`client/lib/api/models.dart` → four domain barrels). The modularity rule has teeth when it is applied.
- Error localization has one standard helper, `localizedError` (`client/lib/api/errors_l10n.dart:16`), used at 15+ call sites — the copy-paste problem is at least uniformly the *right* paste in most files.
- The test suite (34 files, ~2,959 lines) covers the sharp edges rather than padding coverage: `client/test/sse_test.dart`, `client/test/api_token_refresh_test.dart`, `client/test/errors_l10n_test.dart`, and `client/test/leftover_scan_test.dart` — a sweep guard that fails CI if the retired Ink & Seal design tokens ever return to `lib/`.
- Animation and painting code is kept out of pages: `client/lib/features/landing/ink_motion.dart` (192 lines) and `client/lib/features/landing/assembly_diagram.dart` own the motion/keyframe layer; pages only consume it.

## Findings, worst first

### cq-line-budget-drift — Medium

Four client files exceed the repo's ~350-line rule with no modularity issue filed, and the 548-line `project_page.dart` is genuinely mixed-responsibility: one file holds the page build, a leaf-evidence side-loader (`_loadLeafEvidence`), the FEAT-03 proposal decision loop and its accepted-items applier, plus three inline dialog methods. Every feature slice to the busiest screen in the app lands in this file with no agreed seam. The two filed issues also understate current size (~529 vs 633 for `client.dart`, ~478 vs 539 for `models.dart`), so the drift is already underway. A grep of `docs/issues/` for `project_page` returns nothing.

Evidence: `client/lib/features/project/project_page.dart:1`

```
Future<List<({String title, List<Map<String, dynamic>> rows})>>
  _loadLeafEvidence() async { ... } and line 407 Future<void> _edit(Map<String, dynamic> data) { — wc -l: project_page.dart 548, l10n/en.dart 378, workspace_page.dart 371, ideas_page.dart 370, l10n/zh.dart 368; `grep -ril 'project_page' docs/issues/` returns nothing.
```

**Fix:** File a dated modularity issue for `project_page.dart` (split seams: page/evidence section, proposal-decision+apply logic, dialog methods), state a policy for the l10n catalogs, and refresh the stale line counts in the two existing issues.

### cq-async-branch-copypaste — Medium

The identical `FutureBuilder` error/loading ternary is copy-pasted in six files, and two of the copies leak raw exception text instead of the localized branch. The verbatim copies: `ideas_page.dart:125`, `settings_page.dart:63`, `revisions_page.dart:63`, `audience_view_page.dart:70`, twice in `roadmap_page.dart` (211, 220), with semantic twins at `workspace_page.dart:121-123` and `project_page.dart:143-145`. The raw variants at `dashboard_page.dart:98` and `versions_page.dart:73` render `Text('${snap.error}')` — which is how raw exception text still reaches the dashboard and versions screens despite `localizedError` being the standard. Any change to the pending/error UX must be hand-edited in eight places, and the two raw-error copies will be missed exactly as they were when `localizedError` was adopted. A single shared gate widget is also the natural seam for the design pivot, which wants a different error/empty-state treatment everywhere at once.

Evidence: `client/lib/features/workspace/ideas_page.dart:125`

```
return Center(child: snap.hasError ? Text(localizedError(snap.error!, l10n.t)) : const CircularProgressIndicator()); — verbatim identical at settings_page.dart:63, revisions_page.dart:63, audience_view_page.dart:70, roadmap_page.dart:211 and :220; raw twin at dashboard_page.dart:98: if (snap.hasError) return Center(child: Text('${snap.error}'));
```

**Fix:** Add one shared widget (e.g. `chrome/async_gate.dart`) taking the `AsyncSnapshot` and a data-builder; migrate all eight call sites, converting the two raw-error copies to the localized branch in the same pass.

### cq-theme-widget-in-feature — Medium

`ShadePill` is a theme-level pill — built from `Cobalt` palette tokens, with the canonical `ShadeChip` already living in `theme/cobalt_chip.dart:8` at the same pill radius with a tone system — but it is defined at the bottom of a feature file and imported cross-feature: `workspace_page.dart:14` pulls in all of `ideas_page.dart` for a single use at line 221. The file also carries a single-use `extension on BorderSide { toBorder() }` that exists only to feed one call site. The design pivot to square badges will touch pills in two shapes in two locations, and one gets missed.

Evidence: `client/lib/features/workspace/ideas_page.dart:348`

```
class ShadePill extends StatelessWidget {
  const ShadePill({super.key, required this.label}); ... borderRadius: BorderRadius.circular(Cobalt.radiusPill) — workspace_page.dart:14 import 'ideas_page.dart'; used solely for ShadePill at :221; canonical chip at theme/cobalt_chip.dart:8: class ShadeChip extends StatelessWidget {
```

**Fix:** Move `ShadePill` into `theme/cobalt_chip.dart` (or give `ShadeChip` a neutral tone that covers both call sites), delete the `toBorder` extension, and drop the `ideas_page` import from `workspace_page`.

### cq-no-format-gate — Medium

Formatting is entirely unenforced. `.github/workflows/ci.yml:91` runs `flutter analyze --no-fatal-infos`, but nothing checks `dart format`; a read-only probe (`dart format --output=none lib`) reports **55 of 68 files would change**. The clearest symptom is `client/lib/api/sse.dart`: 22 tab-indented lines (the import block at 1–4 and the whole `readProgressSse` region at 96–115) sit inside an otherwise space-indented file — the fault line is visible at line 116, where `readCoachSse` returns to spaces. `revisions_page.dart:52-54` ships a mis-indented `setState` body. The first contributor with format-on-save silently produces a 55-file diff that buries their real work, and the tab block lives in the SSE layer — the most timing-sensitive file in the client. Until this lands, every other cleanup in this review makes the eventual format diff bigger.

Evidence: `client/lib/api/sse.dart:96`

```
	Future<void> readProgressSse(Stream<List<int>> chunks, void Function(RoadmapProgress progress) onProgress) async { — `dart format --output=none lib` reports "Formatted 68 files (55 changed)"; grep -c '\t' lib/api/sse.dart → 22; CI has only "- run: flutter analyze --no-fatal-infos" (.github/workflows/ci.yml:91).
```

**Fix:** One commit running `dart format lib` (behavior-neutral, verified by `flutter test`), followed by adding `dart format --output=none --set-exit-if-changed lib` to `ci.yml`.

### cq-dead-if-arms — Low

Two consecutive if-arms in `_nodeColor` both `return ink` (`i == currentStep - 1` and `i < currentStep - 1`) — the second arm adds no behavior. A future reader must diff two lines to learn they are the same, and any per-step color change has to be made twice or the arms drift apart silently.

Evidence: `client/lib/features/landing/assembly_diagram.dart:167`

```dart
if (mode == AssemblyDiagramMode.hero) return i == 3 ? brand : ink;
    if (i == currentStep - 1) return ink;
    if (i < currentStep - 1) return ink;
    return muted;
```

**Fix:** Collapse to a single `if (i <= currentStep - 1) return ink;`.

### cq-shared-retry-timer — Low

One shared `_retryTimer` field backs two independent backoff loops (`_progressLoop` and `_presenceLoop`), and the two loop bodies (lines 56-72 vs 74-95, including a duplicated backoff-cap line) are near-verbatim duplicates. When both loops back off simultaneously the earlier `Timer` loses its cancel handle — `dispose()` cancels only the latest — leaving a hung `await _backoff` on dead state. Today's behavior is correct only because abandoned timers complete their own completers. Adding a third stream copy-pastes the loop a third time.

Evidence: `client/lib/features/project/roadmap_page.dart:40`

```
Timer? _retryTimer; ... _retryTimer = Timer(d, done.complete); ... backoff = backoff * 2 > const Duration(seconds: 5) ? const Duration(seconds: 5) : backoff * 2; (identical cap at lines 70 and 93); _retryTimer?.cancel(); in dispose at line 99.
```

**Fix:** Extract one parameterized retry-loop helper (stream call, fatal-status set, backoff cap) holding its own `Timer`, or give each loop its own timer field.

### cq-mock-hashes — Low

The landing teaser's mock commit hashes contain non-hex characters (`g` through `l`). The first screen a visitor sees shows "git history" with impossible SHAs; anyone demoing the product to a technical audience gets asked about it.

Evidence: `client/lib/features/landing/step_peek.dart:220`

```dart
const hashes = ['a1b2c3d', 'e4f5g6h', 'i7j8k9l'];
```

**Fix:** Replace with plausible hex strings (e.g. `'e4f5a6b'`, `'c7d8e9f'`).

## Recommendations

- Run `dart format lib` as a standalone commit and add a `dart format --set-exit-if-changed` step to `.github/workflows/ci.yml` next to the `flutter analyze` step.
- Extract the duplicated `FutureBuilder` error/loading branch into a single `chrome/` async-gate widget and migrate all eight call sites, converting `dashboard_page.dart:98` and `versions_page.dart:73` to the localized branch in the same pass.
- Move `ShadePill` into `theme/cobalt_chip.dart` (or fold it into `ShadeChip` as a neutral tone), delete the single-use `toBorder` extension, and remove `workspace_page.dart`'s import of `ideas_page.dart`.
- File a dated modularity issue for `project_page.dart` (548 lines, mixed page/dialog/proposal/evidence responsibilities) and record a policy for the over-budget l10n catalogs; refresh the stale line counts in the `client.dart` and `models.dart` issues.
- Collapse the duplicate if-arms in `assembly_diagram.dart:168-169` and fix the non-hex mock hashes in `step_peek.dart:220` as a micro-cleanup commit.
- Deduplicate `roadmap_page.dart`'s twin retry loops behind one helper with per-loop timers when the next roadmap slice touches that file.
- After the pivot lands, re-run the format+analyze pair and the full client test suite as the definition of done for each design-language commit.

## What this role would do first

Land the `dart format lib` sweep plus the CI format gate in `.github/workflows/ci.yml` as the very next commit — it is purely mechanical, verified by `flutter test`, and immediately absorbs the 22-line tab block in `client/lib/api/sse.dart` and the mis-indented `setState` at `revisions_page.dart:52-54`, so every later cleanup diff in this list stays reviewable instead of being buried inside a 55-file format avalanche.
