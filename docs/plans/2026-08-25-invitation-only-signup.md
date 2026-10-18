# 2026-08-25 — Invitation-only signup + first admin

Status: **Finished** — implemented, live-verified, deployed; see the
[changelog](../changelogs/2026-08-25-invitation-only-signup.md).
*Superseded 2026-10-11: the user made the destructive call this plan
parked with them — 859 junk accounts (840 `@test.local` fixtures + 19
dead old-era accounts, superseding the 582-row legacy count below) were
purged after a snapshot, with the keep-set enumerated
([changelog](../changelogs/2026-10-11-owner-deploy-and-mail-truth.md)).*

## Why

The user registered `<admin-email>` and asked for (a) this user
to be the only user and admin, and (b) an invitation-only registration
switch like `~/Projects/qwenimage`. Reality check: the users table holds
582 rows (legacy migration from the old juanleme/auth table) — deleting
them cascades their workspaces/projects, so this slice **promotes the
account to admin and locks new signups behind invitations** without
touching existing rows; the purge decision stays with the user.

## Design (qwenimage pattern, insideout idioms)

- Migration `20260825120000_invitation_signup.sql`:
  - `insideout.users.system_role` ∈ member/admin (default member).
  - `insideout.invitations` — backend-only (sessions/auth_tokens
    posture): invited email, unique sha256 `code_hash`, `expires_at`,
    `used_at`/`used_by`, `revoked_at`, `created_by`. Single-use claimed
    atomically by `UPDATE … WHERE used_at IS NULL AND revoked_at IS NULL
    AND expires_at > now()`.
  - `insideout.settings` (key → jsonb) seeded
    `signup_mode = "invitation_only"` — fail closed like qwenimage
    ("an unconfigured deployment must not drift open"). Modes:
    `invitation_only | open | closed`.
- Admin promotion: `insideout promote-admin <email>` server subcommand
  (owner-DSN store op) — the personal email stays out of the repo.
- Store: `GetSetting`/`SetSetting`, `CreateInvitation` (replace prior
  unused invite for the same address, one live invite per email),
  `ListInvitations`, `RevokeInvitation`, `ConsumeInvitation` (atomic,
  email-bound), `SetSystemRole`; gated tests first.
- API: public `GET /auth/config {signupMode}`; `POST /auth/register`
  gated (invitation_only → `inviteCode` required + consumed on success;
  closed → 403; open unchanged); admin routes under `requireAdmin`:
  `POST /admin/invitations` (code returned once + emailed),
  `GET /admin/invitations`, `POST /admin/invitations/{id}/revoke`,
  `GET/PUT /admin/settings` (signupMode switch). `userResponse` gains
  `systemRole` so clients can show the admin surface.
- Web: register form shows an invitation-code field when the config
  says so (AuthForm flag, fetched by the page); profile page gains an
  admin panel (mode switch, invite create/list/revoke, code shown
  once); l10n en/zh.
- CLI: `invite <email>`, `invites`, `signup-mode [mode]`, and `register`
  gains `--invite`; MCP: `invite_create`, `invite_list`, and
  `auth_register` gains `invite_code`.

## Out of scope

- Deleting the 582 legacy users (user's explicit call, destructive).
- Multi-use/bulk invitations, per-workspace scoping (system-level
  signup only, like the current single-tenant reality).
- Disabling existing accounts (`disabled_at`).

## Checklist

- [x] Plan + board row
- [x] Gated store tests → migration + store methods
- [x] Register gate + auth config + admin API + `promote-admin`
- [x] apiclient + CLI verbs + MCP tools
- [x] Client: register invite field, admin panel, l10n
- [x] go build/vet/test green; flutter analyze/test green
- [x] Live exercise (local): closed/invitation/open matrix, invite→
      register, wrong code, revoke
- [x] Deploy: server + app; promote <admin-email> in prod; verify prod gate
      (config invitation_only, no-code register 403 via the domain)
- [x] Changelog + backend.md + usage doc + HANDOFF/board closure
