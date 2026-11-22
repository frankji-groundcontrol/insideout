# 2026-11-22 — Roadmap: ordered-flow Canvas, node dependencies, band polish

Plan: [roadmap canvas graph and dependencies](../plans/2026-11-22-roadmap-canvas-graph-and-dependencies/2026-11-22-roadmap-canvas-graph-and-dependencies.md).
The work is on the branch `feat/roadmap-graph`, which is not merged yet. The
people-in-charge server work (S1, S2) is still running; this record grows
when it lands.

## What changed

| Area | Change | Commits |
| --- | --- | --- |
| Band polish (C1) | Now is the only emphasised band. Pressure shows as chips, and "normal" is hidden. Bands' Later moves finished work to a Done band and groups the rest by top-level branch. Bands cap at 1120 and their cards fill the row. The header scrolls away. Light muted text passes AA on lanes. One presence chip per user ([critique](../design-qa/2026-11-21-roadmap-critique.md)). | `6a0335fa`, `27d2e4c5`, `178c1020` |
| Node dependencies (C2) | A node can wait on other nodes of its project: the `roadmap_dependencies` table with RLS, cycle and in-tree refusals, and `POST`/`DELETE /roadmap/{nid}/dependencies`. Every node view carries `dependsOn`, and the agent context carries `waitsOn`. The surfaces are the Go client, CLI `roadmap depend/undepend` and MCP `roadmap_dependency_add/remove` ([architecture](../architecture/roadmap-tree.md)). | `8dcf3f17`, `8496c1e6`, `ef928470`, `ed8cd6af` |
| Canvas (C3) | One level of steps, ordered left to right by dependencies lifted to that level. Loops are grouped. Links that skip a column run through reserved lanes. Opening a step shows its own steps, so that is how the hierarchy is reached. Drag pans, the wheel or a pinch zooms, and there are +/−/fit buttons. A "Depends on…" picker sits in the node menu. The minimap and the fold toggles are gone ([design QA](../design-qa/2026-11-22-roadmap-canvas-graph.md)). | `f717d78c`, `721daf0b` and earlier C3 commits |
| Demo data | The Spark seed adds eight dependencies, with the closed beta waiting on four teams, and checks the self, cycle and in-tree refusals. A link that made Matching and Launch wait on each other was replaced. | `53d823dd`, `d32720db` |

## Verification so far

These gates ran on the branch after C1–C3 and before the routing fix:

- `go build`, `go vet`, `gofmt`: clean.
- `go test ./...`: passes without a database, and on a freshly migrated
  private PostgreSQL 18 database with 0 skipped tests.
- `flutter analyze`: 8 infos (baseline 8). `flutter test`: 694 passed, and
  704 after C3's later commits.
- Redaction: passes.
- Live check on the Spark data: the migration was applied to the dev
  database, eight links were added through the API, and the self, cycle and
  in-tree cases were refused. The Canvas shows Onboarding, Matching and Safety
  in parallel, then Chat, merging into Launch.

The full gates run once more after S1 and S2 are integrated.
