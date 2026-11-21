# 2026-11-21 — Focused review 5: completeness, surface parity, role task trees, structure, indexing

The owner asked for a review of the work with five lenses: whether every
functional property is implemented, whether the web app, REST API, MCP server
and CLI stay consistent, whether each role has its own task tree (a node can be
delegated, opens as that person's own tree, and stays visible to a manager),
whether long files are modular, and whether the documentation index is
appropriate when inspected with Marksman.

The review reads `main` @ `4d186da` together with its 57 uncommitted files. It
changes no source. Its output is the report folder
[`docs/review/2026-11-21-focused-review-5/`](../review/2026-11-21-focused-review-5/README.md).

## Checklist

- [x] Scout: `main` is the product (2762 files on `origin/main`; local `main` is 0 ahead
      and 2 documentation commits behind); 14 leads, every cheap one run.
- [x] Coordinator runs: backend build, vet and tests without a database;
      Flutter analyze and tests; the four docs gates; a headless Marksman pass.
- [x] Reproduce the delegation claim on a private PostgreSQL 18 cluster with a
      probe test injected by `go test -overlay` (nothing written to the tree).
- [x] Fan out ten reviewers on one finding schema (two for completeness, three
      for surface parity, two for the task tree, two for structure, one for
      the index): 120 unique findings, 29 filed as High.
- [x] Skeptics refute every High (five clusters, probes allowed on the private
      database): 20 confirmed, 9 downgraded to Medium, 0 refuted. Completeness
      critic: 7 added findings (one High), score reconciliation, twelve
      root-cause clusters.
- [x] Reproduce the top findings (R1 to R11 in the report README).
- [x] Write, redact and index the report folder.
- [x] Close records: [changelog](../changelogs/2026-11-21-focused-review-5.md),
      [issue record](../issues/2026-11-21-review-5-open-findings.md),
      [practice](../practices/2026-11-21-probe-a-real-database-without-touching-the-tree.md),
      board row, HANDOFF.
- [ ] Remediation: not started. The order is the six waves of
      [40-remediation-plan.md](../review/2026-11-21-focused-review-5/40-remediation-plan.md);
      wave 2 waits for the owner's decisions in
      [41](../review/2026-11-21-focused-review-5/41-task-tree-target-design.md).

## Status

Review delivered 2026-11-21 and indexed: 127 findings, 0 Critical, 21 High
(15 distinct defects), 81 Medium, 25 Low. Nothing is fixed and nothing is
committed. Tracker:
[2026-11-21-focused-review-5.track.yaml](./2026-11-21-focused-review-5.track.yaml).
