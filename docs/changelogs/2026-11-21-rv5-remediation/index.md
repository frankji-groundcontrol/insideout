# 2026-11-21 — Review 5 remediation

**Status: complete on the branch** (2026-11-21); not merged to `main`. Plan:
[2026-11-21-rv5-remediation](../../plans/2026-11-21-rv5-remediation/2026-11-21-rv5-remediation.md).
Source: [focused review 5](../../review/2026-11-21-focused-review-5/README.md).
Branch: `rv5/remediation`, pushed after each wave; `main` is not pushed. At
the owner's request no CI is run for this work: verification is local.

One record per wave, each with the rows it closed, the commits, the gate
results and anything left open:

- [Wave 0](./wave-0.md) — green tip, the install and password-reset slice
  finished and committed, the task-tree concept written; all seven rows closed.
- [Wave 1](./wave-1.md) — the idea body, the build path, the coach transcript and
  the drafted-tree preview; Go toolchain go1.26.11; TREEB-10 partly closed.
- [Wave 2](./wave-2.md) — the task tree for roles: one write rule in SQL for
  the store and row-level security, subtree and node reads, delegation and
  hand-back with an assignment history and feed rows, the Mine view and My
  work, the server roll-up and a foldable canvas, an agent token confined to a
  root; the CLI and MCP reads, the web route and menu.
- [Wave 3](./wave-3.md) — the four surfaces in step: one error style, one patch rule,
  the PRD list and updates paging, the missing CLI, MCP and web verbs, and a
  parity test that fails on any request field a surface cannot reach.
- [Wave 4](./wave-4.md) — structure: a responsibility rule with a function guard, the
  MCP, CLI, setup, LLM and web splits, tests named for what they pin, and the wave 3
  follow-ups.
- [Wave 5](./wave-5.md) — the documentation index: Marksman at 0 diagnostics, the
  doc graph and fragment checks committed, routers that answer what is open.

## Final state

On `rv5/remediation`, with every wave integrated:

- `go build`, `go vet`, `gofmt`: clean.
- `go test ./... -count=1` without a database and on a freshly migrated private
  PostgreSQL 18 database: every package ok, no skipped tests.
- `flutter analyze`: 8 infos (baseline 8). `dart format`: no change.
  `flutter test`: 667 passed.
- Every `scripts/tests/check_*.py`, including the new `check_doc_graph.py`
  (every markdown file reachable from the four routers), and
  `scripts/check-redaction.sh`: exit 0.
- Marksman: 0 diagnostics on a clean export of the tree.

What stays open is listed in each wave record and in
[the open findings](../../issues/2026-11-21-review-5-open-findings.md). One
owner question remains there: whether a delegate's agent may file proposals in
the delegate's subtree.

The method changed partway through: from wave 4 the coordinator dispatched and
reviewed the clusters directly
([lesson](../../learning/2026-11-21-map-files-before-fanning-out-and-judge-between-steps.md)).
