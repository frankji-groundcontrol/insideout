# 2026-11-21 — Spark demo data script

Added `scripts/demo/seed_dating_scenario.py` and its story table
`scripts/demo/spark_story.py`: a stdlib-only seed that builds the "Spark"
dating-app team story (five people, ideas, an approved and committed PRD, a
33-node roadmap with delegations, evidence, validation, proposals, a
hand-back, project updates and a root-scoped agent token) through the HTTP
API only. How to run it and the logins: [demo-data.md](../usage/demo-data.md).
`scripts/tests/test_demo_story.py` checks the story table without a server.

## Verification

- Run against a running local server on a fresh database: every request
  answered the expected status, and a second run refused because the
  workspace already exists.
- Run again on a second fresh database with the same result.
- `python3 -I scripts/tests/test_demo_story.py`: 6 tests pass.

## Findings while running it

- The build's first-leaf seed (`seedFirstLeaf`,
  `server/internal/store/roadmap_plan.go`) does not check that the node is a
  leaf, and orders by `created_at, position`, which tie inside the build's
  one transaction. On one run it set the branch "Report a member" in
  progress, with a seven-day deadline and the builder as owner, inside a
  branch later delegated to someone else. The
  [roadmap tree](../architecture/roadmap-tree.md) says a build seeds its first
  in-progress *leaf*. The script now reports this as a finding when it
  happens; product code is unchanged.

## Operator notes

None: the script is for disposable local databases only.
