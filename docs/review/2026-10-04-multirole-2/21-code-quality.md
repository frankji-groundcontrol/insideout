# 21 · Code Quality

**Lens:** file-size/modularity rule compliance, duplication, dead code, magic numbers, test quality and tiering, log-only test branches, conditional-import platform wiring, coverage gaps
**Score:** **8.4 / 10**
**Bottom line:** The codebase is in strong modular health at 26db979: after the five known dated line-budget issues, nothing else breaches the ~350-line rule — roadmap_test.go (494) and en.dart (355) are correctly out of the rule's scope (a cohesive single-domain test suite and a flat string table, neither chunky implementation), and every other file in server/, client/lib and scripts/ sits under budget. The 2026-10-04 additions are above-average work: the SSE streaming test proves flushing rather than asserting final bytes, the ideas-privacy test exercises the RLS policy itself rather than just query filters, and the quiet-flag logic has an exact threshold-boundary test. The residue is small and mechanical: a ~40-line layout duplication between the two roadmap band widgets, an export-download path with zero coverage at any tier, re-implemented date-truncation logic next to an existing helper (with the hazard unguarded at two older call sites), and a couple of magic constants. No log-only test branches remain — the single t.Logf left is informational and follows hard assertions.

Reviewed main @ 26db979; a skeptic re-checked every High finding from this review. No High findings in this dimension; below the skeptic verification bar. The reviewer ran serially after a rate-limited first attempt.

---

## Code quality — 26db979

**Verdict: healthy.** After the five known dated line-budget issues, this tree contains no new modularity violations, and the 2026-10-04 additions raise the test-quality bar rather than lowering it. What remains is small and mechanical: one ~40-line widget duplication, one uncovered feature path, and a date-truncation helper that was written the same week it was bypassed.

### Line-budget audit

One `wc -l` pass over `server/`, `client/lib` and `scripts/` (excluding generated) leaves exactly two files over 350 that are not already covered by dated issues:

- **`server/internal/store/roadmap_test.go` (494)** — exempt, correctly. The repo rule targets *implementation* files; this is one domain's cohesive store suite (six tests from `TestRoadmap_TreeLifecycle` to `TestRoadmap_Attribution` plus four small helpers) and splitting it would scatter a single behavior's coverage across files. If it keeps growing, `TestRoadmap_Attribution` (line 373, with its own `mkNamedUser`/`strPtr` helpers) is the natural seam — but that is a preference, not a violation.
- **`client/lib/l10n/en.dart` (355)** — exempt. A flat string table is data, not logic; single responsibility by construction, five lines over is noise. `zh.dart` at 345 shows the pair is maintained together.

Everything else clears the bar with room: the next-largest files (ideas_page 348, auth_email_flows.go 344, anthropic_test.go 344, prds.go 343, auth_throttle_test.go 338, tools.go 338) all sit under 350. No chunky scripts anywhere in `scripts/`.

### Test-quality sweep

**Log-only branches: clean.** The entire `server/internal` tree contains exactly one `t.Logf` — `prd_revisions_test.go:70`, an informational line *after* the fatal assertions that already pin winners ≥ 1 and the opaque-error guard at line 64. That is the correct use of Logf; there is no conditional anywhere that should have been `t.Errorf`.

**The new tests are the strongest in the repo.** `conversations_test.go` is the standout: the stub coach writes frame 1 then blocks on `releaseFrame2`, so the test's `readFrame` at line 144 can only succeed if the frame was flushed through the entire middleware stack — a buffering regression fails loudly at the 5-second timeout (lines 155-157) instead of passing on final bytes. The `t.Cleanup` at lines 107-111 always closes the channel, which is precisely the wedge fix 26db979 describes. `ideas_privacy_test.go` tests the RLS policy itself with actor-context SELECTs (lines 78-86) rather than trusting the query filters, and then confirms admin drop authority still holds. `projects_test.go` pins the boundary exactly: `quietAfter` to the second is not flagged, one second more is.

**The lint-tier demotion landed properly.** `feature_wiring_test.dart:5-10` now self-documents as a presence-only backstop and names its behavioral replacements — and those replacements exist and are real: `workspace_page_test.dart` renders the board through a `_FakeApi` with `flagged: true` fixtures and asserts the quiet badge and update line appear (line 65), and `roadmap_page_test.dart:54` asserts the progress view is the default. Nothing left in the substring file is testing behavior that matters; the remaining checks (`inviteCode`, `kIsWeb`, `parentId: node.id`) are genuinely wiring-presence. One gap slipped through, though — see CQ-02.

### Duplication

**CQ-01 (Medium): the two band widgets.** `_BandRow` (`roadmap_bands.dart:73-125`) and `_ProgressBand` (`roadmap_progress_view.dart:50-130`) share byte-identical chrome — the `EdgeInsets.fromLTRB(16, 16, 16, 16)` container with `Cobalt.radiusHero`, the 72px uppercase label column with `letterSpacing: 0.6`, the `Wrap(spacing: 12, runSpacing: 12)`, the 240px card width, the `'—'` empty state. Only the card contents differ (`RoadmapNodeCard` vs the inline pressure/deadline card). Since the Progress view is now the default and Bands is the structural alternative, a visual tweak to one silently skips the other. Both files live in the same directory; a ~60-line shared scaffold widget takes the label, colors, and item children.

### Magic numbers, dead code, robustness

No dead code found in the new files; the conditional import pair (`download_stub.dart` / `download_web.dart`) is correctly wired via `dart.library.ui_web` in `export_page.dart:8-9` with the stub returning `false` so callers keep inline fallbacks — good pattern. Two small residues: the new view re-implements guarded date truncation (`item.deadline!.length >= 10` → `substring(0, 10)`) while `chrome/time_ago.dart:9` already ships exactly that helper, and — the more interesting half — the older `roadmap_node_card.dart` calls `substring(0, 10)` **unguarded** at lines 53 and 164, a RangeError waiting on any short deadline string. The new code guarding the hazard it duplicates is the tell: route all three sites through one helper. `download_web.dart:22`'s bare `60_000.toJS` deserves a name next to its good comment.

### Coverage gap worth a backstop

`export_page.dart:80,89-90` wires `downloadTextFile`/`openTextInNewTab` — the FEAT-03 headline — yet `feature_wiring_test.dart` never reads export_page and no widget test touches it. Everything else demoted to lint-tier got a behavioral home; the export path got neither tier. A one-line substring check is the proportionate fix.

The two mechanical cleanups worth a commit each: the shared band scaffold extraction (CQ-01), and the guarded deadline-truncation helper adopted across all three call sites (CQ-03 + CQ-04).

What this dimension would do first: extract `roadmap_band_scaffold.dart` and delete the duplicated chrome from both band widgets.
