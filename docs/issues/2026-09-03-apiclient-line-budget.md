# 2026-09-03 — apiclient.go over the ~350-line modular-file budget

> **Resolved 2026-10-10.** `server/internal/apiclient/apiclient.go` is 169 lines; the roadmap verbs are
> `roadmap.go` (184) and the PRD verbs `prd.go` (163). A pure move (`e456024`);
> see [backend splits](../changelogs/2026-10-10-close-open-gaps/backend-splits.md).
>
> The original record follows, unedited.

`server/internal/apiclient/apiclient.go` is ~478 lines and mixes three
responsibilities: transport (`do`, `get`, `requestJSON`, `getBytes`,
token handling), roadmap/GitHub verbs, and PRD/version/agent-vocabulary
verbs. It crossed the budget before the 2026-09-03 parity slice; that
slice added ~20 net lines (`PrdExport` + `getBytes`) rather than fixing
the structure, because splitting mid-slice would have buried the
behavioral diff.

## Target structure

- `apiclient.go` — `Client`, token, transport helpers only (~120 lines).
- `roadmap.go` — roadmap CRUD/move/progress, build/expand, repo/guide/
  sync/evidence, presence, cursor.
- `prd.go` — PRD read/update, revisions/snapshot, commit/versions/
  readiness/view/export, agent vocabulary (context/checkpoint/propose/
  decide).
- `crud.go` and `auth.go` already exist and stay as-is.

## Bounded fix prompt

Move methods (no signature changes) into the three files above; keep
every httptest pinning test green untouched — the tests are the proof
the move is behavior-preserving. Gate: `go build ./... && go vet ./...
&& go test ./internal/...`. Do at the next structural touch of
apiclient.
