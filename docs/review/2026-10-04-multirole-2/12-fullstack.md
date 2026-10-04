# 12 · Fullstack

**Lens:** 
**Score:** **7.2 / 10**
**Bottom line:** Waves 0-4 closed the first review's seam gaps, but the second pass finds the problems one layer deeper: the print-export path is dead on popup-blocked browsers (nullable Window? ignored), SSE paths bypass the 401-refresh machinery, and the board's new data only refreshes on pull.

Reviewed main @ 26db979; a skeptic re-checked every High finding from this review. FS-01 was confirmed High by the skeptic: `package:web` 1.1.1 declares `external Window? open(`, the call site discards the return and returns `true` unconditionally, and the only caller gates its fallback on that boolean — the popup-blocked fallback is provably dead.

---

This review was persisted findings-only; the section below is assembled verbatim from its structured findings (the prose wrapper was not retained). 11 findings as filed, worst first.

## Findings

### FS-01 · High ·  **Skeptic: CONFIRMED.**

**openTextInNewTab ignores window.open's nullable return, so the popup-blocked fallback in export_page.dart is dead code and print export silently does nothing.**

- Where: `client/lib/api/download_web.dart:19`
- Evidence: download_web.dart:19 'web.window.open(url, _blank);' with unconditional 'return true'; package:web 1.1.1 html.dart:10095 'external Window? open(['; export_page.dart:89-91 gates the fallback on the return.
- Fix: Capture the result: 'final w = web.window.open(url, _blank); if (w == null) return false;' so export_page's existing fallback fires; add a widget test for the fallback branch.
- Diff size: ~6 lines + test

### FS-02 · Medium · 

**SSE paths (coach send, presence) attach a token snapshot and never participate in the 401-refresh flow; presence loop treats 401 as fatal and dies permanently.**

- Where: `client/lib/api/client.dart:450`
- Evidence: client.dart:451-457 token snapshot for coachSseOpen; roadmap_page.dart:68-69 'fatal = e.status == 401 …; if (fatal || !mounted) return;'.
- Fix: On 401 from the SSE open, await the same single-flight refresh and retry once; downgrade presence-loop 401 to non-fatal when refresh succeeds. First check: manual token-expiry walk per the SSE practice doc.
- Diff size: ~40 lines

### FS-03 · Medium · 

**The board (quiet badge, latest updates) refreshes only via pull-to-refresh — no lifecycle/focus refresh or poll; stale quiet flags persist for a whole session.**

- Where: `client/lib/features/workspace/workspace_page.dart:87`
- Evidence: workspace_page.dart:87-88 RefreshIndicator is the only trigger; no WidgetsBindingObserver anywhere in client/lib; loads once in initState.
- Fix: WidgetsBindingObserver refresh on AppLifecycleState.resumed + a visible refresh action; optionally a 60s poll while visible. First check: RefreshIndicator physics on short lists on web.
- Diff size: ~20 lines

### FS-04 · Medium · 

**Audience dropdown is offered in print mode, but the server 400s print+audience; user hits a raw server error instead of being prevented.**

- Where: `client/lib/features/prd/export_page.dart:60`
- Evidence: export_page.dart:60-69 dropdown live for both formats; prds.go:307-309 400 'audience projections export as markdown (format=markdown) for now'.
- Fix: Disable/hide the audience dropdown when format == 'print' (or auto-switch to markdown when an audience is chosen); surface server errors via a localized helper.
- Diff size: ~10 lines

### FS-05 · Medium · 

**Bands (secondary) view buckets by status and folds done into 'Now', contradicting the default Progress view's deadline truth — while both share the same Now/Next labels.**

- Where: `client/lib/features/project/roadmap_bands.dart:26`
- Evidence: roadmap_bands.dart:26-31 bucket by status with done→'now' vs roadmap_progress_view.dart Now = deadlined in-progress only; both use roadmap.bandNow/bandNext keys.
- Fix: Relabel the Bands view ('By status': Active/Queued/Icebox via new l10n keys) or derive it from the same roadmaptime.Assemble output; one meaning per label.
- Diff size: ~20 lines

### FS-06 · Medium · 

**Mutation callbacks are fire-and-forget with no try/catch or SnackBar — failed capture/drop/convert/create look like no-ops.**

- Where: `client/lib/features/workspace/ideas_page.dart:55`
- Evidence: ideas_page.dart:55-61 _captureNow no catch; ideas_page.dart:107-118 async closures nobody awaits; workspace_page.dart:60-63 same; no SnackBar in either file.
- Fix: try/catch + SnackBar (or inline error Text) around these mutations; a small runGuarded(context, fn) helper keeps it DRY.
- Diff size: ~40 lines

### FS-07 · Low · 

**CI builds with Flutter 3.47.0 while the production Docker image pins 3.44.0 — green CI does not guarantee the shipped artifact builds.**

- Where: `.github/workflows/ci.yml:81`
- Evidence: ci.yml:81 flutter-version: 3.47.0 vs client/Dockerfile:3 'FROM ghcr.io/cirruslabs/flutter:3.44.0 AS build'.
- Fix: Pin CI to the Dockerfile toolchain, or add a client Docker image build job; keep one source of truth for the version.
- Diff size: ~2 lines

### FS-08 · Low · 

**setState after await without mounted guard (the accepted infos baseline hides real unmount crashes).**

- Where: `client/lib/features/prd/export_page.dart:33`
- Evidence: export_page.dart:31-36 await then setState, no mounted check; same shape ideas_page.dart:58-60; inconsistent with the guarded workspace_page.dart:60 / roadmap_page.dart:127.
- Fix: Add 'if (!mounted) return;' after each await before setState/useContext; tighten the infos baseline file-by-file.
- Diff size: ~10 lines

### FS-09 · Low · 

**Board list embeds the full latest-update content (up to 5000 runes) per project while the client renders one ellipsized line.**

- Where: `server/internal/api/projects.go:36`
- Evidence: projects.go:36 latestUpdateContent populated verbatim; content bound maxUpdateContentRunes = 5000; client maxLines: 1.
- Fix: Truncate in projectResponse for the list view (~200 runes) or serve a truncated preview field; keep full content on GET /projects/{pid}.
- Diff size: ~10 lines

### FS-10 · Low · 

**Idea poster falls back to rendering raw untranslated idea.status as body text.**

- Where: `client/lib/features/workspace/ideas_page.dart:304`
- Evidence: ideas_page.dart:304 'Text(idea.content.isEmpty ? idea.status : idea.content' — bare enum like 'private'/'dropped', never through l10n.t.
- Fix: Localize via 'idea.status.<s>' keys, or render a neutral muted hint when content is empty.
- Diff size: ~5 lines

### FS-11 · Low · 

**ShadePill is defined inside ideas_page.dart but consumed cross-feature by workspace_page — page files doubling as a component library.**

- Where: `client/lib/features/workspace/workspace_page.dart:12`
- Evidence: workspace_page.dart:12 imports ideas_page.dart for ShadePill (ideas_page.dart:326-344).
- Fix: Move ShadePill to client/lib/theme/ beside field_card.dart; two-line diff.
- Diff size: ~4 lines

