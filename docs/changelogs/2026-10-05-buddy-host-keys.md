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

## Operators / users

Nothing to do elsewhere. Remaining keys land in later slices per the
plan checklist (DSNs + LLM are env-set locally; mail and the GitHub
keys copy from the filled store on MBP; compose passwords mint on
next use; the PAT stays optional-empty).
