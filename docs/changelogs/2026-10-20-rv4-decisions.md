# Review-4 decision clusters

Date: 2026-10-20. Plan:
[2026-10-20-rv4-decisions](../plans/2026-10-20-rv4-decisions/2026-10-20-rv4-decisions.md).
Not pushed, not tagged, not deployed.

## Landed

Cherry-picked onto local `main` from `d9d7641`, each accepted cluster's SHAs in order.

- Retention and erasure. `77eeed7` then `9ec7b31`, now `d5a45fe` and `3ecef69`.
  Idle coach conversations are purged 365 days after the newest message, and
  `DELETE /api/v1/me` tombstones the account. Detail:
  [retention changelog](2026-10-20-retention-and-erasure.md).
- Session-tool withhold. `59a57c3` then `e235c15`, now `8f3a76d` and `9e6e68e`.
  While a login is held, the MCP server does not advertise
  `auth_login_code_verify`, `auth_magic_link_consume`, or `auth_logout`.

## Stopped on conflict

Completion. Red `65569dc` landed as `2986f4b`
(`TestRoadmapCompletion_RefusesDoneWithoutEvidenceOrReason` only). Fix
`1cb3d1d` conflicted and the cherry-pick was aborted, so the cluster stopped.
The conflict was in:

- `docs/HANDOFF.md`
- `docs/changelogs/README.md`
- `docs/changelogs/index/2026-10.md`
- `docs/learning/README.md`
- `docs/plans/README.md`

The red test is on local `main` without the store rule, the migration, or the
web prompt. It fails when `DATABASE_URL` is set.

## Skipped

Rejected clusters were not cherry-picked.

- Omitted owner and last admin. `c2ee18e`, `221fb87`. The diff did not stay
  inside the handover store files: it also edited
  `server/internal/api/projects.go`, three API tests, changelog indexes, and
  this plan. `memberships.go` was not changed.
- Decision record for the keep-as-built rows. `17f15e7`. The new decision file
  does not state only the settings in the plan. It adds decisions the plan
  does not adopt, says the last-admin half was not decided, and never states
  the plan's retention, completion, session-withhold, or omitted-ownerId
  settings.

## Verification

`go test ./internal/mcpserver/ -count=1` passed. Those tests do not need a
database. The database gate was not re-run here: `DATABASE_URL` was unset and
was not loaded from `.env`. The verifiers had already run the database tests
for the accepted clusters.
