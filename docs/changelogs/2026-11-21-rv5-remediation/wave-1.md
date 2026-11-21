# Wave 1 · Stop losing and misplacing data

Rows 1.1 to 1.8 of the
[remediation plan](../../review/2026-11-21-focused-review-5/40-remediation-plan.md),
as three file-disjoint clusters. Each worker ran in its own git worktree with
a failing test first, an adversarial verifier re-ran every gate and probed the
fix, and the coordinator cherry-picked the accepted branches onto
`rv5/remediation` and re-ran every gate on the integrated tree. The owner's
decisions applied here: no Build from an unapproved PRD; no completion
override reason from an agent token.

## Rows

| Row | IDs | Disposition | Commits |
| --- | --- | --- | --- |
| 1.1 | PARA-02 | FIXED: content is optional in `PATCH /ideas/{id}` (absent or null keeps the body, `""` clears it); the Go client, the CLI (`idea update` without `--content`) and MCP `idea_update` send it only when given | `c3d6841`, `a695b8f`, `85e6da2` |
| 1.2 | FEAT-01, PARB-02 | FIXED: the Build draft is read-only; the confirm creates and links the project, for the PRD author or a workspace admin only, and fails when the link touches no row | `32eb057`, `32379cf` |
| 1.3 | PARB-01 | FIXED: draft and confirm answer 409 `prd_not_approved` for a PRD that is not approved, on every surface | `32379cf` |
| 1.4 | FEAT-02 | FIXED: an agent token builds only onto an empty roadmap (403 `agent_scope_denied` otherwise, decided under the per-project lock) and is refused a `completionReason` | `9e9075e`, `1079abb`, `85e6da2` |
| 1.5 | FEAT-03 | FIXED: a replace over a roadmap with evidence, done or validated nodes answers 409 `roadmap_replace_has_history` with the counts unless the request says `force`; CLI `build-confirm --force` and MCP `force`; the web states what will be lost and asks before forcing | `6301d18`, `2c7b89d`, `35beaab`, `9be3d8a` |
| 1.5 | TREEB-10 | PARTIAL: the replace guard only; deleting a subtree with history is still unguarded and a forced discard writes no project update (moved to wave 2, row 2.6) | `2c7b89d` |
| 1.6 | PARC-05 | FIXED: a move with no `parentId` keeps the parent; `null` makes a root; CLI `--root`, MCP `parent_id: "root"` | `d3cde66`, `d59ff5b`, `35d3f49` |
| 1.7 | FE-01 | FIXED: the coach transcript shows the sent message at once, keeps every turn, re-reads the conversation when a turn ends, and drops a send that never arrived | `3826d01`, `846237c`, `7941126`, `d507192`, `458ff76` |
| 1.8 | FEAT-09, FE-09 | FIXED: Build shows the drafted tree read-only with Apply and Cancel before the confirm call | `35beaab`, `9be3d8a` |

Also in this wave's integration:

- `7bcb71e`: the Go toolchain moves to go1.26.11, which closes
  [the govulncheck record](../../issues/2026-11-21-govulncheck-go-toolchain.md)
  (`govulncheck ./...`: no vulnerabilities in called code).
- `8a7e8e5`: `client/lib/l10n/en_workspace.dart` formatted, which turns CI's
  `dart format` step green (it was red on `main`).

## Gates on the integrated tree

- `go build`, `go vet`, `gofmt`: clean.
- `go test ./... -count=1` without a database and on a freshly migrated private
  PostgreSQL 18 database: every package ok, no skipped tests.
- `flutter analyze`: 8 infos (baseline 8); `dart format --set-exit-if-changed`:
  no change; `flutter test`: 591 passed.
- Every `scripts/tests/check_*.py` and `scripts/check-redaction.sh`: exit 0.
- `govulncheck ./...` with go1.26.11: no vulnerabilities in called code.

## Left open

- TREEB-10's other half: a delete of a subtree with history and a record of a
  forced discard (wave 2).
- A position-only move reads the current parent outside the move's lock; a
  concurrent re-parent between the read and the move is undone (marked in the
  code as a shortcut).
- Two concurrent first confirms by the author can still create two projects
  (pre-existing; the link check does not catch it because both updates touch a
  row).
- A first build that fails after creating the project leaves a linked project
  with an empty roadmap; a retry fills it.
- No test walks the exact order the real server answers the web's replace flow
  in (conflict, then history, then forced); each step is tested alone.
- The coach transcript does not scroll itself to the newest message (as before).
