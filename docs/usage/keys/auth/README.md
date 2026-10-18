# Keys — auth module

Per-key reference for the `auth/` module of the key store. Each file
documents one key kind with the six parts (where the `.key` file is,
propagation, setting, testing, use, and the installation-guide
pointer). The module table, shared flow, and per-host state live in
the [keys README](../README.md).

Set `INSIDEOUT_JWT_SECRET` during the required server setup, then run the
masked server check before booting the API. The installation sequence is
[SETENV Step 2](../../../SETENV.md#step-2--insideout_jwt_secret-required).

- [jwt-secret.md](jwt-secret.md) — `INSIDEOUT_JWT_SECRET`
  (manifest `auth/jwt.yaml`)
