# Keys — app module

Per-key reference for the `app/` module of the key store. Each file
documents one key kind with the six parts (where the `.key` file is,
propagation, setting, testing, use, and the installation-guide
pointer). The module table, shared flow, and per-host state live in
the [keys README](../README.md).

For the guided local lane, set the required database and JWT keys first, then
choose only the app behavior you need. Set each value with
`./scripts/env.sh edit`, run `./scripts/env.sh check server`, and execute the
live probe in that key's section after restarting the server. The installation
sequence is [SETENV Step 4](../../../SETENV.md#step-4--local-dev-toggles).

- [addr.md](addr.md) — `INSIDEOUT_ADDR` (manifest `app/runtime.yaml`)
- [cookie-secure.md](cookie-secure.md) — `INSIDEOUT_COOKIE_SECURE`
  (`app/runtime.yaml`)
- [cors-origins.md](cors-origins.md) — `INSIDEOUT_CORS_ORIGINS`
  (`app/runtime.yaml`)
- [dev-cors.md](dev-cors.md) — `INSIDEOUT_DEV_CORS`
  (`app/runtime.yaml`)
- [access-ttl.md](access-ttl.md) — `INSIDEOUT_ACCESS_TTL`
  (`app/runtime.yaml`)
- [refresh-ttl.md](refresh-ttl.md) — `INSIDEOUT_REFRESH_TTL`
  (`app/runtime.yaml`)
