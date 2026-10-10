# 21 · Code Quality

**Score:** **8.6 / 10** · 7 findings (0 High+)

**Bottom line:** The 90-commit wave left a genuinely clean core — centralized decode/error/read-gate/form/retry-taxon helpers, zero bare returns, disciplined logging, and self-policing scan guards — but its seams survived at the edges: 31 raw numeric spacing values (15 off the DESIGN.md 4px law) that the color-only guard cannot see, one 30-line dialog copy that bypasses the shared form primitive, a triplicated retry wrapper in the agent package, one production-dead export, two stale comments, and five PRODUCT.md promises (Product Branch/Merge chief among them) with no implementation anywhere in the tree.

---

## Code-quality review — review 3, tree at `main` @ 754c502

**Score: 8.6 / 10.** The remediation wave did the hard part right: the load-bearing helpers are genuinely single-sourced and self-policed by scan tests. What survived is seam-level — the guard that watches colors but not spacing, one dialog copy that bypasses the shared form, a retry wrapper pasted three times, one dead export, two stale comments — plus a product-doc divergence that no code review of the server alone would surface. None of it blocks shipping; all of it is cheap to fix, and two of the fixes are pure mechanical commits.

### How I reviewed

I read the full `server/internal` and `client/lib` trees (20322 + 16629 lines across ~270 files), `scripts/`, the test trees, `DESIGN.md`, and `docs/design-system/`. Checks I ran myself:

- `rg` scans for print-vs-log (0 `fmt.Print*` in `internal/`, all in `cmd/` CLI where it belongs; `internal/` uses `slog`), bare error returns (`rg -x "\t\treturn err"` in `internal/api` → 0, against 118 `s.log.Error` sites), and panics (1, the deliberate repanic at `coach.go:175`).
- A package-wide dead-identifier sweep (every unexported func, then every exported func with no consumers outside its package) — one survivor: `ClearVar`.
- An 8-line identical-block detector over `client/lib` (Python, MD5 over normalized chunks), which surfaced the duplication findings below; each pair was then read side by side.
- A raw-value histogram: `rg -o 'SizedBox\((height|width): ([0-9]+)'` → 31 numeric gaps; `rg 'Color(0x'` outside theme/ → 0.
- `flutter test test/color_literal_scan_test.dart test/leftover_scan_test.dart` → 3/3 passed (confirms the committed guards are green and color-clean). `go test -count=1 ./internal/agent/ ./internal/api/` → both `ok`.
- Coordinator leads 14 and 15 (below).

### Findings

| # | Sev | Bucket | Where | The defect | Fix |
|---|-----|--------|-------|-----------|-----|
| CQ-01 | Med | FIX | `client/lib/features/auth/auth_door.dart:100` | `const SizedBox(height: 20),` — 31 raw numeric gaps, 15 off the 4px law; guard scans colors only | Normalize to `Cobalt.spaceN`; extend the scan test to numeric spacing |
| CQ-02 | Med | ADD | `PRODUCT.md:206` | "**Product Branch / Explore direction:** an alternative product hypothesis," — no implementation anywhere | Record dated issues; build the branch/merge slice first |
| CQ-03 | Med | IMPROVE | `client/lib/features/prd/prd_page.dart:125` | `Future<void> _snapshot() async {` — 30-line verbatim copy, bypasses `showBlockForm` | One shared flow both pages call |
| CQ-04 | Low | IMPROVE | `server/internal/agent/responses.go:55` | `streamed := 0` / `countDelta := func(d string) {` triplicated | One `withStreamRetry` helper |
| CQ-05 | Low | IMPROVE | `server/internal/setup/envwrite/envwrite.go:66` | `func ClearVar(envPath, name string) error {` — test-only, production-dead | Delete with tests, or wire into local setup |
| CQ-06 | Low | IMPROVE | `client/lib/features/workspace/workspace_page.dart:317` | `Widget _navCard(` duplicates `_ActionCard` (home_field.dart:272); `_delete` duplicated at :135 and settings_page.dart:78 | Promote the tile to theme/; share the delete flow |
| CQ-07 | Low | IMPROVE | `server/internal/agent/coach.go:226` | "the ledger, once H2 lands, rides along" — it landed; same class at `flow.go:84` "local mode is not implemented yet" | Reword both to present state |

### Notes on the findings

**CQ-01 — the spacing law has no guard.** DESIGN.md:126 says spacing is the named scale (`space1`–`space10` = 4/8/12/16/24/32/40). The histogram of raw values: `20×7, 16×5, 8×4, 12×4, 6×2, 48×2, 36×1, 28×1, 10×1, 2×1`, plus `EdgeInsets.only(bottom: 80)` at landing_page.dart:122 and `horizontal: 6, vertical: 2` at theme/block_chrome.dart:130 — inside the theme layer itself. Fourteen of the 31 are on-scale numbers that simply evaded tokenization; fifteen are off the law entirely. The wave added `color_literal_scan_test.dart` for hex paint (I ran it: green, and zero hex literals outside theme/) but nothing watches spacing, so a future scale retune in `cobalt.dart` would leave a third of the surfaces behind. `prd_page.dart:257` (height 640) and `revisions_table.dart:100` (width 320) are layout caps, not spacing — keep them, with a recorded exception.

**CQ-02 — PRODUCT.md vs the tree (coordinator lead 14, run).** PRODUCT.md:3 claims "Authority: canonical target product experience." What has no implementation end to end: **Product Branch / Explore direction / Merge** (PRODUCT.md:206-210, 222-225, 342-344 — `rg -i "product.?branch|explore direction"` over server and client finds only section-map merging; the 39 migrations hold no branch table); **the Decision Log as a surface** (218-221 — it exists as proposal_decisions rows and a free-text `--note` flag, `cli_version.go:124`, never as the who/why/evidence/alternatives record with a view); **scoped feedback invitations for non-members** (254-259 — the invitations table is workspace signup only); **persona grounding labels** (123-134 — personas are a free-text `users` section; no core/key/later prioritization or grounded/synthesized/assumed machinery); **per-role Roadmap lenses** (277-288 — audience projections exist for PRDs in `audienceview.go`, not for the roadmap graph). Implemented and verified: severity-ranked critique with batched minors (critic.go:139, prompts.go:62,141), deadline pressure bands (roadmap.go:37 → roadmap_progress_view.dart:173-179), audience-specific readiness (prd_readiness.go). The status section (PRODUCT.md:424-431) discloses only the roles gap — the other five diverge silently.

**CQ-03 — the wave's one form-primitive bypass.** `showBlockForm` (theme/block_form.dart) is the shared labeled-fields dialog the wave hardened (blank-confirm stays open, f16aff9a). Both `_snapshot` copies hand-roll the same modal for a single optional note field, and the copies have already drifted once in guardWrite shape. The fix is a shared function both pages call — or simply `showBlockForm` with one non-required field.

**CQ-04 — retry loop, pasted not folded.** The wave centralized the retry *policy* (retryStreamChat, taxonomy.go:53) but left its *call shape* pasted three times — taxonomy.go:34-43, responses.go:55-63, responses.go:66-78. Any change to the no-replay rule touches three files or the streamers disagree.

**CQ-05 — the dead-code survivor.** The wave deleted the dead self-pointer filter (c50cf46); `ClearVar` is the same class, missed. I swept every unexported and package-only-exported identifier; this is the only production-dead one (`ErrLocalNotImplemented` is borderline but documented as Run's purity invariant).

**CQ-06 — two copy-paste seams.** `_ActionCard` (home_field.dart:272) vs `_navCard` (workspace_page.dart:317): identical 17-line tile. `_delete` at workspace_page.dart:135 vs settings_page.dart:78: same confirm-and-delete orchestration. My block scan counts 24 duplicated 8-line blocks between the two workspace files.

**CQ-07 — comment drift.** coach.go:226 describes the ledger as future work; prompts.go:39,78 show it riding in systemPrompt today. flow.go:84's error string says local mode is unimplemented while `cli_setup.go:182` dispatches the fully wired local path.

### What came back clean

Bare error returns: 0 in `internal/api` (every failure is a named `httpx.WriteError`; 118 logged sites). Print-vs-log: correct split — CLI prints, server logs slog. Naming across the new splits: consistent (`Block*` chrome, `Cobalt*` tokens, `readGate`/`guardWrite` verbs, `mcpserver` behind `cmd/insideout-mcp` with the rationale documented at server.go:1-4). Form validation: centralized in `showBlockForm`/`BlockFormField.required` except CQ-03. Read gates: one helper, ten pages. No further dead self-filters of the c50cf46 class — the surviving identity/role checks in client code are legitimate fallbacks.

**Coordinator lead 15 (run): mail funnel.** Code-side it is wired end to end: `mail.go` dev mode logs recipient/subject but never token bodies by default, and the disabled state is disclosed — `auth.go:46` reports `mailEnabled`, consumed at `auth_form.dart:65` with the annotated-not-hidden UX pinned by `auth_form_mail_off_test.dart` (USER-04). The remaining gap is operational (hosted provider never exercised; owner item), not a tree defect — no finding filed.

### The two mechanical cleanups worth a commit each

1. **Client: spacing tokens + the missing guard half.** Replace the 31 raw numeric gaps with `Cobalt.spaceN` (record exceptions for 640/320), decide nearest tokens for the 15 off-scale values, and add the numeric-spacing scan beside the hex-color scan in `color_literal_scan_test.dart`. Acceptance: the new test goes red on the current tree, green after the sweep; ~50 lines.
2. **Server: fold the retry wrapper.** One `withStreamRetry(onDelta, ...)` helper in taxonomy.go replacing the three pasted `streamed := 0 / countDelta := ...` closures; `retry_stream_test.go` and `go test ./internal/agent/` (verified `ok -count=1` here) pin the policy; ~30 lines.

Both are behavior-preserving, both are guarded by existing tests, and neither requires a design decision beyond choosing the nearest token for six values.

## What this role would do first

Land the spacing normalization + guard extension (CQ-01): a ~50-line mechanical diff whose acceptance is `flutter test test/color_literal_scan_test.dart` going red on the old tree and green after the sweep — I ran that file plus leftover_scan_test.dart on 754c502 (3/3 passed), so the harness is verified.
