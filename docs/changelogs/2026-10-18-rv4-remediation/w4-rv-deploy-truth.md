# Wave 4, deploy truth — review-4 remediation

Branch `rv4/w4-rv-deploy-truth`, on top of `rv4/w4-env-truth` (`be40fccb`).
Cluster `w4-rv-deploy-truth`. Findings are cited as `rv4-<ID>`. No value, name
or address is spelled in this fragment. Shared indexes are not updated here.

## What changed

- **The shipped compose stack and the hosted image no longer share one
  X-Forwarded-For line** (rv4-FR-F61, F106, F51, F57, F99, F107, D02; red
  `a74fec2e`, `a029fd61`, `e5d00743`, fix `f2d4e8b4`). `client/nginx.conf`
  takes both `X-Forwarded-For` lines from `${NGINX_XFF}`. The client image
  bakes the hosted pass-through (`$http_x_forwarded_for`), so the edge's
  append stays the only hop and hosted stays at
  `INSIDEOUT_TRUSTED_PROXY_HOPS=1`. Compose sets
  `$proxy_add_x_forwarded_for` on the client service and defaults the server
  to 1, so the rightmost entry is the peer nginx saw and a client-written
  value stays to its left. The server's own unset default is still 0.
  `docs/usage/deployment.md`, `docs/usage/environment.md` and `.env.example`
  say to add one to the hop count for every extra appending proxy in front of
  the compose nginx, and to bind the published client port to loopback or
  firewall it when the count is 1 or more unless nothing else can reach it.
  The sentence that called the default 0 right for compose is gone. The
  shared-bucket paragraph now names the silent resend cooldown (one user's
  mail request, the next person's 200 with no email and no log line) along
  with the login lockout and the refresh cap. `clientIP`'s comment and
  `docs/architecture/backend.md` name `NGINX_XFF`. The three nginx guards
  already render the template; the header and webhook guards needed no
  further edit because they render through the same substitution the
  re-resolve guard learned in its red commit. Real nginx: the hosted
  rendering passes a client-written value through, and the compose rendering
  ends the chain in the peer it saw.

- **The deploy note no longer claims a single instance has no overlap**
  (rv4-FR-F192, F39, F102, F111; red `e3fe7920`, fix `a01095cc`). The new
  image migrates at boot, before it listens and before the singleton guard,
  while a rolling deploy still has the old process serving. The previous
  image selects `password_hash`, so login, register, `GET /me` and profile
  reads fail with SQLSTATE 42501 for that overlap (reproduced against a
  throwaway database: the old `SELECT` of that column as the app role is
  42501; a select of the granted columns is not). For this release the doc
  says to stop the old instance before the new one boots, and on Railway to
  set the overlap seconds to 0. A compose recreate has no overlap. The
  owner `GRANT` that puts the previous image back is unchanged.

- **An empty `DATABASE_OWNER_URL` is no longer described as an owner
  `DATABASE_URL`** (rv4-FR-F75; red `6d9935ed`, fix `18554ad3`). `serve`
  calls `CheckServeRole` before `Migrate` and refuses a pool role with
  `BYPASSRLS` or `SUPERUSER`. The environment row now says empty is valid
  only for `migrate` and `seed` run with an owner DSN, that `serve` requires
  the non-bypass app role, and that the owner URL is required whenever
  migrations are pending. A throwaway `serve` with the owner role in
  `DATABASE_URL` and no owner URL exited 1 with `refusing to serve` before
  it listened.

## Not a defect at this base

- **rv4-FR-F62.** `docs/usage/deployment.md` already named
  `INSIDEOUT_TRUSTED_PROXY_HOPS` (the hop rule, formerly around lines 118 and
  133), and so did `docs/usage/environment.md` and
  `docs/architecture/backend.md`. The red guard's "does not name the
  variable" check did not fire. The stale "default 0 is right" sentence was
  the compose-stack claim fixed under F61 and D02, not a missing variable.
  The resume paragraph of `docs/HANDOFF.md` already says to set the hosted
  value to 1. What is still missing is outside this cluster (below).

## Deferred owner question

Splitting `20261018200000_users_read_grants.sql` across two releases is a
release-process decision and is not made here.

Question: ship the password_hash-free user reads and the
`_credential_for_email` call in one release and the `REVOKE` in the next, so
a rolling deploy's old image keeps working, or keep the single migration and
require zero overlap for this release?

Options: (1) expand/contract across two releases; (2) keep the single
migration and the zero-overlap procedure the deploy doc now states (stop the
old instance before the new one boots; on Railway set the overlap seconds to
0). Option 2 is what an operator can do today. Option 1 is not started.

## Integrator

- INTEGRATOR TODO: docs/HANDOFF.md: Shipping bullet next to `railway up --service server` should repeat that the hosted server stays at INSIDEOUT_TRUSTED_PROXY_HOPS=1 (the edge appends, nginx passes the chain through); the resume paragraph already says so.
- INTEGRATOR TODO: docs/changelogs/2026-10-18-rv4-remediation/wave-2.md: qualify the rollback sentence "rollback is redeploying the previous image" so it is not read as a general rollback once 20261018200000 has run; it is correct only for the pre-migration role-check exit, and the previous image's user reads then fail with SQLSTATE 42501 until the owner runs the documented GRANT.
- INTEGRATOR TODO: README.md: next to `docker compose up -d`, say the bundled nginx appends the peer it saw and compose defaults INSIDEOUT_TRUSTED_PROXY_HOPS to 1, add one per extra appending proxy, and bind or firewall CLIENT_PORT when hops is 1 or more unless nothing else can reach it.
- INTEGRATOR TODO: docs/usage/installation.md: `docker compose up -d postgres` is the postgres-only path (no nginx, hops 0 is right); the file never mentions the variable, so the same sentence belongs beside any full `docker compose up -d`.
- INTEGRATOR TODO: SKILL.md: the root skill never mentions the variable or `docker compose up -d`; add that sentence wherever an operator is pointed at the full compose stack.
- INTEGRATOR TODO: skills/insideout-setup/references/modules.md: beside `docker compose up -d postgres` and the full-compose mention, distinguish the postgres-only path (hops 0) from the full stack (nginx appends, compose hops 1, bind or firewall the client port when hops is 1 or more).
- INTEGRATOR TODO: skills/insideout-setup/SKILL.md: compose is mentioned and the variable is not; add the same sentence where the full compose stack is offered.
- INTEGRATOR TODO: scripts/env.sh: the empty DATABASE_OWNER_URL info line still says migrate uses DATABASE_URL and it must already be the owner, without the serve refusal or the migrate-and-seed-only scope.
