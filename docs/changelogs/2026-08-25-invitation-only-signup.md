# 2026-08-25 — Invitation-only signup + first admin

The user registered `frank.ji@aya.yale.edu`, asked for it to be the only
user and admin, and requested an invitation-only registration switch like
`~/Projects/qwenimage`. Reality found on inspection: the users table
holds 582 rows (legacy migration from the old juanleme/auth table) —
deleting them cascades their workspaces/projects, so this slice promotes
the account to the sole admin and locks **new** signups behind
invitations; the 582-row purge decision stays with the user.

## What changed (qwenimage pattern, insideout idioms)

- Migration `20260825120000_invitation_signup.sql`: `users.system_role`
  ∈ member/admin; `insideout.invitations` (backend-only: email-bound,
  unique sha256 `code_hash`, expiring, single-use, revocable, one live
  invite per address); `insideout.settings` with `signup_mode` seeded
  **fail-closed** to `invitation_only` (`open`/`closed` the alternatives).
- Register gate (`auth.go`): closed → 403; invitation_only → `inviteCode`
  required, claimed atomically **before** the account is created (wrong
  code provisions nobody; an email conflict can burn an invite — the
  admin mints another). Public `GET /auth/config {signupMode}` drives the
  client's invite field.
- Admin API (`admin.go`, `requireAdmin` = auth + `system_role='admin'`;
  no self-promotion route exists): `POST /admin/invitations` (code shown
  once + emailed to the invitee), `GET /admin/invitations`,
  `POST /admin/invitations/{id}/revoke`, `GET/PUT /admin/settings`.
- Admin bootstrap: `insideout promote-admin <email>` server subcommand
  (owner pool — cross-user UPDATEs are RLS-blocked for the app role).
  Applied to the operator's account; the email stays out of the repo.
- `userView` gains `systemRole` (admin-only surfaced in responses).
- Projections (all four surfaces agree): CLI `invite [--ttl h] <email>`,
  `invites`, `signup-mode [mode]`, `register --invite <code>`; MCP
  `invite_create` / `invite_list`, `auth_register` + `invite_code`;
  apiclient auth-config/invitation/signup-mode verbs.
- Web: the register form fetches `/auth/config` and shows a 邀请码 field
  when invitation-only; the profile page gains an admin section
  (mode switch chips, invite create with one-time code dialog, recent
  invitations with revoke) shown only when `user.systemRole == 'admin'`.

## Verification

- TDD: gated store tests for the settings round trip (with save/restore
  cleanup — the setting is shared global state on the shared database),
  invitation lifecycle (single-use, email-bound, expiry, revoke,
  replacement), and `SetSystemRole` (owner pool). `go vet` + non-DB
  suites green; `flutter analyze` (22 pre-existing, none new) and
  `flutter test` 50/50.
- Live matrix against a local server on the real database: config
  `invitation_only`; no-code 403, wrong-code 403, valid-code 201,
  replay 403, revoked-code 403, non-admin on admin routes 403, no-token
  401; open-mode register 201; closed-mode register 403; mode switch and
  restore through the admin API; CLI invite/invites/signup-mode; MCP
  `invite_list` over real stdio. Live exercise caught a real bug — the
  pre-registration claim wrote `uuid.Nil` into the `used_by` FK (500);
  fixed with `NULLIF` in `ConsumeInvitation` and re-verified.
- Probe users/invitations deleted after (cascade verified); final state:
  0 probe rows, `signup_mode="invitation_only"`, the operator's account
  the only admin.

## Not done here

- The 582 legacy users remain — the user's explicit call (destructive).

  *Superseded 2026-10-11: the legacy purge is done — 859 junk accounts deleted
  after a schema-scoped snapshot, five real accounts kept by enumerated keep-set
  ([owner-deploy record](2026-10-11-owner-deploy-and-mail-truth.md)).*
- Email verification on open-mode signup (still out of scope, as before).
- Invite prefill from the `?invite=` signup URL deep link in the register
  form (the server emails the link; the field is paste-into for now).

Plan: [2026-08-25 invitation-only signup](../plans/2026-08-25-invitation-only-signup.md).
