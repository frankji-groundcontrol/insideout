# Keys — compose module

Per-key reference for the `compose/` module of the key store. Each
file documents one key kind with the six parts (where the `.key` file
is, propagation, setting, testing, use, and the installation-guide
pointer). The module table, shared flow, and per-host state live in
the [keys README](../README.md).

These keys apply only to the bundled Docker lane. Set a port or bootstrap
password with `./scripts/env.sh edit`, run `docker compose config --quiet`, and
use the key's probe after the relevant service is healthy. The installation
sequence is [SETENV Step 5](../../../SETENV.md#step-5--only-if-you-run-docker-compose).

- [server-port.md](server-port.md) — `SERVER_PORT`
  (manifest `compose/stack.yaml`)
- [postgres-port.md](postgres-port.md) — `POSTGRES_PORT`
  (`compose/stack.yaml`)
- [postgres-superuser-password.md](postgres-superuser-password.md) —
  `POSTGRES_SUPERUSER_PASSWORD` (`compose/stack.yaml`)
- [client-port.md](client-port.md) — `CLIENT_PORT`
  (`compose/stack.yaml`)
