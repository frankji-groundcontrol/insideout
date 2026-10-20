# Review-4 decision clusters

Date: 2026-10-20. Plan:
[2026-10-20-rv4-decisions](../plans/2026-10-20-rv4-decisions/2026-10-20-rv4-decisions.md).
Not pushed, not tagged, not deployed.

## Landed

The workflow accepted retention and session-withhold and cherry-picked them.
The completion fix conflicted in five doc indexes; that conflict was resolved
and the fix landed. The omitted-owner fix and the decision record were
rejected by the verifier for leaving their file lists; both were reviewed
and cherry-picked after that.

- Retention and erasure. `d5a45fe` and `3ecef69`. Idle coach conversations
  are purged 365 days after the newest message, and `DELETE /api/v1/me`
  tombstones the account.
  [Detail](2026-10-20-retention-and-erasure.md).
- Session-tool withhold. `8f3a76d` and `9e6e68e`. While a login is held, the
  MCP server does not advertise `auth_login_code_verify`,
  `auth_magic_link_consume`, or `auth_logout`.
- Completion. `0a87f91`, from worker fix `1cb3d1d`. `in_progress` to `done`
  needs linked evidence or a one-sentence reason. Migration
  `20261020150000`.
  [Detail](2026-10-20-completion-evidence-or-reason.md).
- Omitted owner. `e25bb80` and `269391f`, from `c2ee18e` and `221fb87`. A
  PATCH that leaves `ownerId` out keeps the current owner. `null` and `""`
  still clear it, and clearing still needs a workspace admin. The API
  change was required to tell an omitted key from `null`.
  [Detail](2026-10-20-omitted-project-owner.md). The last admin was already
  refused by `ErrLastAdmin` in `memberships.go`.
- Keep-as-built decisions. `5938aa5`, from `17f15e7`.
  [Record](../issues/2026-10-20-rv4-decisions-adopted.md).

## Still the owner's console

The hosted database role, `INSIDEOUT_TRUSTED_PROXY_HOPS=1`, GitHub
`deployment_status`, key rotation, and the Railway `VERSION` variable.
What makes `v1.0.0` was not decided. Viewer-zone timestamps were not decided.

## Verification

`go test ./internal/mcpserver/ -count=1` passed during integration. Those
tests do not need a database. The verifiers ran the retention, completion,
and hand-over tests on private Postgres. This checkout did not re-run them:
`DATABASE_URL` was unset and was not loaded from `.env`.
