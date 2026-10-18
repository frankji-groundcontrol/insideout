# jwt-secret

`INSIDEOUT_JWT_SECRET` — the HMAC signing secret for access and refresh JWTs.
The server refuses to start when it is empty or shorter than 32 characters.

## 1. Where the `.key` file is

`hosts/<host>/keys/auth/.secrets/jwt-secret.key` (manifest
`auth/jwt.yaml`). Store one line with no `KEY=` prefix and mode `0600`.

## 2. How it is propagated

The root `.env` is exported by `scripts/dev.sh`, and compose passes the value
directly into the API container. `keys.py fill` copies it only into an empty
host key; the key store is a backup, not a second runtime source.

## 3. How to set it

Run `./scripts/env.sh init` and choose the generate option when it asks for
`INSIDEOUT_JWT_SECRET`. This creates a fresh value without displaying it.
For an existing value, use `./scripts/env.sh edit`, select the variable, and
paste it without quotes. Generate a different secret for every environment.

## 4. How to test it

Before the live check, run the placement and shape suite from the repository root:

```bash
python3 -m unittest discover -s scripts/tests/keys -t . -p 'test_*.py'
```

It reports the key kind and never prints secret values.

Run the masked validation first:

```bash
./scripts/env.sh check server
```

Expected output contains `INSIDEOUT_JWT_SECRET ... set (N chars)` with no
secret text. Start the server through `scripts/dev.sh`; a usable secret lets
it boot, while a missing or short value fails before it opens the API. If the
check reports a short or placeholder value, generate a new one with
`./scripts/env.sh init`; never paste the secret into a command argument or log.

## 5. How it is used

The API signs access and refresh tokens for the web client, CLI, and MCP
server. Every process that must accept the tokens needs the same secret for
that environment.

## 6. Installation guide

The required local-server step is [Installation — Your own server](../../installation.md#your-own-server).
The key-by-key setup is [SETENV Step 2](../../../SETENV.md#step-2--insideout_jwt_secret-required),
and the variable contract is [environment.md](../../environment.md#go-backend).
