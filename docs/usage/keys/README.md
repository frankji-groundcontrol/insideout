# Key reference (per key, six parts)

Every credential kind gets a standalone doc with the same six parts:
**where** the `.key` file is, **how it is propagated**, **how to set
it**, **how to test it**, **how it is used**, and **where the
installation guide** covers it (the convention agents_home's
`docs/usage/keys/` established). The filename is the **key name** —
exactly like the `.key` file it documents — inside the module folder
of its manifest under
[`hosts/<host>/keys/`](../../../hosts/README.md).

| Module | File | Env var | Manifest |
| --- | --- | --- | --- |
| [auth/](auth/) | [jwt-secret.md](auth/jwt-secret.md) | `INSIDEOUT_JWT_SECRET` | `auth/jwt.yaml` |
| [database/](database/) | [database-app-dsn.md](database/database-app-dsn.md) | `DATABASE_URL` | `database/postgres.yaml` |
| [database/](database/) | [database-owner-dsn.md](database/database-owner-dsn.md) | `DATABASE_OWNER_URL` | `database/postgres.yaml` |
| [database/](database/) | [postgres-app-password.md](database/postgres-app-password.md) | `POSTGRES_APP_PASSWORD` | `database/local-stack.yaml` |
| [database/](database/) | [postgres-owner-password.md](database/postgres-owner-password.md) | `POSTGRES_OWNER_PASSWORD` | `database/local-stack.yaml` |
| [github/](github/) | [gh-app-id.md](github/gh-app-id.md) | `INSIDEOUT_GH_APP_ID` | `github/app.yaml` |
| [github/](github/) | [gh-app-private-key.md](github/gh-app-private-key.md) | `INSIDEOUT_GH_PRIVATE_KEY` | `github/app.yaml` |
| [github/](github/) | [gh-client-secret.md](github/gh-client-secret.md) | `INSIDEOUT_GH_CLIENT_SECRET` | `github/app.yaml` |
| [github/](github/) | [gh-webhook-secret.md](github/gh-webhook-secret.md) | `INSIDEOUT_GH_WEBHOOK_SECRET` | `github/webhooks.yaml` |
| [github/](github/) | [github-token.md](github/github-token.md) | `GITHUB_TOKEN` (optional) | `github/tokens.yaml` |
| [llm/](llm/) | [llm-api-key.md](llm/llm-api-key.md) | `INSIDEOUT_LLM_API_KEY` | `llm/provider.yaml` |
| [mail/](mail/) | [mail-api-key.md](mail/mail-api-key.md) | `INSIDEOUT_MAIL_API_KEY` | `mail/resend.yaml` |

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

| Key | franks-macbook-pro | buddys-macbook-pro-2 |
| --- | --- | --- |
| jwt-secret | filled | filled |
| database-app-dsn / database-owner-dsn | filled | filled |
| llm-api-key | filled | filled |
| mail-api-key | filled | filled |
| gh-webhook-secret / gh-app-id / gh-app-private-key / gh-client-secret | filled | filled (PEM normalized to real multi-line for the `_FILE` consumer) |
| postgres-app-password | filled | filled |
| postgres-owner-password / github-token | empty by design | empty by design |

Per-key shape tests mirroring this tree:
`scripts/tests/keys/` (run `python3 -m unittest discover -s tests/keys
-t .` from `scripts/`; CI runs it in the env-suites job).

The variable-level contract for every env var (credential or not) is
[environment.md](../environment.md); the first-time path is
[local-development.md](../local-development.md).
