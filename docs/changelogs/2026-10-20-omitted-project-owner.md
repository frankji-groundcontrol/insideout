# 2026-10-20 — An omitted project owner stays

`PATCH /api/v1/projects/{pid}` used to treat a missing `ownerId` the same as
`null` and `""`: all three cleared the owner. An owner who is not a workspace
admin was then told that leaving the key out was giving the project away.

## What changed

- A body that omits `ownerId` keeps the current owner. The project's owner
  may make that edit. An admin may too.
- `ownerId: null` and `ownerId: ""` still clear the owner, and clearing still
  needs a workspace admin. Handing the project to someone else is unchanged:
  a non-admin owner cannot give it away (`ErrHandoverNeedsAdmin`, HTTP 403).
- The store spells the clear as `ProjectUpdateFields.ClearOwner`. A nil
  `OwnerID` without that flag is the omission. Callers that used to clear by
  passing a nil owner now set `ClearOwner`.
- No migration. The `projects_update` policy is unchanged.

The last-admin rule was already on this branch: `RemoveMember` and
`UpdateMemberRole` refuse to remove or demote the only workspace admin
(red `b24e563c5b677f3d0dcb313abc195900b3560ea9`, fix
`f608aa9b51b615e3decfbd1820c4f2daa7eea59b`). Two admins can still demote one.
This change does not touch that code. The same private database re-ran those
tests and they passed.

## Verification

Private PostgreSQL 18 on `127.0.0.1:55470`, roles `insideout_owner`
(`BYPASSRLS`) and `insideout_app` (`NOBYPASSRLS`), migrated from this
worktree, torn down afterwards. The app role was `DATABASE_URL`. No DSN was
taken from the environment or from `.env`.

Red, before the contract change (`c2ee18e74162a84a9e8b9011666b50565ec262c6`):

- `TestRvHandoverAnOmittedOwnerIsKept` failed: a non-admin owner's edit that
  omitted the owner returned `ErrHandoverNeedsAdmin`.
- `TestRvHandoverAnOmittedOwnerIdKeepsTheOwner` failed: the same PATCH
  answered 403.

Green, after the change: those two tests passed, and so did the hand-over
refusals (another member, `null`, `""`), the admin clears, the role edits in
`TestProjectEditsFollowTheirRoles`, `TestAuthz_WorkspaceMembershipLifecycle`,
and `TestRvStoreDbLastAdmin_SelfDemoteIsRefused` /
`TestRvStoreDbLastAdmin_LeavingIsRefused`. `gofmt` and `go vet` on the
edited packages were clean.

Nothing was pushed, tagged, or deployed.
