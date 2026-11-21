# 2026-11-21 — Focused review 5

**Status: delivered 2026-11-21.** Documentation only; no source file changed.
Plan: [2026-11-21-focused-review-5](../plans/2026-11-21-focused-review-5.md).
Report: [docs/review/2026-11-21-focused-review-5/](../review/2026-11-21-focused-review-5/README.md).

## What was done

- Reviewed `main` @ `4d186da` together with its 57 uncommitted files on the
  owner's five questions: feature completeness, consistency across the web
  app, REST API, MCP server and CLI, the task tree for roles, code and file
  structure, and the documentation index inspected with Marksman.
- Ten reviewers on one finding schema returned 120 unique findings. Five
  skeptic clusters tried to refute the 29 filed as High: 20 confirmed, 9
  downgraded to Medium, 0 refuted. A completeness critic added 7 findings and
  grouped all of them into twelve root causes. Final count: 127 findings, 0
  Critical, 21 High (15 distinct defects), 81 Medium, 25 Low.
- The coordinator reproduced eleven results (R1 to R11 in the report README)
  on a private PostgreSQL 18 cluster with probe tests injected by
  `go test -overlay`, so nothing was written into the tree. Method:
  [practice](../practices/2026-11-21-probe-a-real-database-without-touching-the-tree.md).
- Wrote the report folder (README, executive summary, register, ten lens
  files, critic, remediation plan in six waves, task-tree target design) and
  linked it from the [review index](../review/README.md).
- Filed the open High findings and the owner's decisions in
  [the issue record](../issues/2026-11-21-review-5-open-findings.md).

## Verification

- `python3 -I scripts/tests/check_doc_links.py`: 0 broken relative links in
  tracked files; the same rule run over the new, untracked files: 574 links,
  0 broken. Twelve quoted lines in the report contain another file's link
  syntax; a space was inserted between `]` and `(` in each so the gate does
  not read them as links of the report.
- `./scripts/check-redaction.sh` exits 0. A key-shaped scan of the new files
  matches only the file name `41-task-tree-target-design.md` (the letters
  `sk-` inside `task-`); a hostname, address and token scan finds only the
  placeholder `a@example.org`.
- Marksman over the tree: no diagnostic comes from the new folder or the new
  records (every sibling link is written with a leading `./`); the tree total
  stays at 321.
- `git status`: the probes left no file in the repository.

## What was not done

- No finding was fixed. The pushed tip stays red in CI until row 0.1 of the
  [remediation plan](../review/2026-11-21-focused-review-5/40-remediation-plan.md)
  lands.
- The Flutter app was not run in a browser, the streaming endpoints were not
  exercised live, and the hosted deployment was not touched.
- `docs/architecture/` was not changed: the task-tree page is row 0.5 of the
  plan and waits for the owner's decisions.

## Operator action

None. Nothing was deployed and no migration was added.
