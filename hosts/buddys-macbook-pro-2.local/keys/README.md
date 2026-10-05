# Keys — insideout credentials on this host

Same scheme as the other host: manifests tracked, values only in each
module's git-ignored `.secrets/<name>.key` (mode 0600, `ls -a` to see).

| Module | Manifest | Keys (env var) | State |
| --- | --- | --- | --- |
| [auth/](auth/) | [jwt.yaml](auth/jwt.yaml) | jwt-secret (`INSIDEOUT_JWT_SECRET`) | filled 2026-10-05 (openssl-minted, per-environment) |
| [database/](database/) | [postgres.yaml](database/postgres.yaml) | database-app-dsn (`DATABASE_URL`), database-owner-dsn (`DATABASE_OWNER_URL`) | both filled 2026-10-05 (pooler DSNs; migrate probe green) |
| [database/](database/) | [local-stack.yaml](database/local-stack.yaml) | postgres-app-password (`POSTGRES_APP_PASSWORD`) | filled 2026-10-05 (copied; compose stack unused here) — postgres-owner-password empty by design |
| [github/](github/) | [app.yaml](github/app.yaml) | gh-app-id, gh-app-private-key, gh-client-secret (`INSIDEOUT_GH_*`) | filled 2026-10-05; the PEM `.key` is real multi-line (normalized from the escaped form — the `_FILE` consumer reads it raw) |
| [github/](github/) | [webhooks.yaml](github/webhooks.yaml) | gh-webhook-secret (`INSIDEOUT_GH_WEBHOOK_SECRET`) | filled 2026-10-05 |
| [github/](github/) | [tokens.yaml](github/tokens.yaml) | github-token (`GITHUB_TOKEN`) | empty by design (optional PAT) |
| [llm/](llm/) | [provider.yaml](llm/provider.yaml) | llm-api-key (`INSIDEOUT_LLM_API_KEY`) | filled 2026-10-05; boot probe: real coach |
| [mail/](mail/) | [resend.yaml](mail/resend.yaml) | mail-api-key (`INSIDEOUT_MAIL_API_KEY`) | filled 2026-10-05; magic-link request accepted (200) |

10 of 12 filled; `postgres-owner-password` and `github-token` are
operator-minted and empty by design. The PEM is consumed via
`INSIDEOUT_GH_PRIVATE_KEY_FILE` pointing at its `.key`. With more than
one host in `hosts/`, scope the tooling:
`INSIDEOUT_KEYS_HOST=buddys-macbook-pro-2.local python3 scripts/keys.py
audit|fill` (never prints values). Per-key shape tests:
`python3 -m unittest discover -s tests/keys -t .` from `scripts/`.
The root `.env` stays the only file running processes read
([docs/SETENV.md](../../../../docs/SETENV.md) is the bridge manual).
