## Code-quality review — client/

Scope: all of `client/lib` (68 files, 10,173 lines) and `client/test` (34
files, ~2,959 lines), checked against the repo's own modularity rule
(AGENTS.md: files under ~350 lines, with a dated issue owed for any chunky
file that can't be split now). Every lead was verified by opening the file;
none of the supplied leads were refuted, and two are bigger than reported.

### Formatting: no gate, and the tree isn't clean

The single most surprising result is that formatting is entirely unenforced.
`.github/workflows/ci.yml:91` runs `flutter analyze --no-fatal-infos`, but
nothing checks `dart format`. Probing read-only with
`dart format --output=none lib` reports **55 of 68 files would change**.
The clearest symptom is `client/lib/api/sse.dart`: 22 tab-indented lines
(the import block at 1–4 and the whole `readProgressSse` region at 96–115)
sit inside an otherwise space-indented file — you can see the fault line at
line 116, where `readCoachSse` returns to spaces. `revisions_page.dart:52-54`
ships a mis-indented `setState` body that any formatter would fix.

This is the classic trap: the first contributor with format-on-save silently
produces a 55-file diff that buries their real work, and the tab block lives
in the SSE layer — the most timing-sensitive file in the client. The fix is
two cheap steps, worth a commit each: one behavior-neutral `dart format lib`
(verified by `flutter test`), then a `--set-exit-if-changed` step in CI.
Until that lands, every other cleanup in this review makes the eventual
format diff bigger.

### Line budget: tracked for two files, silently broken for four

The rule is real and partially working. Both heavyweight API files have
well-written split issues with target structures: `client/lib/api/client.dart`
(633 lines → core/product/stream) and `client/lib/api/models.dart`
(539 lines → four domain barrels). That is the process working as designed.

But four more files are over budget with **no issue filed**:
`features/project/project_page.dart` at 548, `l10n/en.dart` at 378,
`features/workspace/workspace_page.dart` at 371, and
`features/workspace/ideas_page.dart` at 370. A grep of `docs/issues/` for
`project_page` returns nothing. That matters most for `project_page.dart`,
which is not just long but genuinely mixed: one file holds the page build,
a leaf-evidence side-loader (`_loadLeafEvidence`, line 56), the FEAT-03
proposal decision loop and its accepted-items applier, plus three inline
dialog methods (`_edit` at 407, `_delete` at 473, `_addUpdate` at 498).
Every feature slice to the busiest screen in the app lands in this file
with no agreed seam. The two filed issues also understate current size
(~529 vs 633, ~478 vs 539), so the drift is already underway.

The l10n catalogs deserve an explicit policy rather than silence: they are
data-shaped like `models.dart`, which got an issue, so either file one or
record why key catalogs are exempt.

### Copy-paste: one FutureBuilder branch, eight homes

The pending/error branch of `FutureBuilder` is duplicated near-verbatim in
six files, all some spelling of:

```dart
Center(child: snap.hasError
    ? Text(localizedError(snap.error!, l10n.t))
    : const CircularProgressIndicator())
```

— `ideas_page.dart:125`, `settings_page.dart:63`, `revisions_page.dart:63`,
`audience_view_page.dart:70`, and twice in `roadmap_page.dart` (211, 220),
with semantic twins at `workspace_page.dart:121-123` and
`project_page.dart:143-145`. Two copies use the raw, un-localized variant —
`dashboard_page.dart:98` and `versions_page.dart:73` both render
`Text('${snap.error}')` — which is how raw exception text still reaches the
dashboard and versions screens despite `localizedError` existing as the
standard. A single shared gate widget (snapshot in, builder for data in)
deletes eight copies and fixes both leaks in one mechanical pass. This is
also the natural seam for the design pivot, which wants a different
error/empty-state treatment everywhere at once.

### Widget ownership: a theme pill living in a feature file

`ideas_page.dart:348-370` defines `ShadePill`, a pill container built from
`Cobalt` palette tokens — a theme widget by construction. Another feature
imports it: `workspace_page.dart:14` pulls in all of `ideas_page.dart` for a
single use (`ShadePill(label: l10n.t('project.quiet'))` at line 221). The
theme already owns the canonical version: `ShadeChip` in
`theme/cobalt_chip.dart:8`, with the same pill radius and a tone system.
The file also carries a single-use `extension on BorderSide { toBorder() }`
(lines 368-370) that exists only to feed line 361. One move into
`cobalt_chip.dart` (or a neutral `ShadeChip` tone covering both call sites)
plus deleting the extension removes a cross-feature import and two shapes of
the same pill before the pivot redraws them anyway.

### Smaller items, verified

- `assembly_diagram.dart:167-170`: two consecutive arms both
  `return ink` (`i == currentStep - 1` and `i < currentStep - 1`) —
  collapsible to one comparison. Dead in the strict sense: the second arm
  adds no behavior.
- `roadmap_page.dart:40-99`: one `_retryTimer` field backs both
  `_progressLoop` and `_presenceLoop`; the second `_backoff` call clobbers
  the first loop's cancel handle, and `dispose()` cancels only the latest.
  Today's behavior is correct only because abandoned timers complete their
  own completers; the twin loop bodies (56-72 vs 74-95, including the
  duplicated backoff-cap line) want one parameterized helper with per-loop
  timers.
- `step_peek.dart:220`: the landing teaser's mock git hashes are
  `['a1b2c3d', 'e4f5g6h', 'i7j8k9l']` — the latter two contain `g` through
  `l`. The first screen a visitor sees shows impossible SHAs.

### What is genuinely good

The error-localization idiom is one helper (`localizedError`,
`api/errors_l10n.dart:16`) used at 15+ sites — the duplication problem is at
least uniformly the *right* paste in most files. The test suite targets the
sharp edges rather than padding coverage: `sse_test.dart`,
`api_token_refresh_test.dart`, `errors_l10n_test.dart`, and
`leftover_scan_test.dart` — a sweep guard that fails CI if the retired
Ink & Seal tokens ever return to `lib/`. Animation and painting are kept out
of pages (`ink_motion.dart`, `assembly_diagram.dart`), and the two worst API
files already have actionable, structure-level split issues — evidence the
modularity rule has teeth when it is applied.

### Sequencing

Land the format sweep first (mechanical, unblocks review of everything
else), then the shared async-gate widget (biggest deletion count, fixes the
two raw-error leaks), then the ShadePill move, and file the
`project_page.dart` modularity issue before the next feature slice touches
that screen. The micro-cleanups (dead arms, mock hashes) ride along as one
small commit.
