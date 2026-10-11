# Keys — compose module

Per-key reference for the `compose/` module of the key store. Each
file documents one key kind with the six parts (where the `.key` file
is, propagation, setting, testing, use, and the installation-guide
pointer). The module table, shared flow, and per-host state live in
the [keys README](../README.md).

- [server-port.md](server-port.md) — `SERVER_PORT`
  (manifest `compose/stack.yaml`)
- [postgres-port.md](postgres-port.md) — `POSTGRES_PORT`
  (`compose/stack.yaml`)
- [postgres-superuser-password.md](postgres-superuser-password.md) —
  `POSTGRES_SUPERUSER_PASSWORD` (`compose/stack.yaml`)
- [client-port.md](client-port.md) — `CLIENT_PORT`
  (`compose/stack.yaml`)
