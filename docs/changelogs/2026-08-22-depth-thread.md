# 2026-08-22 — Depth thread: revision verbs, live presence, time-first roadmap

The three remaining threads from the week's arc, all closed in one
pass. Plan: [depth-thread](../plans/2026-08-22-depth-thread.md).

## PRD-revision verbs (parity Stage 3 closed)

- CLI `insideout revisions <prd-id>` and `insideout snapshot [--note N]
  <prd-id>`; MCP tools `revisions` and `snapshot` (27 tools total).
- Live: snapshot recorded, revisions listed (1), through the domain.

## Real-time canvas presence

- `internal/presence`: in-memory per-project session registry with TTL
  pruning, deterministic snapshots, and change broadcast (unit-tested:
  touch/prune/leave/subscribe, ordering).
- `GET /projects/{id}/presence/stream` (SSE; connecting registers a
  per-tab client id under the authed user, heartbeats every 10s double
  as TTL refresh, disconnect leaves) and `GET /projects/{id}/presence`
  (snapshot for CLI `presence` / MCP `presence`). Membership enforced —
  a non-member's stream 403s (observed live).
- Web roadmap page: presence chips (who is here) fed by the SSE stream.
- Live: two tabs saw each other join and leave in their streams; the
  CLI snapshot after both left was `[]`.

## Time-first roadmap (PRODUCT.md "Time is a first-class constraint")

- Migration `20260822120000`: `roadmap_nodes.deadline` (nullable).
- `roadmaptime` package: pressure states (normal → near → high risk →
  overdue) and the Progress assembly — Now is deadlined work only,
  Next at most three by earliest deadline, Done counted, and
  in-progress work missing a deadline surfaced as `needsDeadline`
  ("an item without a deadline cannot enter Now").
- API: node views carry `deadline` + computed `pressure`; PATCH accepts
  `deadline` (RFC3339 sets, `""` clears); `GET
  /projects/{id}/roadmap/progress` returns the Progress view.
- Web: deadline chips colored by pressure on node tiles, a date picker
  + clear in the edit dialog, and a Now / Next / Done progress strip on
  the roadmap page. CLI: `roadmap update --deadline RFC3339|clear`,
  `roadmap progress`; MCP: `roadmap_update` deadline option + `progress`
  tool.
- Live: setting a 1-day deadline flipped the node to `high_risk` and
  into Progress.Now; clearing it moved the item to `needsDeadline`.
  Two real bugs fixed on the way: `ListRoadmap`'s row scan missed the
  new column (14-vs-13, caught live as a 500), and the CLI's `--deadline
  clear` dropped the field (empty string ≠ absent).

## Housekeeping

Full suites green (server + client 45/45), vet, gofmt; both services
deployed; scratch users purged.
