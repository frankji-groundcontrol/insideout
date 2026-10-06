# 2026-10-06 — Three personas exercised live across CLI, Web, and MCP

## What changed

No product code in this pass. The live simulation
([plan](../plans/2026-10-05-user-simulation-cli-mcp.md)) finished its
matrix: three personas, three surfaces, three cases each, on a local
sim server and a fresh local database. Reusable harness notes are in
[the practice](../practices/2026-10-06-simulate-a-user-across-surfaces.md).
Drivers stay under `scripts/sim/` (`mcp_session.py`, `web_driver.py`).

## What the run showed

Persona A (everyday builder) went from an empty idea to a committed
"Day zero" version with three roadmap nodes and deadlines. Persona B
(operator) had an agent propose a four-step structure, accepted it,
set deadlines, and rejected a later scope cut. Persona C (reader)
joined A's workspace as `member` and read the same progress, the same
empty evidence, and the same "Day zero" version on CLI, MCP, and web.
He could not see B's workspace.

The promise holds for the happy path: a reader can say what is now,
what is next, and which version is committed. It does not hold for
authority. A `member` can add roadmap nodes and file proposals
([BUG-020](../issues/2026-10-06-bug-020-workspace-member-writes-the-plan.md)).
Three earlier findings stay open:
[BUG-017](../issues/2026-10-06-bug-017-roadmap-move-collides-sibling-position.md)
(sibling positions collide),
[BUG-018](../issues/2026-10-06-bug-018-roadmap-pressure-label-untranslated.md)
(web prints the raw pressure key),
[BUG-019](../issues/2026-10-06-bug-019-applied-proposal-reorders-items.md)
(applied proposal items land out of order).

## Parity

| Fact | CLI | MCP | Web |
| --- | --- | --- | --- |
| A's now / next / done | Cut the boring bits (2026-10-20); song, text; done 0 | same | same, under 现在 / 接下来 |
| A's evidence on the current node | `[]` | `[]` | project has no repo bound; evidence page is the GitHub card, empty |
| A's committed version | Day zero, revision 2, +8 sections, by Mira Voss | same | same card, "对该读者已就绪" |
| Management audience | ready, five sections, no gaps | same | 管理 chip selects; card text unchanged (commit was for 决策) |
| B's four nodes and dates | Intake 10-15, Handoff 10-22, Verification 10-29, Archive 11-05 | same | same |
| B hidden from a non-member | 404 workspace / project not found | same 404 | blank page, no B content |
| Reader cannot write | **no** — add and propose both succeed | same gate | project page still shows 接受 / 驳回 on an already-rejected proposal |
| Pressure label | enum `normal` | enum `normal` | raw key `roadmap.pressure.normal` |
| Applied proposal order | Intake, Handoff, Verification, Archive | same | same |

## Verification

Every row above was read back from the live API through the CLI and
one MCP process, and the web rows from screenshots of the release
build. The stray node a reader created was deleted by the owner, and
the stray proposal was rejected with a reason; a following list showed
the original three nodes.

## Operator note

Nothing to deploy. The sim server, web build, and browser were local
only. Tokens and the database connection stay outside the repo.
