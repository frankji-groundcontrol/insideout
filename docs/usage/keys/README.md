# Key reference (per key, six parts)

Every env var mapped in the key store gets a standalone doc with the
same six parts: **where** the `.key` file is, **how it is
propagated**, **how to set it**, **how to test it**, **how it is
used**, and **where the installation guide** covers it (the
convention agents_home's `docs/usage/keys/` established, model ids
and endpoints included). The filename is the **key name** — exactly
like the `.key` file it documents — inside the module folder of its
manifest under [`hosts/<host>/keys/`](../../../hosts/README.md).

**Never-absent rule** — a kind may be *blank* (a catalog or compose
default is in effect) but never *missing*: every mapped kind has a
`.key` file on every host the store describes (`keys.py init` creates
the empty placeholders), a row here, and a per-key test. Just because
a var can be defaulted doesn't mean it is left out.

| Module | File | Env var | Manifest |
| --- | --- | --- | --- |
| [app/](app/) | [addr.md](app/addr.md) | `INSIDEOUT_ADDR` | `app/runtime.yaml` |
| [app/](app/) | [cookie-secure.md](app/cookie-secure.md) | `INSIDEOUT_COOKIE_SECURE` | `app/runtime.yaml` |
| [app/](app/) | [cors-origins.md](app/cors-origins.md) | `INSIDEOUT_CORS_ORIGINS` | `app/runtime.yaml` |
| [app/](app/) | [dev-cors.md](app/dev-cors.md) | `INSIDEOUT_DEV_CORS` | `app/runtime.yaml` |
| [app/](app/) | [access-ttl.md](app/access-ttl.md) | `INSIDEOUT_ACCESS_TTL` | `app/runtime.yaml` |
| [app/](app/) | [refresh-ttl.md](app/refresh-ttl.md) | `INSIDEOUT_REFRESH_TTL` | `app/runtime.yaml` |
| [auth/](auth/) | [jwt-secret.md](auth/jwt-secret.md) | `INSIDEOUT_JWT_SECRET` | `auth/jwt.yaml` |
| [compose/](compose/) | [server-port.md](compose/server-port.md) | `SERVER_PORT` | `compose/stack.yaml` |
| [compose/](compose/) | [postgres-port.md](compose/postgres-port.md) | `POSTGRES_PORT` | `compose/stack.yaml` |
| [compose/](compose/) | [postgres-superuser-password.md](compose/postgres-superuser-password.md) | `POSTGRES_SUPERUSER_PASSWORD` | `compose/stack.yaml` |
| [compose/](compose/) | [client-port.md](compose/client-port.md) | `CLIENT_PORT` | `compose/stack.yaml` |
| [database/](database/) | [database-app-dsn.md](database/database-app-dsn.md) | `DATABASE_URL` | `database/postgres.yaml` |
| [database/](database/) | [database-owner-dsn.md](database/database-owner-dsn.md) | `DATABASE_OWNER_URL` | `database/postgres.yaml` |
| [database/](database/) | [postgres-app-password.md](database/postgres-app-password.md) | `POSTGRES_APP_PASSWORD` | `database/local-stack.yaml` |
| [database/](database/) | [postgres-owner-password.md](database/postgres-owner-password.md) | `POSTGRES_OWNER_PASSWORD` | `database/local-stack.yaml` |
| [github/](github/) | [gh-app-id.md](github/gh-app-id.md) | `INSIDEOUT_GH_APP_ID` | `github/app.yaml` |
| [github/](github/) | [gh-app-private-key.md](github/gh-app-private-key.md) | `INSIDEOUT_GH_PRIVATE_KEY` | `github/app.yaml` |
| [github/](github/) | [gh-client-secret.md](github/gh-client-secret.md) | `INSIDEOUT_GH_CLIENT_SECRET` | `github/app.yaml` |
| [github/](github/) | [gh-private-key-file.md](github/gh-private-key-file.md) | `INSIDEOUT_GH_PRIVATE_KEY_FILE` (kind: path) | `github/app.yaml` |
| [github/](github/) | [gh-webhook-secret.md](github/gh-webhook-secret.md) | `INSIDEOUT_GH_WEBHOOK_SECRET` | `github/webhooks.yaml` |
| [github/](github/) | [github-token.md](github/github-token.md) | `GITHUB_TOKEN` (optional) | `github/tokens.yaml` |
| [llm/](llm/) | [llm-api-key.md](llm/llm-api-key.md) | `INSIDEOUT_LLM_API_KEY` | `llm/provider.yaml` |
| [llm/](llm/) | [llm-base-url.md](llm/llm-base-url.md) | `INSIDEOUT_LLM_BASE_URL` | `llm/provider.yaml` |
| [llm/](llm/) | [llm-model.md](llm/llm-model.md) | `INSIDEOUT_LLM_MODEL` | `llm/provider.yaml` |
| [llm/](llm/) | [llm-schema.md](llm/llm-schema.md) | `INSIDEOUT_LLM_SCHEMA` | `llm/provider.yaml` |
| [llm/](llm/) | [ai-max-concurrent.md](llm/ai-max-concurrent.md) | `INSIDEOUT_AI_MAX_CONCURRENT` | `llm/coach.yaml` |
| [llm/](llm/) | [ai-queue-wait.md](llm/ai-queue-wait.md) | `INSIDEOUT_AI_QUEUE_WAIT` | `llm/coach.yaml` |
| [mail/](mail/) | [mail-api-key.md](mail/mail-api-key.md) | `INSIDEOUT_MAIL_API_KEY` | `mail/resend.yaml` |
| [mail/](mail/) | [mail-from.md](mail/mail-from.md) | `INSIDEOUT_MAIL_FROM` | `mail/resend.yaml` |
| [mail/](mail/) | [public-url.md](mail/public-url.md) | `INSIDEOUT_PUBLIC_URL` | `mail/resend.yaml` |
| [mail/](mail/) | [mail-dev-body.md](mail/mail-dev-body.md) | `INSIDEOUT_MAIL_DEV_BODY` | `mail/dev.yaml` |

[SETENV.md](../../SETENV.md) — the `.env` operating manual — links
into these files from its per-key steps; the detail lives here, not
there.

## Shared flow (not repeated per file)

Every key lands the same way — the full sequence, with verification and
failure signals, is the practice record
[2026-10-05-setting-up-a-key.md](../../practices/2026-10-05-setting-up-a-key.md):

1. `./scripts/env.sh edit` — value into the root `.env` (the only file
   processes read; never printed).
2. `INSIDEOUT_KEYS_HOST=<host> python3 scripts/keys.py init` — an
   empty `.key` placeholder (0600) for every mapped kind, so the kind
   is never absent even when a default is in effect.
3. `INSIDEOUT_KEYS_HOST=<host> python3 scripts/keys.py fill` —
   empty-only copy of env-present values into `.secrets/<name>.key`.
4. Scoped `audit` + `./scripts/env.sh check` + the key's own probe
   (each file's "how to test").

## Testing

Two layers, both required before a key counts as set up:

1. **Shape/placement suite** (automated, repeatable) — from `scripts/`:

   ```bash
   python3 -m unittest discover -s tests/keys -t .
   ```

   [`scripts/tests/keys/`](../../../scripts/tests/keys/README.md)
   mirrors this tree: one test file per key (filename = key name)
   asserting the file exists (never-absent rule), mode 0600, the
   right env var mapped, non-empty unless the manifest carries a
   fallback/optional/operator_fills field, and the kind's shape —
   PEM framing for the App key, `re_` prefix for mail, DSN role
   (including pooler suffixes) for the databases, duration strings
   for TTLs, positive integers for ports/concurrency. Plus store-wide
   invariants (every mapped kind present on every present host) and
   the `keys.py` tool behaviors with multi-host scoping. Values are
   never printed; hosts without a `.secrets/` (CI) skip cleanly. Run
   it after any key lands, copies between hosts, or rotates — and on
   every host the store describes: its first cross-host run caught an
   escaped-form PEM, 0755 secrets dirs, and a wrong-case host folder
   on the other machine.
2. **Live probe** (each file's part 4) — the key actually working:
   `migrate` for the owner DSN, a bad-login 401 for the app DSN, the
   absent offline-coach notice for the LLM key, a magic-link request
   for mail, signed deliveries answered 200 for the webhook secret.
   A boot that only *warns* about an unusable key is a failure, not a
   pass — grep for the warning's absence.

## Per-host state (2026-10-06)

Filled = a value is in the `.key` file; blank-fallback = the file
exists empty, so the catalog/compose default is in effect (the
never-absent rule satisfied).

| Key | franks-macbook-pro | buddys-macbook-pro-2 |
| --- | --- | --- |
| jwt-secret | filled | filled |
| database-app-dsn / database-owner-dsn | filled | filled |
| llm-api-key | filled | filled |
| llm-base-url / llm-model / llm-schema | pending (offline coach locally; live values ride the deployment) | filled 2026-10-06 |
| ai-max-concurrent / ai-queue-wait | blank-fallback | blank-fallback |
| mail-api-key | filled | filled |
| mail-from / public-url | filled 2026-10-06 | filled 2026-10-06 |
| mail-dev-body | blank-fallback | blank-fallback |
| gh-webhook-secret / gh-app-id / gh-app-private-key / gh-client-secret | filled | filled (PEM normalized to real multi-line for the `_FILE` consumer) |
| gh-private-key-file | pending (uses the env-embedded lane) | filled 2026-10-06 |
| postgres-app-password | filled | filled |
| postgres-owner-password / github-token | empty by design | empty by design |
| addr / cookie-secure / cors-origins / dev-cors / access-ttl / refresh-ttl | blank-fallback | blank-fallback |
| server-port / postgres-port / postgres-superuser-password / client-port | blank-fallback | blank-fallback |

The variable-level contract for every env var (credential or not) is
[environment.md](../environment.md); the first-time path is
[local-development.md](../local-development.md).
