# 2026-09-03 — Surface parity gaps (API ↔ CLI ↔ MCP)

A three-surface audit (API 76 routes / CLI 42 product verbs / MCP 46
tools) found six non-deliberate alignment gaps; the live matrix then
caught a seventh, pre-existing bug. All fixed TDD-first; MCP tool count
46 → **50**.

## What changed

- **Invite revoke wired**: `insideout invite revoke <id>` and MCP
  `invite_revoke` now reach `POST /admin/invitations/{id}/revoke`;
  apiclient `RevokeInvitation` is dead code no more.
- **`signup_mode` MCP tool**: no arg reads the public `GET /auth/config`;
  with `invitation_only|open|closed` writes `PUT /admin/settings`
  (admin). Mirrors CLI `signup-mode`.
- **`prd_export` MCP tool**: `GET /prds/{id}/export?format=markdown`
  (optional `audience`, `format=print`) returned inline — MCP has no
  filesystem. New apiclient `PrdExport(prd, format, audience)`;
  `PrdExportAudience` keeps its exact URL (CLI `view --export`
  unchanged), sharing a `getBytes` helper.
- **Workspace join**: new apiclient `JoinWorkspace(code)` →
  `POST /workspaces/join`; CLI `join <code>`; MCP `workspace_join`.
- **`roadmap_move` position on MCP**: the hardcoded `nil` was at the MCP
  call site, not in apiclient (`RoadmapMove` always accepted it); the
  tool now takes an optional `position`.
- **`build` expected_count on MCP**: same shape — `BuildFromPrd` always
  accepted it; the tool now takes optional `expected_count`, so agents
  can confirm a `replace_conflict` 409 retry instead of stalling.
- **CLI usage text**: `roadmap` usage now lists the implemented
  `progress` and `presence` subcommands.
- **Bug caught by the live matrix**: apiclient `SetSignupMode` sent
  `POST /admin/settings`; the route is `PUT` (`api/admin.go:23`) — CLI
  `signup-mode <mode>` had been 405ing against the real API since the
  invitation slice. Fixed to PUT, pinned by
  `TestSetSignupModeUsesPut`.

## Verification

- `(cd server && go build ./... && go vet ./... && go test ./...)` —
  all packages green; each new verb pinned by httptest/stdio tests that
  failed for the right reason first (RED→GREEN per fix).
- Live matrix on a local server (scratch Homebrew Postgres 14+ cluster,
  provisioned via `server/db/provision_roles.sql`, migrated + seeded,
  torn down after): invite create → `invite revoke` → ledger shows
  `revokedAt` → revoked code rejected at register with 403;
  `signup-mode open` round-trip; second user registered, `join <code>`
  into the seeded workspace; MCP over real stdio — `signup_mode` read,
  `prd_export` full markdown, `roadmap_move` position 0 reorder,
  `workspace_join` (memberCount 2), `build` 409 `replace_conflict`
  without expected_count then success with `expected_count=10`.

## Records

- New MCP tools live in `server/internal/mcpserver/admin_tools.go`
  (invite_revoke, signup_mode) and `product_tools.go` (workspace_join,
  prd_export) — registered via `addAdminTools` in `server.go`.
- Pre-existing modularity note: `internal/apiclient/apiclient.go` is
  ~478 lines, over the ~350 guideline; filed as a dated issue.
- Stale checkboxes on
  [2026-08-21-cli-mcp-parity](../plans/2026-08-21-cli-mcp-parity.md)
  Stage 3 backfilled — those verbs shipped in later slices; the board
  had been misreadable in audits.

Plan: [2026-09-03 surface parity gaps](../plans/2026-09-03-surface-parity-gaps.md).
