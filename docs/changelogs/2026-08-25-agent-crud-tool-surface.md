# 2026-08-25 — Agent CRUD tool surface

Audit of the 39-tool surface against the API found the external-coding-
agent loop broken in four places: the idea inbox was write-only, PRD
sections could not be edited, GitHub tracking could bind+sync but not
**read the evidence collected**, and an agent could not bootstrap a
workspace/project. This slice completes the matrix — **+7 tools → 46**
(37 product + 9 auth), 1:1 across API ↔ CLI ↔ MCP — plus a repo-root
[SKILL.md](../../SKILL.md) that teaches any coding agent the workflow
and the agent vocabulary (checkpoint / propose / humans decide / Commits
are human-only).

## What changed

- apiclient (`crud.go`, httptest first): `Ideas`, `UpdateIdea`,
  `DeleteIdea`, `UpdatePrd` (title + sections), `SetPrdStatus`,
  `NodeEvidence`, `CreateWorkspace`, `CreateProject`.
- CLI (`cli_crud.go` + dispatch): `ideas <ws>`, `idea update|delete`,
  `prd update [--title] [--section k=v]… [--status] <id>`,
  `evidence <node>`, `workspaces create`, `projects create` — flags
  precede positionals (repo convention; the first live pass caught my
  usage strings violating it).
- MCP (`product_tools.go`): `idea_list`, `idea_update`, `idea_delete`,
  `prd_update` (sections as `sections_json`, optional status), 
  `roadmap_evidence`, `workspace_create`, `project_create`.
- SKILL.md: setup, the working loop (bootstrap → triage → author →
  roadmap → track/evidence → vocabulary), guardrails (no agent-side
  deletes of PRD/workspace/project; the web coach's conversations stay
  web-only).

## Design decisions recorded

- `idea_delete` is the API's **soft drop** (row stays with status
  `dropped`; lists return every status) — documented in all three
  surfaces and SKILL.md so agents filter client-side instead of
  assuming disappearance.
- Deliberately excluded: PRD/workspace/project DELETE (human, web-side),
  conversations/messages (web coach's channel; SSE doesn't fit stdio),
  member management, updates CRUD (`checkpoint` covers agent reporting).

## Verification

- apiclient verbs pinned by httptest (method/path/body incl. the
  title-carrying PRD patch); `go vet` + `cmd`/`apiclient`/`mcpserver`
  suites green.
- Live matrix on a local server against the real database (throwaway
  workspace/project/user, deleted after): workspace+project bootstrap,
  idea create → list → update → delete (soft-drop semantics confirmed
  against the real API), `idea convert` → `prd update` sections
  round-trip, `evidence` read on a fresh node, and MCP `idea_list` over
  real stdio (2 ideas, statuses intact).
- The count in the usage doc was already stale (said 27→38; reality was
  28→39 before this slice) — corrected to 46.

Plan: [2026-08-25 agent CRUD tool surface](../plans/2026-08-25-agent-crud-tool-surface.md).
