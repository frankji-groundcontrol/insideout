# Key reference (per key, six parts)

Every credential kind gets a standalone doc with the same six parts:
**where** the `.key` file is, **how it is propagated**, **how to set
it**, **how to test it**, **how it is used**, and **where the
installation guide** covers it (the convention agents_home's
`docs/usage/keys/` established). The layout mirrors the store exactly:
one module folder per
[`hosts/<host>/keys/`](../../../hosts/README.md) module, one file per
manifest, filename = the manifest's `setting`.

| Module | File | Keys (env var) |
| --- | --- | --- |
| [auth/](auth/) | [jwt.md](auth/jwt.md) | jwt-secret (`INSIDEOUT_JWT_SECRET`) |
| [database/](database/) | [postgres.md](database/postgres.md) | database-app-dsn (`DATABASE_URL`), database-owner-dsn (`DATABASE_OWNER_URL`) |
| [database/](database/) | [local-stack.md](database/local-stack.md) | postgres-app-password, postgres-owner-password (`POSTGRES_*_PASSWORD`) |
| [github/](github/) | [app.md](github/app.md) | gh-app-id, gh-app-private-key, gh-client-secret (`INSIDEOUT_GH_*`) |
| [github/](github/) | [webhooks.md](github/webhooks.md) | gh-webhook-secret (`INSIDEOUT_GH_WEBHOOK_SECRET`) |
| [github/](github/) | [tokens.md](github/tokens.md) | github-token (`GITHUB_TOKEN`, optional) |
| [llm/](llm/) | [provider.md](llm/provider.md) | llm-api-key (`INSIDEOUT_LLM_API_KEY`) |
| [mail/](mail/) | [resend.md](mail/resend.md) | mail-api-key (`INSIDEOUT_MAIL_API_KEY`) |

[SETENV.md](../../SETENV.md) — the `.env` operating manual — links
into these files from its per-key steps; the detail lives here, not
there.

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

The variable-level contract for every env var (credential or not) is
[environment.md](../environment.md); the first-time path is
[local-development.md](../local-development.md).
