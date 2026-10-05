# jwt-secret

`INSIDEOUT_JWT_SECRET` — HMAC signing secret for access/refresh JWTs.

## 1. Where the `.key` file is

`hosts/<host>/keys/auth/.secrets/jwt-secret.key` (manifest
`auth/jwt.yaml`). One line, no `KEY=` prefix, 0600.

## 2. How it is propagated

The root `.env` is the consumer — `scripts/dev.sh` exports it before
every command; `docker-compose` passes it through to the server. The
store copy is the backup; `keys.py fill` bridges env→empty-key.

## 3. How to set it

Self-minted: `openssl rand -hex 32` (64 hex chars). **One per
environment** — local, each deployment — never reuse across them; ≥32
characters is fail-fast in `server/internal/config`.

## 4. How to test it

`./scripts/env.sh check` (masked), then boot the server — a missing or
short secret aborts at config load with a named error. Issued tokens
are accepted on every request of the same server instance.

## 5. How it is used

Signs access and refresh JWTs for every authenticated surface (web,
CLI, MCP — they all ride the same tokens).

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 2
links here. The variable's contract row (`INSIDEOUT_JWT_SECRET`,
required, min 32 chars) is in
[environment.md](../../environment.md).
