# Key test suites (per key, mirroring the store)

One test file per mapped kind — filename = the key name, module
folder = the store's module — exactly mirroring
[`hosts/<host>/keys/`](../../../hosts/README.md) and
[docs/usage/keys/](../../../docs/usage/keys/README.md). Two
cross-cutting suites sit at this root: the tool behaviors
(`keys.py` parsing, init, fill, audit, multi-host scoping) and the
store-wide invariants (perms, never-absent, emptiness only with a
fallback/optional/operator_fills field, git-blindness, host.yml
naming).

| Module | File | Key (env var) |
| --- | --- | --- |
| [app/](app/) | [test_addr.py](app/test_addr.py) | addr (`INSIDEOUT_ADDR`, host:port) |
| [app/](app/) | [test_cookie-secure.py](app/test_cookie-secure.py) | cookie-secure (blank or exactly `0`) |
| [app/](app/) | [test_cors-origins.py](app/test_cors-origins.py) | cors-origins (CSV of origins) |
| [app/](app/) | [test_dev-cors.py](app/test_dev-cors.py) | dev-cors (blank or exactly `1`) |
| [app/](app/) | [test_access-ttl.py](app/test_access-ttl.py) | access-ttl (Go duration) |
| [app/](app/) | [test_refresh-ttl.py](app/test_refresh-ttl.py) | refresh-ttl (Go duration) |
| [auth/](auth/) | [test_jwt-secret.py](auth/test_jwt-secret.py) | jwt-secret (`INSIDEOUT_JWT_SECRET`) |
| [compose/](compose/) | [test_server-port.py](compose/test_server-port.py) | server-port (TCP port) |
| [compose/](compose/) | [test_postgres-port.py](compose/test_postgres-port.py) | postgres-port (TCP port) |
| [compose/](compose/) | [test_postgres-superuser-password.py](compose/test_postgres-superuser-password.py) | postgres-superuser-password (non-placeholder) |
| [compose/](compose/) | [test_client-port.py](compose/test_client-port.py) | client-port (TCP port) |
| [database/](database/) | [test_database-app-dsn.py](database/test_database-app-dsn.py) | database-app-dsn (`DATABASE_URL`) |
| [database/](database/) | [test_database-owner-dsn.py](database/test_database-owner-dsn.py) | database-owner-dsn (`DATABASE_OWNER_URL`) |
| [database/](database/) | [test_postgres-app-password.py](database/test_postgres-app-password.py) | postgres-app-password (`POSTGRES_APP_PASSWORD`) |
| [database/](database/) | [test_postgres-owner-password.py](database/test_postgres-owner-password.py) | postgres-owner-password (operator-filled) |
| [github/](github/) | [test_gh-app-id.py](github/test_gh-app-id.py) | gh-app-id (`INSIDEOUT_GH_APP_ID`) |
| [github/](github/) | [test_gh-app-private-key.py](github/test_gh-app-private-key.py) | gh-app-private-key (PEM shape) |
| [github/](github/) | [test_gh-client-secret.py](github/test_gh-client-secret.py) | gh-client-secret (`INSIDEOUT_GH_CLIENT_SECRET`) |
| [github/](github/) | [test_gh-private-key-file.py](github/test_gh-private-key-file.py) | gh-private-key-file (pointer target exists) |
| [github/](github/) | [test_gh-webhook-secret.py](github/test_gh-webhook-secret.py) | gh-webhook-secret (`INSIDEOUT_GH_WEBHOOK_SECRET`) |
| [github/](github/) | [test_github-token.py](github/test_github-token.py) | github-token (optional) |
| [llm/](llm/) | [test_llm-api-key.py](llm/test_llm-api-key.py) | llm-api-key (`INSIDEOUT_LLM_API_KEY`) |
| [llm/](llm/) | [test_llm-base-url.py](llm/test_llm-base-url.py) | llm-base-url (absolute http(s) URL) |
| [llm/](llm/) | [test_llm-model.py](llm/test_llm-model.py) | llm-model (non-empty id) |
| [llm/](llm/) | [test_llm-schema.py](llm/test_llm-schema.py) | llm-schema (messages|responses) |
| [llm/](llm/) | [test_ai-max-concurrent.py](llm/test_ai-max-concurrent.py) | ai-max-concurrent (positive int) |
| [llm/](llm/) | [test_ai-queue-wait.py](llm/test_ai-queue-wait.py) | ai-queue-wait (Go duration) |
| [mail/](mail/) | [test_mail-api-key.py](mail/test_mail-api-key.py) | mail-api-key (`INSIDEOUT_MAIL_API_KEY`) |
| [mail/](mail/) | [test_mail-from.py](mail/test_mail-from.py) | mail-from (display-name sender) |
| [mail/](mail/) | [test_public-url.py](mail/test_public-url.py) | public-url (absolute origin) |
| [mail/](mail/) | [test_mail-dev-body.py](mail/test_mail-dev-body.py) | mail-dev-body (blank or exactly `1`) |
| — | [test_tool.py](test_tool.py) | `keys.py` behaviors incl. init + multi-host scoping |
| — | [test_store.py](test_store.py) | store-wide invariants incl. never-absent (real tree, CI-safe) |

Every per-key test asserts **placement and shape only** — the file
exists (never-absent: blank is legal, absent is not), mode 0600, the
right env var mapped, and a kind-specific shape (PEM framing, `re_`
prefix, DSN role, numeric id, Go duration, positive integer, port
range). Values are never printed, logged, or compared to literals. On
a machine without a host's `.secrets/` (CI checkouts), the per-key
suites skip cleanly.

Run from `scripts/`:

```bash
python3 -m unittest discover -s tests/keys -t . -p "test_*.py" -v
```
