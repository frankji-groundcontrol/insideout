# Key reference (per key, six parts)

Every credential kind gets a standalone doc with the same six parts:
**where** the `.key` file is, **how it is propagated**, **how to set
it**, **how to test it**, **how it is used**, and **where the
installation guide** covers it (the convention agents_home's
`docs/usage/keys/` established). Files mirror the store's manifests
under [`hosts/<host>/keys/`](../../../hosts/README.md) one to one.

## Files

| File | Keys (env var) | Manifest |
| --- | --- | --- |
| [jwt.md](jwt.md) | jwt-secret (`INSIDEOUT_JWT_SECRET`) | `auth/jwt.yaml` |
| [postgres.md](postgres.md) | database-app-dsn (`DATABASE_URL`), database-owner-dsn (`DATABASE_OWNER_URL`) | `database/postgres.yaml` |
| [local-stack.md](local-stack.md) | postgres-app-password, postgres-owner-password (`POSTGRES_*_PASSWORD`) | `database/local-stack.yaml` |
| [gh-app.md](gh-app.md) | gh-app-id, gh-app-private-key, gh-client-secret (`INSIDEOUT_GH_*`) | `github/app.yaml` |
| [gh-webhooks.md](gh-webhooks.md) | gh-webhook-secret (`INSIDEOUT_GH_WEBHOOK_SECRET`) | `github/webhooks.yaml` |
| [github-token.md](github-token.md) | github-token (`GITHUB_TOKEN`, optional) | `github/tokens.yaml` |
| [llm-provider.md](llm-provider.md) | llm-api-key (`INSIDEOUT_LLM_API_KEY`) | `llm/provider.yaml` |
| [mail-resend.md](mail-resend.md) | mail-api-key (`INSIDEOUT_MAIL_API_KEY`) | `mail/resend.yaml` |

## Shared flow (not repeated per file)

Every key lands the same way — the full sequence, with verification and
failure signals, is the practice record
[2026-10-05-setting-up-a-key.md](../../practices/2026-10-05-setting-up-a-key.md):

1. `./scripts/env.sh edit` — value into the root `.env` (the only file
   processes read; never printed).
2. `INSIDEOUT_KEYS_HOST=<host> python3 scripts/keys.py fill` —
   empty-only copy into the module's `.secrets/<name>.key` (0600).
3. Scoped `audit` + `./scripts/env.sh check` + the key's own probe
   (each file's "how to test").

## Per-host state (2026-10-05)

| Kind | franks-macbook-pro | buddys-macbook-pro-2 |
| --- | --- | --- |
| jwt-secret | filled | filled |
| database-app-dsn / -owner-dsn | filled | filled |
| llm-api-key | filled | pending |
| mail-api-key | filled | pending |
| gh-webhook-secret / gh-app-id / gh-app-private-key / gh-client-secret | filled | pending |
| postgres-app-password | filled | pending |
| postgres-owner-password / github-token | empty by design | pending |

The installation guide proper is [SETENV.md](../../SETENV.md); this
folder adds the per-credential layer on top.
