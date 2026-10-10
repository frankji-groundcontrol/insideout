# Backend file splits

Four Go files were over the ~350-line budget in CLAUDE.md. Each is now split
along a responsibility seam, with no behaviour change.

| File | Before | After | New file(s) |
| --- | --- | --- | --- |
| `server/internal/apiclient/apiclient.go` | 500 | 169 | `roadmap.go` (184), `prd.go` (163) |
| `server/internal/agent/anthropic.go` | 414 | 235 | `anthropic_parse.go` (186) |
| `server/internal/agent/anthropic_test.go` | 355 | 141 | `anthropic_parse_test.go` (221) |
| `server/internal/store/roadmap.go` | 403 | 309 | `roadmap_move.go` (103) |
| `server/internal/store/roadmap_test.go` | 561 | 300 | `roadmap_move_test.go` (116), `roadmap_attribution_test.go` (159) |
| `server/internal/agent/coach.go` | 467 | 349 | `coach_loop.go` (131) |

Seams: the apiclient file keeps the transport and auth verbs while roadmap and
PRD verbs move out; the Anthropic stream parser leaves the client; the roadmap
store's move/reorder operations leave the CRUD file; the coach's provider loop
leaves the turn handler.

## How it was verified

- **Pure move, proven line by line.** For each split, the non-blank lines of
  the old file at the base commit (ignoring `package`/`import` boilerplate)
  are compared with the union of the new files. Every pair came back
  `PURE MOVE`. The coach split is checked at its own commit (`8ef010a`)
  because the coach fixes ([backend-fixes.md](backend-fixes.md)) landed on top
  of it afterwards, each as a separate test-first commit.
- **Real PostgreSQL 18** on a disposable local cluster: `go build`, `go vet`,
  `gofmt -l` and `go test ./... -count=1 -v` over the whole module. 470 PASS,
  0 FAIL, 1 SKIP (`TestReadSecretTerminal`, which needs a terminal). The CI skip
  gate allows exactly that one.
- Two worker pre-check commits for the build-from-PRD handler were not merged
  (see [the issue record](../../issues/2026-07-25-build-from-prd-runs-llm-before-conflict-guard.md)).

## Operator notes

None. No runtime behaviour changed.
