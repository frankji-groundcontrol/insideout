# 2026-10-07 — Own-URL fresh-database bootstrap: setup does not check or provision the two roles

> **Resolved 2026-10-10.** `insideout setup --local --postgres own` probes the two roles before it
> migrates. Both present: it migrates as before. Both missing and the connecting
> role can plausibly create them: it offers to (a prompt at a terminal,
> `--create-roles` otherwise), generates the passwords into `.env` without printing
> them, and rewrites the DSNs. Anything else stops before any migration and prints
> the SQL. A second class, found while testing this, is also handled: roles that
> exist but a database `insideout_owner` can neither own nor create in
> (`65b0824`). Limits, and the verification on a roleless cluster and a real
> terminal, are in
> [setup roles](../changelogs/2026-10-10-close-open-gaps/setup-roles-and-bundled-owner.md).
>
> The original record follows, unedited.

## Problem

`insideout setup --local --postgres own` (and `setup.sh` before it)
writes the DSN and migrates, but a **fresh** database lacks the
two-role provisioning the bundled Docker path gets from
`docker/postgres-init/001-create-app-role.sh`. The migration then
fails with either `role "insideout_owner" does not exist` or — with
the roles created but the owner missing BYPASSRLS — `infinite
recursion detected in policy` (see the
[learning record](../learning/2026-10-07-own-url-role-provisioning-precondition.md)).
The error surfaces mid-migration with no hint that the fix is
provisioning, not the migrations.

## Proposal

Setup (both `insideout setup` and, later, the consolidated tool)
should, on the own-URL path:

1. Probe for the two roles (`pg_roles` lookup) before migrating.
2. If absent and the DSN user has sufficient privileges, offer to
   create them exactly per the init script (NOSUPERUSER, owner
   BYPASSRLS + database ownership, app CONNECT-only) — with the
   passwords generated, stored in `.env`
   (`POSTGRES_OWNER_PASSWORD`/`POSTGRES_APP_PASSWORD`), and never
   printed.
3. If privileges are insufficient, fail before migrating with the
   exact provisioning SQL the operator needs, mirroring the init
   script.

## Constraints

- Never print generated passwords or the DSN.
- Migrations must keep running as the provisioned model dictates —
  this does not change the two-role design
  ([database-and-rls.md](../architecture/database-and-rls.md)).

Found live during the setup-TUI slice-2 exercise
([changelog](../changelogs/2026-10-07-setup-tui-slice-2.md)).
