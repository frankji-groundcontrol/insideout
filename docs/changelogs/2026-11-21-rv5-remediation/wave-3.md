# Wave 3 · Keep the four surfaces in step

Rows 3.2 to 3.7 of the
[remediation plan](../../review/2026-11-21-focused-review-5/40-remediation-plan.md).
Row 3.1, the parity test, comes last, after the wave 4 MCP and CLI splits; this
record gains it then. The wave ran in stacked stages. Server-contract went
first; the CLI and MCP (surfaces) and the web then ran in parallel on top of it.
Each worker wrote a failing test first, an adversarial verifier re-ran the
gates, and the coordinator cherry-picked the result and re-ran every gate on the
integrated tree.

## Rows

| Row | IDs | Disposition | Commits |
| --- | --- | --- | --- |
| 3.2 | STRS-06, PARC-07, PARC-09, PARA-09 | FIXED: one patch rule on the CLI and MCP (an absent or empty value leaves a field, the word `clear` clears it), shared request structs for the edits; `avatarUrl` and `coverUrl` are kept when absent and cleared by `null` or `""`, on the server and in every client | `4174628d`, `436ccf84`, `037d0ae2`, `6e34ce7b`, `639e854a`, `7575aea8`, `fe13a74a`, `64511bc1`, `297dec18` |
| 3.2 | STRS-02 | Moved to wave 4 (the MCP registration helper) | |
| 3.3 | FE-02, PARB-03, PARB-07 | FIXED: `GET /workspaces/{id}/prds`; `prdId` on every project view; `GET /projects/{pid}/updates?before=&limit=` with `nextCursor`; the same on the CLI, MCP and web (a PRD list on the workspace page, a project links to its PRD, "Load older updates") | `8fbee6a3`, `5cd05c87`, `3e9f5554`, `01082424`, `f7535a85`, `f612ca26`, `c70cef2b`, `bd028b73`, `0da25777`, `098f27c2` |
| 3.4 | PARB-04, PARB-06, PARC-04, PARA-03, FE-07, FE-04, PARB-08, PARB-11 | FIXED: project show, update and delete, workspace edit, export, evidence add, validate and account delete on the CLI; the project and workspace reads and edits and posting updates are open to agent tokens (an `ownerId` from an agent is still refused) and MCP has tools for them; account deletion, the project owner, and the `insideout.yaml` guide on the web | `a1c39cbc`, `b6de0fcc`, `7152f9e0`, `4279225a`, `c7aa9cb2`, `28c33dec`, `56d7c176`, `30625d15`, `2f6c3df3`, `5c6343ec` |
| 3.5 | PARA-06, PARC-11, PARB-09, PARA-11, PARB-12 | FIXED: workspace delete, member role and removal, update edit and delete, and profile edit on the CLI; README, SKILL.md and the usage page state the parity that exists; the two SKILL.md commands that exited 2 are fixed. PARA-06 leaves one gap: there is no single-idea read on the Go client or CLI | `20d22428`, `6d3ee1ed`, `96d71977`, `96ea5a53`, `94545cb2`, `f5d6139e` |
| 3.6 | PARC-08, PARC-10, PARA-12 | FIXED: one error style (snake_case `code` on every error, camelCase extra keys, a guard test over every error write); evidence of an unknown or invisible node answers 404 and uses the API's time layouts; an invitation `ttlHours` outside 1 to 720 answers 400 | `513f40c6`, `ab532066`, `27488b1f`, `48bb5894`, `ad38540b`, `2ac9aa4a` |
| 3.6 | STRS-11 | PARTIAL: one store-error helper used by the roadmap, project, PRD, idea and agent handlers (43 call sites); the remaining ladders are in [the store-error ladders record](../../issues/2026-11-21-store-error-ladders.md) | `d2056963` |
| 3.6 | PARB-10 | PARTIAL: proposal detail, decision reason and commit notes are bounded at 2000 characters; a checkpoint `nodeId` must be a node of that project. The other unbounded fields are a wave 4 follow-up | `c74509b6`, `16f29f38` |
| 3.7 | PARB-05, PARA-04, PARA-07, PARA-05, PARC-06, PARC-12 | FIXED: accepting a proposal applies its items unless `--no-apply`; MCP errors no longer name hidden tools; MCP registers no tool on a human-only route; the CLI, MCP and web report the server's mail word; the node edit dialog sends only changed fields; every roadmap, GitHub and proposal request on the web goes through an `ApiCall` builder | `8f1747c5`, `49b7bdd0`, `0812439c`, `35baf3ba`, `60de7565`, `14a25bad`, `5d8ae7d7`, `2eaed3e9`, `c1405256`, `6ed07679`, `496a84f5`, `c5996121`, `6c644379`, `4e14390a`, `4fda710f`, `60002f8c`, `dcab83a3` |
| — | FE-08, FE-10, FE-05 | FIXED on the web: the dock's Capture and Ideas taps focus the ideas wall's capture field; coach roles and evidence kinds are localized; workspace, member, project and proposal controls show only to those the server lets write | `cf947b4e`, `e8355c8d`, `d93d91e7`, `0a2a28d0`, `8770d2dd`, `34d62cf5` |

## Changes a caller sees

- **Error codes.** Every code is lowercase snake_case, for example
  `github_rate_limited`, `app_throttle` and `circuit_open`. Extra keys are
  camelCase, for example `retryAfterSeconds`. An error with no specific code
  carries its status name, for example `not_found`.
- **Clearing a field.** On the CLI the word that clears a field is `clear`, not
  `none`. `--acceptance ""` no longer clears anything. On MCP an empty value
  leaves the field unchanged.
- **MCP tools.** Seven tools whose routes refuse every token MCP can send are
  no longer registered. MCP gains `project`, `project_update`, `workspace`,
  `workspace_update`, `update_add` and `update_edit`.
- **Agent scope.** Agent tokens now reach 47 of 81 routes. `PATCH /roadmap/{nid}`
  with an `ownerId` that is not a member now answers 400 `owner_not_member`
  instead of 403, the same as the delegation route.
- **Updates feed cursor.** `nextCursor` is `null` exactly when no older update
  exists. Before, a page of exactly 50 also returned a cursor.

## Gates on the integrated tree

The tree includes waves 3 and 4 so far.

- `go build`, `go vet`, `gofmt`: clean.
- `go test ./... -count=1`, without a database and on a freshly migrated
  private PostgreSQL 18 database: every package ok and no skipped tests, apart
  from one guard. The function-length guard (wave 4, built in parallel)
  flagged two functions that wave 3 grew: `addAuthTools`, now 156 lines, and
  `runClientCommand`, 92. Both are listed at those lengths and assigned to
  the wave 4 splits (`23eca01d`).
- `flutter analyze`: 8 infos (baseline 8). `dart format`: no change.
  `flutter test`: 650 passed.
- Every `scripts/tests/check_*.py` and `scripts/check-redaction.sh`: exit 0.

## Left open

- Row 3.1, the parity test.
- Picked up by the wave 4 follow-up cluster:
  - `canDelete` and `canEdit` on the workspace view: the server lets only the
    creator delete a workspace, but the web shows Delete to every admin;
  - the PRD list refreshing with the page;
  - the mail word on the forgot-password page;
  - the rest of PARB-10;
  - stale lines in `PRIVACY.md` and the usage pages.
- Picked up by web-controllers: "Load older updates" pages vanish when the
  project page rebuilds.
- `project update` and `workspace update` read the row and then PATCH it (marked
  in the code as a shortcut), so an edit that lands between the two is lost.
- A cursor whose update is deleted between pages answers 404; a position
  cursor would survive the deletion.
