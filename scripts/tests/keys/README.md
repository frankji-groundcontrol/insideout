# Key test suites (per key, mirroring the store)

One test file per credential kind — filename = the key name, module
folder = the store's module — exactly mirroring
[`hosts/<host>/keys/`](../../../../hosts/README.md) and
[docs/usage/keys/](../../../../docs/usage/keys/README.md). Two
cross-cutting suites sit at this root: the tool behaviors
(`keys.py` parsing, fill, audit, multi-host scoping) and the
store-wide invariants (perms, non-empty, git-blindness, host.yml
naming).

| Module | File | Key (env var) |
| --- | --- | --- |
| [auth/](auth/) | [test_jwt-secret.py](auth/test_jwt-secret.py) | jwt-secret (`INSIDEOUT_JWT_SECRET`) |
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
| [mail/](mail/) | [test_mail-api-key.py](mail/test_mail-api-key.py) | mail-api-key (`INSIDEOUT_MAIL_API_KEY`) |
| [mail/](mail/) | [test_mail-from.py](mail/test_mail-from.py) | mail-from (display-name sender) |
| [mail/](mail/) | [test_public-url.py](mail/test_public-url.py) | public-url (absolute origin) |
| — | [test_tool.py](test_tool.py) | `keys.py` behaviors incl. multi-host scoping |
| — | [test_store.py](test_store.py) | store-wide invariants (real tree, CI-safe) |

Every per-key test asserts **shape and placement only** — mode 0600,
non-empty (unless operator-filled/optional), the right env var mapped,
and a kind-specific shape (PEM framing, `re_` prefix, DSN role,
numeric id, minimum strength). Values are never printed, logged, or
compared to literals. On a machine without a host's `.secrets/` (CI
checkouts), the per-key suites skip cleanly.

Run from `scripts/`:

```bash
python3 -m unittest discover -s tests/keys -t . -p "test_*.py" -v
```
