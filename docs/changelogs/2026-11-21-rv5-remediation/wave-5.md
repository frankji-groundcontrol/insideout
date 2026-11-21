# Wave 5 · The documentation index

Rows 5.1 to 5.6 of the
[remediation plan](../../review/2026-11-21-focused-review-5/40-remediation-plan.md),
in two clusters. The doc-records cluster (rows 5.3, 5.6) ran beside wave 3,
because it shared no file with it. The doc-graph cluster (rows 5.1, 5.2, 5.4,
5.5) ran last, because its link rewrite touches every markdown file.

## Rows

| Row | IDs | Disposition | Commits |
| --- | --- | --- | --- |
| 5.1 | IDX-05, IDX-11, IDX-12 | FIXED: [`.marksman.toml`](../../../.marksman.toml) sets `title_from_heading = false`, and 1498 sibling and child links in 289 files now start with `./`. The link graph has the same 2860 edges before and after. The `./` rule is in the review convention and a [practice](../../practices/README.md). The go-floor patterns in review 4's register row PM-12 are set as code. Marksman reports 0 diagnostics (288 before); a control file with a broken and an ambiguous link still raises both, so the 0 is real | `66747b2d`, `ca8cd8bc`, `cb5aa72b`, `adf29c8d` |
| 5.2 | IDX-03 | FIXED: [`scripts/tests/check_doc_graph.py`](../../../scripts/tests/check_doc_graph.py) fails on a markdown file not reachable from the four routers, an index that misses a direct child, and a bare link that matches more than one file. Two exceptions remain, each with a reason. It failed with 297 problems on the tree before the fixes; now all 700 files are reachable and 47 indexes are checked | `0e7f218b`, `3d96f51e`, `3b648398` |
| 5.3 | IDX-01, IDX-10, IDX-06 | FIXED: the task board keeps only open rows, and finished rows moved verbatim to [month files](../../plans/index/README.md). The issue index opens with a table of what is open. The changelog month files are the only changelog index | `dee943f2`, `a9f5e104`, `aaa0f31d` |
| 5.4 | IDX-08 | FIXED: `docs/index.md` maps `SKILL.md`, both skills, `hosts/` and the policy files. The stale `docs/PLAN.md` is deleted; `docs/INIT.md` stays, because it is accurate and a dated record links to it | `a0472101` |
| 5.5 | IDX-09 | FIXED: [`check_doc_links.py`](../../../scripts/tests/check_doc_links.py) checks heading fragments in undated documents and includes untracked files. Its first run found one real broken anchor, in the key setup tutorial | `387b4c58`, `dae3388a` |
| 5.6 | FEAT-04, FEAT-11, FEAT-07, FEAT-08, FEAT-10, FEAT-12, FE-12 | FIXED: the per-Project roles now have a [record](../../issues/2026-11-21-per-project-roles.md). The promise-register rows that claimed more than the tree carry a dated note. The deployed service hostnames are placeholders | `3e54bea6`, `1033ecf0`, `0b77d3c7` |

## Gates on the integrated tree

See [the final gate run](./index.md#final-state).

## Left open

- The promise register keeps its PARTIAL and IMPLEMENTED stamps with a dated
  note rather than new stamps. Changing a stamp would trip the promise-gap
  guard's rule that each one link an issue.
- `check-docs-recorded.sh` still counts a staged `docs/plans/index/` file as
  the active plan for a `[checkpoint]` commit, as it already did for changelog
  index files.
