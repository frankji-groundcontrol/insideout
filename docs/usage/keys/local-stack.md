# Local-stack Postgres passwords

`POSTGRES_APP_PASSWORD` / `POSTGRES_OWNER_PASSWORD` — the two role
passwords for the **bundled docker-compose** Postgres only (setup path
b in SETENV.md). Not needed when pointing at an external instance.

## 1. Where the `.key` files are

`hosts/<host>/keys/database/.secrets/postgres-app-password.key` and
`postgres-owner-password.key` (manifest `database/local-stack.yaml`).
Split from the DSN keys because they belong to a different
installation (the disposable local stack).

## 2. How it is propagated

`docker-compose.yml` interpolates both from `.env` at `up` time to
provision the container's roles. Nothing else reads them.

## 3. How to set them

Self-minted (`openssl rand -hex 32`), with one hard rule: the app
password must equal the password embedded in `DATABASE_URL`, and the
owner password the one in `DATABASE_OWNER_URL`, when those DSNs point
at the compose stack (`localhost:5442`). Mint when the stack is next
used — staying empty otherwise is by design.

## 4. How to test them

`./scripts/env.sh check` cross-checks the compose app password against
`DATABASE_URL` automatically; then `docker compose up -d` and a
`migrate` against `localhost:5442` proves both roles answer.

## 5. How they are used

`docker-compose.yml` creates `insideout_app` and `insideout_owner` in
the ephemeral container with these passwords; the DSNs then
authenticate as those roles.

## 6. Installation guide

[SETENV.md](../../../SETENV.md) "Only if you run docker compose"
(Step 5) and the compose walkthrough in
[local-development.md](../local-development.md).
