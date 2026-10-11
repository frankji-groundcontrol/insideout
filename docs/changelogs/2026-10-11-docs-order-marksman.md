# 2026-10-11 — Docs order: survey, READMEs, index consistency, Marksman cross-references

**Status: landed 2026-10-11** at this commit. Plan:
[2026-10-11-docs-marksman-order](../plans/2026-10-11-docs-marksman-order.md)
(closed — all five owner goals verified).

Brings all ~505 tracked markdown documents under one order per the owner's
goal: every document read and summarized, key folders described, cross-agent
record inconsistencies resolved, and **zero orphan documents** — verified by
both a deterministic link-graph script and a headless Marksman LSP driver.
Executed as multiple dynamic workflows.

## What has landed so far

- **Census + gates** (`/tmp/insideout-gap/docs/graph.py`, `marksman_lsp.py`):
  505 tracked docs; 20 orphans, 14 doc-to-doc broken-link occurrences, 70
  dead non-markdown links — of which the dated-record citations of the
  deleted Nuxt `app/` tree stay as-is by the historical-record policy, and
  5 occurrences are verbatim evidence quotes inside dated review records
  (documented `QUOTED_EVIDENCE` exemption; a quote is the evidence, never
  rewritten to "fix" a link). Marksman 2026-02-08 driven headlessly over
  the whole tree via LSP stdio: hard broken = 0, 57 advisory
  ambiguous-name warnings (duplicate basenames; the repo's standard
  relative links are unaffected).
- **Workflow C1 — link surgery** (dwfrun-b3fb78a0): 4 file-disjoint
  clusters, 16 files changed, `docs/review/2026-10-06-frontend-cobalt/raw/
  README.md` created describing the 10 raw reviewer outputs; the
  born-broken `review-sweep-ci-and-infra.md` target (never in git history)
  repointed to the in-file anchor that holds that content. Verified:
  adversarial diff review passed **and** the coordinator independently
  re-ran both gates on the final tree (`orphans=0 md_broken=0`,
  marksman hard=0) — necessary because the workflow's internal gate had
  run before the verifier's final repair round (lesson recorded in
  [learning](../learning/2026-10-11-rerun-gates-after-a-workflows-last-repair.md)).
- **Survey** (workflow A, dwfrun-6061d2b7): 17/17 reader batches, 506/506
  documents read — per-file one-line English summaries plus observations
  as JSON (204 total: 38 suspected inconsistencies, 35 stale, 35 missing
  refs), all deliberately unconfirmed so C2 verified each against the
  tree first. Triage rules established: living docs fixed outright;
  same-day factual contradictions corrected via appended notes;
  later-overtaken dated records get supersession annotations, never
  rewrites. Per-batch JSON in `/tmp/insideout-gap/docs/survey/`.
- Debt filed: [changelogs README exceeds the line
  budget](../issues/2026-10-11-changelogs-readme-line-budget.md) (473
  lines) — the bounded split executes in C2.
- **Workflow C2 — semantic fixes, settled** (dwfrun-6b72b856): six
  file-disjoint clusters, 151 files changed/new, 21 observations verified
  and skipped. Living docs corrected outright (Go 1.26 everywhere; the
  MCP tool count no longer frozen — the registry stood at 52 while docs
  said 50, counted from `addXTool` registrations; v0.1.0 and the public
  repo stated as fact); same-day contradictions annotated
  `*Corrected 2026-10-11*` (multirole-2's 105/106/108 counts reconciled
  by recounting rows; FEAT-03 ID collision disambiguated; 23→31
  migrations; 30→32 clusters); ~25 later-overtaken records annotated
  `*Superseded*` linking the newer truth; changelogs README split into
  `index/YYYY-MM.md`; quoted evidence untouched; adversarial review
  passed with the post-repair gate `orphans=0 md_broken=0`. Its review-fix
  round also restored every dated-record original sentence an earlier
  round had rewritten in place (corrections live only in appended italic
  notes), added the never-listed 2026-07-23-backend-version-control
  bullet (106/106 records each linked from exactly one month index), and
  moved BUG-011's banner to the standard italic note. The four verifier
  findings were dispositioned by the coordinator: SKILL.md drift wording
  aligned to the evidenced registry state, and the three over-budget
  dated records got [a policy-bound line-budget
  issue](../issues/2026-10-11-historical-records-over-line-budget.md).
- **Workflow D — final audit, repair round 1** (dwfrun-c214ad44): the
  graph gate was already green (`orphans=0 md_broken=0`); the coverage
  gate read `missing=8 unlink=103` and the round closed it at the
  index/README layer only — 8 folder description READMEs created
  (`changelogs/index/`, `history/workshop-classes/`, the six
  `usage/keys/` module folders), the changelogs README gained a
  month-grouped flat index of all 101 direct record files
  (filename-per-line, generated from the tree so no typo can break a
  link), the plans README linked the two 2026-09-18 goal-prompt
  records, and the keys/history READMEs now link the new folder READMEs
  so nothing new went orphan. Verified: coverage `missing=0 unlink=0`,
  graph `orphans=0 md_broken=0` (526 files), docs-truth exit 0,
  redaction exit 0. Lesson: [a truncated gate snapshot can mislabel
  its sections](../learning/2026-10-11-truncated-gate-snapshots-mislabel-sections.md).
- **Workflow D — final audit, repair round 2** (dwfrun-c214ad44, sampling
  acceptance): acceptance sampling of round 1 flagged five residual index
  gaps; all fixed at the index/README layer only, no record statements
  rewritten — the frontend-cobalt review README now indexes its five
  pipeline artifacts (`reviews-wave1/2.json`, `reviews-all.json`,
  `build_result.py`, `result.json`); `shots-wave3/captures.md` gained the
  missing `11-landing-dark.png` row (30 PNGs = 30 table rows); the
  changelogs README Records list gained the five dated folders it omitted
  (each linked via its `index.md`, intro wording updated to say folders
  are listed too); `usage/keys/auth/` got the module README its six
  sibling modules already had (parent row repointed from the bare
  `auth/` link); and the history README's class-01-review mention became
  a real link. Every new target `test -f`-verified; a relative-link
  sweep over the six touched files found 0 broken; both gates re-run on
  the repaired tree: `GATE orphans=0 md_broken=0` (528 files),
  `COVERAGE missing=0 unlink=0`.

## Verification (coordinator's own runs on the final landed tree)

- `python3 /tmp/insideout-gap/docs/graph.py` → `GATE orphans=0
  md_broken=0` (528 files incl. untracked) — zero orphan documents,
  zero broken doc-to-doc links.
- `python3 /tmp/insideout-gap/docs/readme_coverage.py` → `COVERAGE
  missing=0 unlink=0` — every key folder carries a description file and
  links all its direct children.
- `python3 -I scripts/tests/check_docs_truth.py` → exit 0 (52 MCP tools
  registered; backend package count agrees).
- `sh scripts/check-redaction.sh` → exit 0. (The audit workflow logged
  `redaction exit=127` — a wrong path in the audit script itself
  (`check-redaction.sh` vs `scripts/check-redaction.sh`), not a failing
  check; re-run with the right path it is green.)
- `python3 /tmp/insideout-gap/docs/marksman_lsp.py` → `files=528 …
  hard=0` (2 historical-allowed citations, 82 advisory ambiguous-name
  warnings — duplicate basenames, harmless for standard relative links).

## What an operator must do

Nothing — docs and records only, nothing deployed, no runtime surface
touched.
