# 22 · Documentation index and Marksman

**Lens:** the index hierarchy, reachability, router honesty, Marksman diagnostics
**Score:** **6.2 / 10**
**Bottom line:** The documentation graph is structurally sound (657 of 659 markdown files reachable from the four routers, every docs/ file within four hops of docs/index.md, zero broken relative links in tracked files), but the routers that answer "what is open" are not honest: the task board has no row for the only work in flight, the architecture index fails its own gate on the committed tree, the two scripts that once proved "zero orphans" were never committed, and an orphan plan plus a disagreeing changelog index have already reappeared. Marksman's 319 ambiguity diagnostics are 298 bare links to non-unique file names plus 21 links to three documents that carry two H1 headings; a two-line .marksman.toml and a "./" authoring rule take the count to zero in a scratch copy. The recorded growth "57 to 319" compares different units: like for like it is 236 to 319. Not checked: fragment anchors with a GitHub-exact slugger (my slugger over-reports; I report only the three I confirmed by reading), CI run status on the pushed tip, and the content accuracy of individual records.

Read at `main` @ `4d186da` together with the uncommitted working tree. A skeptic re-checked every High in this file; the verdicts are in the table near the end and in the [findings register](./01-findings-register.md).

---

## Scope read

docs/index.md; every README.md and index.md under docs/ to two levels; docs/changelogs/index/; docs/HANDOFF.md, CLAUDE.md, AGENTS.md, README.md, SKILL.md; the 2026-10-11 indexing pass (plan and changelog); and the coordinator's Marksman output. Four things were run outside the repository under /tmp/insideout-rv5/: a link-graph script (linkgraph.py) over 651 tracked and 8 untracked markdown files; Marksman on a small scratch workspace to learn its resolution rules; Marksman on a copy of the tree with the proposed cure applied; and Marksman on the 2026-10-11 tree (9f0b1c7) for a like-for-like count.

## What is good

- 657 of 659 markdown files are reachable from README.md, CLAUDE.md, AGENTS.md or docs/index.md. Every docs/ file is at most four hops from docs/index.md (20 at one hop, 453 at two, 152 at three, 10 at four).
- scripts/tests/check_doc_links.py exits 0 and runs in CI through the glob at .github/workflows/ci.yml:347.
- The uncommitted documents are indexed: docs/usage/README.md:7-8, docs/changelogs/README.md:31-33, docs/design-qa/README.md:24.
- docs/changelogs/index/2026-10.md shows the month-file pattern that keeps a 118-record folder's README at 155 lines.
- No index file exceeds 350 lines.

## Marksman: what the 319 diagnostics are

| Cause | Count | Why Marksman reports it | Cure |
| --- | --- | --- | --- |
| A. Link without a leading `./` or `../` to a name that is not unique | 298 | Such a path is matched as a suffix across the whole workspace; `./name.md` and `../x/name.md` are exact | Authoring rule; no setting exists |
| B. Exact relative link to a document with two H1 headings | 21 | A document link resolves to its title headings; two H1s are two targets | `title_from_heading = false` |
| Code 2, register line 123 | 2 | `[3-9][0-9]` in a table cell reads as a reference link labelled `0-9` | Backticks around the two patterns |

Cause A by name: 150 numbered review reports reused by five review folders; 104 dated names shared by docs/plans, docs/changelogs and docs/design-qa; 24 README.md, index.md and keys/README.md; 8 deployment.md; 8 verification.md; 4 summary.md. Cause B is three registers (9, 5 and 7 links). On the scratch copy the setting took 319 to 298 and the `./` prefix took 298 to 0.

The count did not grow from 57 to 319. The 2026-10-11 record counted distinct basenames (57) and documents (82). The same driver on that tree reports 236 diagnostics; today it is 319. Of the 83 new ones, 66 come from docs/review/2026-10-18-multirole-4. Growth stops when the report template writes `./` and a gate enforces it.

## Findings, worst first

- **IDX-01 (High)** docs/plans/README.md:66 — "**Status meanings:** In flight = active work; Pending = not started; Finished". No row is In flight, yet 57 files of install and password-reset work are uncommitted with no plan. 53 of 54 rows are terminal (already open as rv4-PM-09 / R3-36, now worse). Fix: add the plan and row; archive finished rows by month and repoint the two guards.
- **IDX-02 (High)** docs/architecture/backend.md:147 — "- accounts: `users.go`, `sessions.go`, `auth_tokens.go` (magic links, resets,". check_docs_truth_rv4.py exits 1 with four misses; three are committed files, so the gate is red on origin/main. Fix: name the four files.
- **IDX-03 (Medium)** docs/plans/README.md:14 — the review-4 row does not link docs/plans/2026-10-20-rv4-wave-4b/, which has no inbound link on either main. The orphan scripts of 2026-10-11 were never committed. Fix: link it; commit the gate.
- **IDX-04 (Medium)** docs/HANDOFF.md:25 — "**2026-10-20: wave 4b is on local `main`, not pushed.**" Local main is two commits behind origin/main, where this is corrected. The claim sits in four routers; three are modified on both sides. Fix: commit, rebase, state push state once.
- **IDX-05 (Medium)** docs/changelogs/2026-10-11-docs-order-marksman.md:22 — "hard broken = 0, 57 advisory". Cause A above, uncured; the noise hides a wrong target at docs/issues/2026-10-11-changelogs-readme-line-budget.md:5. Fix: `./` rule, rewrite 298 links, gate.
- **IDX-06 (Medium)** docs/changelogs/README.md:25 — "Every record file and dated folder directly in this folder, newest first" omits 2026-10-12-remote-setup-auth.md. Fix: delete the flat list.
- **IDX-07 (Medium)** docs/architecture/index.md:19 — no current-state document for roadmap node ownership. Fix: add docs/architecture/roadmap-tree.md.
- **IDX-08 (Medium)** docs/index.md:30 — the map omits SKILL.md, skills/insideout-server, hosts/ and four root policy files, and lists the stale docs/PLAN.md. Fix: add a group, delete PLAN.md and INIT.md.
- **IDX-09 (Medium)** skills/insideout-setup/references/modules.md:5 — "(../SKILL.md#step-0--platform-and-destination)" points at a heading the uncommitted rewrite removed; fragments are unchecked. Fix: repoint; check fragments.
- **IDX-10 (Medium)** docs/issues/README.md:12 — "**State on 2026-10-11:** every record is fixed, superseded or closed with". Open state is dated prose. Fix: an Open table plus index/ sub-files.
- **IDX-11 (Low)** docs/review/2026-10-04-multirole/01-findings-register.md:7 — "# Finding register", a second H1. Fix: the setting below.
- **IDX-12 (Low)** docs/review/2026-10-18-multirole-4/01-findings-register.md:123 — "go1\.(2`[6-9]`\|`[3-9]` `[0-9]`)". Fix: backticks.

## Three-hop test

| Question | Path | Result |
| --- | --- | --- |
| What is in flight now? | CLAUDE.md, docs/plans/README.md | Fails: no row |
| Which review-4 findings are open? | docs/index.md, docs/issues/README.md, 2026-10-18-review-4-open-findings.md | Passes, through prose |
| How does roadmap node ownership work today? | docs/index.md, docs/architecture/index.md | Fails: no record |
| What did the reset watch change? | docs/index.md, docs/changelogs/README.md, 2026-10-21-reset-watch.md | Passes |
| How do I install as a user? | README.md, docs/usage/user-guide.md | Passes |

## Lead verification

| Lead | Verdict | Deciding line |
| --- | --- | --- |
| 8 | Confirmed, understated | The gate prints four misses; `git ls-files` lists three of the files |
| 10 | Counts confirmed; growth refuted | Same driver on 9f0b1c7: "total_diagnostics=238", 236 of them code 1 |
| 10 | Folder split corrected | 157 review, 84 changelogs, 53 plans, 10 usage, 6 issues, 3 architecture, 3 design-qa, 1 learning, 2 hosts |
| 11 | Confirmed | docs/HANDOFF.md:9 "never history" against dated paragraphs at lines 65-78 |
| 12 | Partly refuted | `git status -sb`: "## main...origin/main [behind 2]" |
| 14 | Applied | R3-36 at docs/issues/2026-10-18-review-4-open-findings.md:167 |

## Recommendations

1. One shape for every record folder: a README holding the convention and what is open, and index/ or archive/ month files for the rest.
2. State push and deploy state in docs/HANDOFF.md only.
3. Put the `./` rule in the review report template.
4. Check heading fragments and untracked files in check_doc_links.py.
5. Append a correction to the 2026-10-11 record: 57 was basenames, 82 was documents, 236 was diagnostics.

## What this role would do first

Add the In flight row and plan for the uncommitted slice and name the four missing files in backend.md; those two edits restore the routers a maintainer reads first. Then add this file at the repository root:

```toml
# .marksman.toml
[core]
title_from_heading = false
```

The one gate that keeps the index honest is scripts/tests/check_doc_graph.py, run by the existing CI glob. It fails when a tracked markdown file is unreachable from the four routers, when a record folder's index does not link a direct child, or when a link without `./` or `../` matches more than one file.

## Skeptic verdicts on this lens's Highs

| ID | Verdict | Reason | What the skeptic ran |
| --- | --- | --- | --- |
| IDX-01 | DOWNGRADED to Medium | The facts hold: the board has 54 rows, one Pending and none In flight or Blocked, and no plan under docs/plans covers the slice (only this review's track file and an unrelated 2026-10-07 plan match). That breaks CLAUDE.md's rule to open a dated plan before multi-step editing, and a `[checkpoint]` commit of the slice would block without one. It is not High: the slice is not invisible. The working-tree docs/HANDOFF.md leads its "What to do next" section with it and links the two changelog records, origin/main's HANDOFF.md says the machine holds "a whole uncommitted feature slice (install front door, reset watch, reset code — 35 tracked files plus new files)", and git status shows 59 dirty paths, so the collision scenario needs a reader who skips both the handoff and git status. No runtime behaviour is wrong. The archive half restates the open record R3-36 (rv4-PM-09) in docs/issues/2026-10-18-review-4-open-findings.md, which that record itself rates "Unowned; low". | `grep -n "In flight\\|Blocked\\|Pending" docs/plans/README.md` -> lines 16, 66, 68 only (one Pending row plus the legend); status tally of the 54 P-rows -> 1 Pending, the rest Completed/Finished/Fixed/Done/Closed/Exercised plus one "Waves 0 to 4 on origin/main"; `grep -rln "reset-watch\\|install-front-door" docs/plans` -> 2026-11-21-focused-review-5.track.yaml and 2026-10-07-setup-tui.md only. |
| IDX-02 | CONFIRMED | A red gate on the pushed tip, confirmed three ways. The gate fails on a clean clone of 4d186da and of c2900fd with three misses (conversation_retention.go, erasure.go, me_delete.go; the fourth, reset_watch.go, is untracked and appears only in the working tree). The real CI run on origin/main c2900fd failed the job "env catalog suites (python)" at the step "contract guards (scripts/tests/check_*.py)" with those same three lines, and every other guard in the step printed ok. The three files entered in commits dated 2026-10-20 (d5a45fe, 3ecef69) and backend.md does not differ between HEAD and origin/main. No docs/issues record covers it; origin/main's HANDOFF.md records only the security workflow as red and says the ci run "was still in progress at recording time". The same CI run is also red in two other jobs (server test step, client analyze step), so fixing this gate alone does not make the tip green. | `gh run view 44787948546 --json jobs` (ci on c2900fd) -> "env catalog suites (python) failure / STEP contract guards (scripts/tests/check_*.py) failure"; `--log-failed` -> "backend.md's Store layer section does not name server/internal/store/conversation_retention.go", "... erasure.go", "backend.md's API surface section does not name server/internal/api/me_delete.go", "##[error]contract guard failed". Local: the gate loop in a /tmp clone at 4d186da and at c2900fd -> "FAIL 1 scripts/tests/check_docs_truth_rv4.py" with the same three lines, all other gates pass. |

## Ordered recommendations

1. Adopt one target shape for every record folder, already proven by changelogs: a short README that states the convention and holds only what is open or current, plus index/ sub-files for the rest. Concretely: docs/plans/README.md keeps In flight, Pending and Blocked rows and moves finished rows to docs/plans/archive/YYYY-MM.md (closes R3-36); docs/issues/README.md keeps an Open table and moves the bug book and closed records to docs/issues/index/; docs/changelogs/README.md drops its flat list and keeps the month files; docs/learning/README.md (77 lines for 69 records) stays as it is until it passes about 150 lines.
2. State each volatile fact once. Push and deploy state belongs in docs/HANDOFF.md only; the board, the month index and the issue index link to it instead of repeating it. One fact in four hand-edited routers is why the 2026-11-21 sync had to edit four index files and why three of them will now conflict.
3. Commit the reachability check instead of running it from a temporary directory: scripts/tests/check_doc_graph.py, picked up by the existing glob at .github/workflows/ci.yml:347, under 150 lines, reusing the tracked-file listing from check_doc_links.py.
4. Add .marksman.toml with title_from_heading = false and adopt the rule that a sibling or child link starts with ./ ; put the rule in the review report template, because each templated review folder adds about 66 diagnostics.
5. Add docs/architecture/roadmap-tree.md so the owner's central design question has a current-state answer two hops from the documentation map, and remove docs/PLAN.md and docs/INIT.md, which only the map links to.
6. Extend scripts/tests/check_doc_links.py to check heading fragments in undated documents and to include untracked markdown files, so an uncommitted slice cannot break anchors unnoticed.
7. Correct the record of the last pass with an appended note in docs/changelogs/2026-10-11-docs-order-marksman.md: 57 was distinct basenames and 82 was documents; the diagnostic count on that tree was 236.
