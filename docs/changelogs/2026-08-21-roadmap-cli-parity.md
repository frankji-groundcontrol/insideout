# 2026-08-21 — Roadmap parity on the CLI (agent + branch CRUD + sync)

Plan: [roadmap-parity-and-github](../plans/2026-08-21-roadmap-parity-and-github.md).
The roadmap API was complete but reachable only from the web client;
this gives the CLI (and, by the frozen 1:1 rule, the future MCP
surface) the full roadmap vocabulary.

## What changed

- `apiclient` gained: `BuildFromPrd` (agent: PRD → project with a
  branched roadmap), `ExpandNode` (agent: grow a branch), roadmap
  `list/add/update/move/delete`, `SetRepo`, `SyncGithub`.
- CLI verbs, all over the same bearer-auth contract:
  `insideout build <prd-id>`, `insideout expand <node-id>`,
  `insideout roadmap list|add|update|move|delete`, `insideout repo set
  <project-id> <repo-url>`, `insideout sync <project-id>`. Flags
  precede positional ids (Go flag convention). Node statuses are
  `locked|pending|in_progress|done`.
- Fixed along the way: `RoadmapDelete` failed decoding a JSON body into
  a nil target (locked with a regression test); usage strings initially
  taught the wrong argument order and wrong status set.

## Verification

- `go build`, `go vet`, `gofmt`, and the apiclient tests (now covering
  delete-with-body) all green.
- Live against the hosted API through a real chain: register scratch
  user → workspace → idea → convert to PRD → **`build` created a
  16-node branched roadmap** → branch add (root + child, `--parent`
  respected) → update (rename + `in_progress`) → move to root →
  delete → **`expand`** grew a planner node with new children →
  `repo set` bound this repository → **`sync` pulled 20 evidence
  updates** from it.

## Still open (Stage B of the plan)

GitHub progress is pull-based with a server-level token. The plan
designs the full loop: user-registered GitHub App for per-user access,
`POST /api/v1/hooks/github` webhook with HMAC-SHA256 verification, and
a repo-side matching guide (`insideout.yaml`) mapping branches, labels,
and paths to roadmap nodes — events advance leaf-level evidence only,
per PRODUCT.md.
