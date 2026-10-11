# 2026-10-11 — Docs order: survey, READMEs, index consistency, Marksman cross-references

Owner goal: with the Marksman LSP server installed, bring the 505 tracked
markdown documents under one order — every document read and summarized,
every key folder carrying a README that describes its contents (including
children), a sound index graph, agent-record inconsistencies resolved, and
zero orphan documents (Marksman-verifiable). Executed as multiple dynamic
workflows per the owner's standing preference.

## Checklist

- [x] Marksman installed (2026-02-08 via brew) and a headless LSP driver
      that can (a) collect diagnostics for broken links and (b) build the
      inbound-reference graph per document.
      (`marksman_lsp.py` smoke-tested over all 507 files: hard broken = 0,
      57 advisory ambiguous-name warnings; standard-relative-link gate
      stays with the graph script — Marksman's own broken-link diagnostics
      cover its catalog/wiki-links only.)
- [x] Workflow A — survey: reader agents over file-disjoint batches of the
      505 docs; each returns a per-file one-line summary + observations
      (stale claims, suspected contradictions, missing references); output
      `/tmp/insideout-gap/docs/survey-*.json`. **Done 2026-10-11**
      (dwfrun-6061d2b7): 17/17 batches, 506/506 docs read, 204
      observations (38 inconsistencies / 35 stale / 35 missing refs), all
      deliberately unconfirmed — C2 verified each before editing.
- [x] Graph + gap computation (deterministic script, run between A and C):
      link graph from markdown links, orphan list (no inbound repo links
      and not reachable from any index), missing-README folder list,
      candidate contradiction list from survey observations.
      (`graph.py`: orphans=20, md_broken=14, calibrated for the index.md
      convention and untracked files; prints a `GATE` line.)
- [x] Workflow C1 — mechanical link surgery: 4 file-disjoint clusters
      fixing the 14 breakages and indexing all 20 orphans, graph-gated with
      ≤3 repair rounds and adversarial diff review. **Done 2026-10-11**
      (dwfrun-b3fb78a0): 16 files changed, `raw/README.md` created,
      `review-sweep-ci-and-infra.md` (a born-broken target — never in git
      history) repointed to the in-file anchor holding that content.
      Coordinator re-verification caught the script ordering (gate ran
      before the verifier's last repair): the 5 links the verifier reverted
      are the verbatim evidence quotes covered by the policy below, and the
      gate script now carries a documented `QUOTED_EVIDENCE` exemption;
      independent re-run: `orphans=0 md_broken=0`, marksman hard=0.
- [x] Workflow C2 — semantic fixes: consistency triage from the survey,
      README completion (design-system README; changelogs README split per
      [the line-budget issue](../issues/2026-10-11-changelogs-readme-line-budget.md)),
      cross-reference weaving where docs mention other docs or key concepts
      without linking. **Done 2026-10-11** (dwfrun-6b72b856): 151 files
      changed/new across 6 file-disjoint clusters; 21 observations verified
      and skipped (already fixed or unfounded); living docs corrected
      outright, dated records annotated (`*Corrected/Superseded 2026-10-11*`
      with links, original sentences preserved), changelogs README split
      into `index/YYYY-MM.md`; adversarial review passed and the
      post-repair gate read `orphans=0 md_broken=0`. Verifier findings
      dispositioned by the coordinator: SKILL.md drift wording aligned to
      the evidenced registry state (52 tools), and
      [a policy-bound line-budget issue](../issues/2026-10-11-historical-records-over-line-budget.md)
      filed for the three over-budget dated records.
- [x] Workflow D — final audit: whole-set marksman diagnostics + graph
      gates (orphans=0, md_broken=0, marksman hard=0), README presence for
      every key folder, spot-check of summaries; verdict JSON.
      **Done 2026-10-11** (dwfrun-c214ad44): five gates + two sampling
      verifiers; the samplers' five verified gaps (frontend-cobalt README
      missing the five pipeline artifacts; shots-wave3 missing 1 of 30
      PNGs; changelogs README not linking the five dated folders directly;
      `usage/keys/auth/` the only module without a README; history README
      link for class_01_review) were fixed in-run and re-verified.
      Coordinator's own final-tree runs: graph `orphans=0 md_broken=0`
      (528 files), coverage `missing=0 unlink=0`, docs-truth 0,
      redaction 0 (`scripts/check-redaction.sh`; the in-run exit 127 was
      a wrong-path bug in the audit script, not a failing check),
      marksman `hard=0`.
- [x] Records: dated changelog (extended through D and finalized at the
      landing commit), plan closed, board row Finished, HANDOFF thread
      line closed, track completed; committed, pushed, CI + three-way
      host sync verified.

## Status

**Finished 2026-10-11.** All five owner goals delivered: (1) every
document read and summarized (17 survey batches, 506/506); (2) every key
folder carries a description README/index including its subfolders
(coverage gate `missing=0`); (3) the index graph is sound and every
folder's own index links all its children (`unlink=0`); (4) 204
cross-agent observations triaged — living docs corrected, dated records
annotated, 21 verified-and-skipped; (5) verified by the Marksman LSP
server (headless driver, `hard=0`) and the link-graph gate — **zero
orphan documents, zero broken doc-to-doc links** on the landed tree.

Census at start: 505 tracked .md:
changelogs 130, review 81, issues 69, plans 68, learning 53, usage 39,
design-qa 14, practices 11, history 7, architecture 7, root/misc rest).
Known suspects already: `docs/PLAN.md`, `docs/INIT.md`, `docs/TODO.md`
(likely pre-reorganization fossils), any "private repo" claims predating
today's public flip.

**Engineering review:** skipped as a separate multi-role pass — recorded
why per the repo rule: this is docs-only work with no runtime surface,
its correctness is decided by deterministic gates (the link-graph script
re-run per repair round, Marksman LSP diagnostics at the end) rather than
by judgment, and every prose change gets an adversarial verifier inside
the fix workflows. Measured census at start: 505 tracked docs, 20 orphans
(no inbound link), 14 doc-to-doc broken-link occurrences, 70 dead
non-markdown links of which the dated-record citations of the deleted
Nuxt `app/` tree stay as-is (historical-record policy, same as the board
note on the 2026-08-18 Nuxt deletion). The same exemption covers
verbatim evidence quotes inside dated review records: they keep the
quoted source's own link text (`docs/TODO.md` as the root README writes
it, `../../client/README.md` as docs/usage writes it,
`design-system/cobalt/README.md` as docs/index.md writes it) even where
the path does not resolve from the review file's directory — the quote
is the evidence, not a live link, and must never be rewritten to
"fix" it.
