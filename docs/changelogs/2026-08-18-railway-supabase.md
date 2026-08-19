# 2026-08-18 — Railway server uses shared Supabase, dedicated Postgres removed

## What changed

The hosted Go API now uses the same shared-instance Postgres as
`supabase-community` (`insideout` schema, role `insideout_app`). Session
pooler on port **5432** (not 6543, not 6432). The dedicated Railway
`Postgres` service was deleted after a live write-back check.

`INSIDEOUT_LLM_*` vars were copied onto Railway `server` (stdin, not
logged). The new LLM env names shipped in the same deploy.

## Verification

- Local server against that DSN: register `lp083739` + workspace visible
  via MCP SQL.
- After `railway up --service server` (SUCCESS) and variable update:
  `GET /healthz` 200; register `rc084135` on the public API; MCP SQL
  showed workspace `Railway Cutover WS 20260818084135`.
- `railway service list` is `app` + `server` only.

## Operator notes

Railway `live172819` data lived on the deleted plugin instance and is
gone from the hosted API. The 358-user Supabase corpus is what production
reads now. Set `DATABASE_URL` only via `--stdin`. Do not reconnect
`${{Postgres.DATABASE_URL}}`.
