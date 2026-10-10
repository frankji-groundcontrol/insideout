# BUG-029 — the Versions eyebrow counted revisions, not commits

## Symptom

The line above the commit stack, "Commit {n} of {total}", read "Commit 3 of 2"
for the demo PRD (zh: 第 3 次提交，共 2 次): a position larger than the count.
Found by the 2026-10-10 web walk.

## Root cause

`VersionsStack` passed `selected.revision` as `n`. A commit's `revision` is the
PRD revision it pinned, and every edit makes a revision, so commits pin numbers
with gaps (the demo PRD's two commits pin revisions 2 and 3, a PRD edited
between commits can pin 2 and 5). The column is `UNIQUE (prd_id, revision)`
(`prd_commits` migration), so the pinned number says which edit was committed,
never which commit this is. Introduced with the commit stack in the Cobalt
restyle (`cc139a8`); no test pinned the line.

## Fix

| Step | Commit |
| --- | --- |
| Red test: newest of two commits pins revision 5, the older pins 2. The page must say "Commit 2 of 2", then "Commit 1 of 2" after the older commit is brought forward | `112593d` |
| Fix: `n` is the number of listed commits at or before the front one | `d2fc678` |

## Verification

- `versions_commit_card_test.dart`: `the eyebrow counts commits, not the
  revisions they pin`. Red for the intended reason (no "Commit 2 of 2"; the page
  said 5) with the older commit's card present.
- Mutations: `<` for `<=` and `commits.length` for the count both fail it.
- Real browser, release build: the demo PRD's Versions page reads 第 2 次提交，共
  2 次 for the newest commit.

Known ceiling, unchanged: the server lists at most the newest 100 commits of a
PRD (`ListPrdCommits`, `LIMIT 100`), so the line counts within that list.

## Status

**Fixed 2026-10-10.**
