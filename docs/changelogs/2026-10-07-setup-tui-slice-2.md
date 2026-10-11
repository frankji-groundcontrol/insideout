# 2026-10-07 — Setup TUI slice 2: local mode assembled and live-exercised

**Scope:** slice 2 of the setup TUI plan — the local branch of
`insideout setup` (the B+D+E join), live-exercised against a real
PostgreSQL 18 cluster. Plan:
[`docs/plans/2026-10-07-setup-tui.md`](../plans/2026-10-07-setup-tui.md).

## What changed

- **`server/cmd/insideout/cli_setup_local.go`** — the local
  assembly: repo-checkout discovery (marker walk, honors
  `INSIDEOUT_SETUP_REPO_ROOT` as the test affordance), `.env`
  ensured from the skeleton through the Go envwrite package,
  `INSIDEOUT_JWT_SECRET` generated when placeholder/short
  (mirroring `setup.sh`'s rule), `DATABASE_URL` stored for the
  bundled DSN (app role, loopback, port/password from `.env`) or the
  stdin DSN (hidden read at a terminal, plain line when piped),
  bundled path `docker compose --env-file <env> up -d postgres`,
  native pgx probe with a 30s first-boot wait, in-process
  `store.Open` + `Migrate` (owner-pool branch when
  `DATABASE_OWNER_URL` is set), a boot offer that injects the
  written values into the child's environment (the server reads only
  process env) with `INSIDEOUT_COOKIE_SECURE=0`, and a summary gate
  (set/default/placeholder/missing counts, required-outstanding
  names — never values). `--no-start` mirrors the script: write
  `.env` only.
- `cli_setup.go`: the `--local` branch now runs the assembly (the
  old "not assembled" stub removed; `setup.sh` remains the
  pre-binary fallback).
- **Tests** (`cli_setup_local_test.go`): bundled DSN derivation, JWT
  rotation rule, child-env injection, env-value reader, checkout
  marker walk, summary-gate masking + outstanding naming, the
  `--no-start` end-to-end write path against a fake checkout with a
  piped DSN, and the missing-docker clean failure (skips where
  docker exists).

## Live exercise (real PostgreSQL, no mocks — no docker on this host)

Homebrew postgresql@18 scratch cluster (initdb on port 5599, roles
provisioned per `docker/postgres-init/001-create-app-role.sh`):
`setup --local --postgres own` with the DSN piped → secret written
(not printed; DSN shown redacted) → pgx probe pass → **31
migrations applied in-process** → summary `2 set · 27 missing ·
required satisfied` → exit 0. Server booted with the child-env values
against the scratch DB: **`/healthz` → 200, `/api/v1/me` → 401**;
clean stop; cluster torn down. Two real findings the exercise taught,
recorded as
[learning](../learning/2026-10-07-own-url-role-provisioning-precondition.md)
and [an issue](../issues/2026-10-07-own-url-fresh-db-bootstrap.md).

*Corrected 2026-10-11: the live run applied 31 migrations (the on-disk
file count), not 23 — the figure above is amended per the
[landed record](2026-10-07-setup-tui-landed.md), which corrected the count
on its walk the same day.*

## Verification

- `go vet ./...` clean; `go test ./...` **25 packages ok, 0
  failures**; Python env batteries **47 + 37 passed**; `gofmt` clean;
  dual-mode binary **19,832,274 bytes**.
- The bundled-Docker lane is unit-covered but **not live-run on this
  host** (no docker installed); its compose command construction,
  DSN derivation, and missing-docker failure are tested; the first
  host with Docker should run the bundled path end-to-end once.

## Operator notes

Nothing deployed. `setup.sh` is unchanged and still works. The
own-URL path requires the two NOSUPERUSER roles to pre-exist
(`insideout_owner` with BYPASSRLS owning the database, `inside_app`
with CONNECT) — the issue tracks teaching setup to check for them.
