# 13 · Project Manager

**Lens:** 
**Score:** **6.5 / 10**
**Bottom line:** CI now real and strict, README honest, waves 0-4 verifiable — but the release gate is still open and grown (production behind main by all five remediation commits), and the PRODUCT.md-vs-code gap is the roadmap: Decision Log, Product Branches/Merge, per-project authority roles, personas, Lenses, exceptions-first lead home, facts panel, feedback invitations, overdue-reason flow are absent or partial.

Reviewed main @ 26db979; a skeptic re-checked every High finding from this review. [Correction] PM-13's headline claim — "all five remediation commits are undeployed" — was refuted by execution evidence: after 26db979 both Railway services were deployed (server `railway up` reached SUCCESS in the deployment list; the app service was restarted and rebuilds from main) and the production domain answered 200 (landing) / 401 (API without token). The row is downgraded to a Medium record-hygiene gap (no commit+date deploy record in the tree; the signed-in page load remains a human-only gate). PM-10, PM-11 and PM-12 were downgraded High→Medium: PRODUCT.md's own header frames them as end-state vision, not current promises.

---

This review was persisted findings-only; the section below is assembled verbatim from its structured findings (the prose wrapper was not retained). 12 findings as filed, worst first.

## Findings

### PM-13 · High ·  **Skeptic: DOWNGRADED → Medium.**

**All five remediation commits are undeployed and the PM-02 signed-in hosted verify is still open, so the hosted product is behind main by everything a stranger would check.**

- Where: `docs/changelogs/2026-10-04-remediation-wave-3-4.md:124`
- Evidence: changelog wave-3-4:124-126 'Deploy: both services changed' is an instruction, not a record; HANDOFF.md:14-21 'Still open, both human-only: … the signed-in hosted page load (PM-02 release gate)'.
- Fix: railway up server, restart app, signed-in page load on the domain, record deploy commit + verify date in HANDOFF item 1, board row, dated changelog.
- Diff size: S (ops + ~10 docs lines)

### PM-10 · High ·  **Skeptic: DOWNGRADED → Medium.**

**Decision Log — PRODUCT.md end-state loop step 5 — is absent; only a single free-text decisionNote string exists on Commits.**

- Where: `server/internal/api/prd_commits.go:31`
- Evidence: PRODUCT.md:218-220 'A Decision Log accompanies meaningful Commits, Merges, overrides, and reversed decisions…' vs prd_commits.go:31 DecisionNote string only; no decision-log table/route/UI.
- Fix: Slice: a decisions table keyed to PRD (+ future branch/merge targets) with actor, kind, rationale, evidence refs, alternatives; GET /api/v1/prds/{id}/decisions; render under the versions stack.
- Diff size: L (~5 files)

### PM-11 · High ·  **Skeptic: DOWNGRADED → Medium.**

**Product Branches and human Merge have zero implementing surface — grep confirms no code anywhere.**

- Where: `PRODUCT.md:208`
- Evidence: PRODUCT.md:208-211 'Product Branch / Explore direction… Merge / Combine conclusion' — grep productbranch|product_branch across server/ and client/: 0 hits.
- Fix: Slice 1: fork a PRD at a Commit into a Branch row (same revision machinery, parent pointer + driver), render in versions_stack; slice 2: human-initiated Merge with three-way summary + Decision Log entry.
- Diff size: L

### PM-12 · High ·  **Skeptic: DOWNGRADED → Medium.**

**Per-project authority model (Decision Owner / Driver / Contributor / Reviewer / Viewer) is absent; only workspace admin/member exists.**

- Where: `server/internal/store/memberships.go:35`
- Evidence: PRODUCT.md:229-240 'Product authority is assigned per Project' with five roles vs memberships.go:35 workspace role only; 'Driver' appears solely as a comment alias in prd_commits.go:50.
- Fix: Add project_members with the five roles, creator→Driver, Driver assigns Decision Owner; gate commit/milestone confirmation on Driver+Decision Owner.
- Diff size: L

### PM-14 · Medium · 

**Persona model — loop step 3 — absent; 'persona' exists only as copy inside audience-view purpose strings.**

- Where: `server/internal/audienceview/audienceview.go:39`
- Evidence: PRODUCT.md:123-133 persona records + grounding labels vs only audienceview.go:39 purpose text and a 'named audience' coach fact (agent/tools.go:264).
- Fix: Model personas as structured facts first (fact ledger + kind/tier/grounding), render a personas block in the management view; defer CRUD.
- Diff size: M

### PM-15 · Medium · 

**The coach is still the fixed four-stage machine PRODUCT.md explicitly supersedes; only the prompt layer carries the adaptive contract.**

- Where: `server/internal/agent/tools.go:244`
- Evidence: PRODUCT.md:88-90 'adapts interview depth… does not force a fixed number of questions' vs tools.go:244-294 mechanical sequence gates ('cannot draft yet', 'cannot critique yet', 'cannot finalize yet').
- Fix: Direction: keep fact/draft grounding rules but decouple from stage order — allow draft/critique on any non-empty content, make advance_stage advisory. First check: agent/tools.go gate table and its tests.
- Diff size: M

### PM-16 · Medium · 

**Roadmap Lenses and the exceptions-first lead home are both absent.**

- Where: `client/lib/features/dashboard/dashboard_page.dart:20`
- Evidence: PRODUCT.md:277-287 five Lenses (only agent context exists); PRODUCT.md:400-403 'exceptions-first… not raw commit volume' vs dashboard_page.dart:20-24 a workspace list + create dialog; no /overview or lens routes.
- Fix: First slice: GET /api/v1/overview aggregating per-workspace overdue/high-risk counts + latest evidence per project (reuse roadmaptime.Assemble), rendered as an exceptions strip above the board.
- Diff size: M

### PM-17 · Medium · 

**Internal evidence experience (per-fact status + confirm/correct/remove/discuss) and scoped feedback invitations are absent.**

- Where: `client/lib/features/prd/coach_panel.dart:81`
- Evidence: PRODUCT.md:183-188 per-fact actions vs coach_panel.dart:81-84 a snackbar onFactRecorded; no GET facts route; PRODUCT.md:254-256 scoped feedback invitations — grep feedback: zero product surfaces.
- Fix: Slice: GET /api/v1/prds/{id}/facts + a facts panel with confirm/correct/remove mapped to existing ledger mutations; feedback invitations need a design pass first.
- Diff size: M / L

### PM-18 · Medium · 

**Deadline states exist, but the overdue human-decision flow (continue/change/transfer/abandon with reason) and the Git-required development gate are absent.**

- Where: `server/internal/roadmaptime/roadmaptime.go:14`
- Evidence: roadmaptime.go:14-17 pressure states; grep rebaseline/abandon-reason/overdue-reason: 0 hits; PRODUCT.md:299-301 'Overdue work is never silently closed…'; PRODUCT.md:308-310 Git-required-at-Focus with no enforcement.
- Fix: Small slice: on PATCH roadmap node leaving an overdue state, require a reason string; Git gate = server check on execution item→in_progress without a bound repo.
- Diff size: M

### PM-19 · Medium · 

**PM-05's tag/CODEOWNERS half remains: no git tag marks any verified point and no CODEOWNERS exists.**

- Where: `.github/workflows/ci.yml:1`
- Evidence: git tag empty at main@26db979; no CODEOWNERS in git ls-files; CI half is done.
- Fix: Tag v0.2.0 on the commit verified in PM-13 (annotated, changelog index as body); add a one-line CODEOWNERS.
- Diff size: S

### PM-20 · Low · 

**Board 'live' claim still overstates: pull-to-refresh only, no poll/SSE on the board.**

- Where: `README.md:23`
- Evidence: README.md:23-24 '实时显示进度… live progress' vs workspace_page.dart:87-88 RefreshIndicator only; no Timer/poll/SSE on the board.
- Fix: Add a 30-60s poll or reword to '刷新查看最新动态 / refresh for the latest'.
- Diff size: S

### PM-21 · Low · 

**Fully-merged cursor/vibrant-ink-seal-198b still lives on origin; License bullet still missing from README.**

- Where: `refs/remotes/origin/cursor:1`
- Evidence: git branch -r --merged main includes origin/cursor/vibrant-ink-seal-198b (tip 8de2730, ancestor of PR #3 merge); README.md:85-90 Learn More has no License entry (LICENSE:1 MPL-2.0).
- Fix: git push origin --delete cursor/vibrant-ink-seal-198b; add one License bullet under Learn More.
- Diff size: S

