# 2026-10-05 — Buddy host key store (key-by-key, jwt first)

**Scope:** bring `buddys-macbook-pro-2.local` (the secondary dev
workstation) into the per-host key scheme as a second host, one key at a
time following the six-part convention (where / propagated / set /
tested / used / install guide) that the agents_home family doc
established. Hosts tree + records only; no application code, nothing
deployed. Plan:
[`docs/plans/2026-10-05-buddy-host-keys.md`](../plans/2026-10-05-buddy-host-keys.md).

## What changed (slice 1 — jwt)

- **`hosts/buddys-macbook-pro-2.local/`** — new host folder: `host.yml`
  (metadata only; name and `ssh_alias: MBP2` match agents_home's record
  for the same machine), `README.md`, `keys/README.md` (live state
  table), and the first manifest `keys/auth/jwt.yaml`. Manifests land
  key-by-key as each key is actually set here, so the tracked tree
  always reflects verified state.
- **Value filled** from the local `.env` with the sanctioned tool,
  scoped to the new host: `INSIDEOUT_KEYS_HOST=buddys-macbook-pro-2.local
  python3 scripts/keys.py fill` — wrote `.secrets/jwt-secret.key`
  (dir 0700, file 0600, values never printed). The secret was
  openssl-minted by the user earlier this session; per-environment by
  design.
- **`hosts/README.md`** — second host row + the multi-host note:
  audit/fill need `INSIDEOUT_KEYS_HOST` once two hosts exist.

## Verification

- Scoped fill printed `1 written, 0 kept, 0 skipped`; scoped audit shows
  `INSIDEOUT_JWT_SECRET key=filled env=set`; file is 64 B at mode 0600
  inside a 0700 dir — consistent with `openssl rand -hex 32`.
- Isolation proven: `INSIDEOUT_KEYS_HOST=franks-macbook-pro.local audit`
  still reports every MBP key `missing` on this machine (its values live
  on that machine); the scoped fill cannot touch another host's paths.
- `git status` shows only the tracked manifest/readme files — the
  `.secrets/` dotfolder is git-ignored as designed.

## What changed (slice 2 — usage installation part)

- **`docs/usage/keys.md`** — the six-part reference for the key store
  (where / propagated / set / tested / used / install guide), mirroring
  agents_home's `docs/usage/keys/` convention; one family doc rather
  than a file per key because every key here shares one flow. Includes
  the 12-kind table with value sources and per-host state, the scoped
  audit/fill loop, per-key test probes, and boot-time usage roles.
- Indexed as a row in `docs/usage/README.md`; cross-linked from the
  `docs/SETENV.md` hosts paragraph (install guide ↔ store reference).

## What changed (slice 3 — practice record)

- **`docs/practices/2026-10-05-setting-up-a-key.md`** — the repeatable
  full sequence for landing any key on any host, distilled from the jwt
  slice: preconditions, manifest-first, `env.sh` as the value bridge,
  scoped `keys.py fill`, the four-point verification (audit / perms /
  cross-host isolation / git-blind check), the per-key probe, records,
  and the failure signals (skip-when-placeholder, the multi-host
  refusal that prevents value mixing, git-visible `.secrets`). Indexed
  in `docs/practices/README.md`.

## What changed (slice 4 — Postgres DSNs + supabase→postgres rename)

- **`database/postgres.yaml` on both hosts** — the manifest was
  `supabase.yaml`; renamed because the *kind* is the role-scoped
  Postgres DSNs while Supabase is merely today's host of the instance
  (user call, 2026-10-05). Values unaffected: `.key` files live in the
  module's shared `.secrets/`, keyed by name, not by manifest filename
  — MBP's filled values survive the rename untouched.
- **Buddy: both DSNs filled** from the local `.env` via the scoped
  `keys.py fill` (`2 written, 1 kept` — the kept one is jwt,
  empty-only semantics observed). Files 161/163 B at 0600 — the same
  byte sizes MBP's store records for the same DSNs.
- **Docs surfaces updated**: both hosts' `keys/README.md` tables
  (postgres.yaml row; Buddy state row), `docs/usage/keys.md` DSN rows
  (source now "managed Postgres DSN (Supabase today)"; Buddy column
  filled), plan checklist.

## Verification (slice 4)

- Scoped audit after the rename: all 3 mapped kinds `key=filled
  env=set` (the glob reads `database/postgres.yaml` fine); git status
  shows no `.secrets` paths.
- `./scripts/env.sh check`: 4 ok, 0 warnings, 0 failures.
- Owner-DSN live probe:
  `./scripts/dev.sh -C server go run ./cmd/insideout migrate` →
  `migrate: nothing to do, already up to date` — real connection,
  real authentication, currency confirmed.

## Operators / users

Nothing to do elsewhere. Remaining keys land in later slices per the
plan checklist (DSNs + LLM are env-set locally; mail and the GitHub
keys copy from the filled store on MBP; compose passwords mint on
next use; the PAT stays optional-empty).
