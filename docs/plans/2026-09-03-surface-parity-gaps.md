# 2026-09-03 — Surface parity gaps (API ↔ CLI ↔ MCP)

Status: **complete** (2026-09-03). A three-surface audit (API 76 routes /
CLI 42 product verbs / MCP 46 tools) found six non-deliberate alignment
gaps plus hygiene items; the live matrix then caught a seventh,
pre-existing bug (BUG-015). Deliberate omissions (coach
conversations/SSE, webhooks, cursor, human-only deletes) stayed out of
scope. Record:
[changelog](../changelogs/2026-09-03-surface-parity-gaps.md).

Method: TDD per fix — failing apiclient/mcpserver/CLI test first
(httptest, no mocks of the API contract), then the smallest passing
change. Client-side only: no store/migration changes.

## Fixes (in order)

- [x] 1. Invitation revoke: wired into CLI (`invite revoke <id>`,
      cli_admin.go) and MCP (`invite_revoke`, new mcpserver
      `admin_tools.go`). Live: create → revoke → ledger `revokedAt` →
      revoked code 403s at register.
- [x] 2. signup-mode on MCP: `signup_mode` (no arg = public read, arg =
      admin write). Live read over real stdio.
- [x] 3. PRD export on MCP: `prd_export` (format/audience optional);
      new apiclient `PrdExport`; CLI `view --export` URL pinned
      unchanged. Live: full markdown over stdio.
- [x] 4. Workspace join: apiclient `JoinWorkspace(code)`, CLI
      `join <code>`, MCP `workspace_join`. Live on both surfaces
      (second user joined two workspaces by code).
- [x] 5. `roadmap_move` position on MCP: hardcode was at the MCP call
      site, not apiclient; tool now takes optional `position`. Live:
      NodeB reordered to position 0.
- [x] 6. `build` expectedCount on MCP: tool now takes optional
      `expected_count`. Live: 409 `replace_conflict` without it,
      success with `expected_count=10`.
- [x] 7. CLI usage text: `roadmap` usage lists `progress|presence`.
- [x] 8. (Found by the live matrix) apiclient `SetSignupMode` POST →
      PUT; CLI `signup-mode <mode>` had 405'd since 2026-08-25.
      [BUG-015](../issues/2026-09-03-bug-015-signup-mode-wrong-method.md).

## Hygiene / records

- [x] MCP tool count 46 → 50: `SKILL.md` (frontmatter, signup/join
      note, prd_export, parity paragraph) and
      `docs/usage/cli-mcp-and-guidance.md` (verb table + MCP section).
- [x] Backfilled stale checkboxes:
      [2026-08-21-cli-mcp-parity](2026-08-21-cli-mcp-parity.md) Stage 3 +
      header status, and
      [2026-08-21-roadmap-parity](2026-08-21-roadmap-parity-and-github.md)
      Stage C.
- [x] Changelog entry; board row; issues index updated.
- [x] Live matrix on a local server (scratch Homebrew Postgres cluster,
      roles provisioned via `server/db/provision_roles.sql`, migrated +
      seeded, torn down after) — all eight fixes exercised over real
      HTTP and real MCP stdio.
- [x] Modularity debt filed:
      [apiclient.go line budget](../issues/2026-09-03-apiclient-line-budget.md).

## Verification

`(cd server && go build ./... && go vet ./... && go test ./...)` — all
packages green; every new verb has a RED→GREEN-pinned test. Live matrix
evidence in the changelog.
