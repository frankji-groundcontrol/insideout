# Keys — insideout credentials on this host

Same scheme as the other host: manifests tracked, values only in each
module's git-ignored `.secrets/<name>.key` (mode 0600, `ls -a` to see).
Manifests land key-by-key as each key is actually set here — the table
is the live state.

| Module | Manifest | Keys (env var) | State |
| --- | --- | --- | --- |
| [auth/](auth/) | [jwt.yaml](auth/jwt.yaml) | jwt-secret (`INSIDEOUT_JWT_SECRET`) | filled 2026-10-05 (openssl-minted, per-environment) |

With more than one host in `hosts/`, scope the tooling:
`INSIDEOUT_KEYS_HOST=buddys-macbook-pro-2.local python3 scripts/keys.py
audit|fill` (never prints values). The root `.env` stays the only file
running processes read; this tree is the out-of-git store behind it
(`scripts/env.sh init` is the bridge — see
[docs/SETENV.md](../../../../docs/SETENV.md)).
