# Keys — insideout credentials on this host

Same scheme as the other host: manifests tracked, values only in each
module's git-ignored `.secrets/<name>.key` (mode 0600, `ls -a` to see).

| Module | Manifest | Keys (env var) | State |
| --- | --- | --- | --- |
| [app/](app/) | [runtime.yaml](app/runtime.yaml) | addr, cookie-secure, cors-origins, dev-cors, access-ttl, refresh-ttl (`INSIDEOUT_ADDR` … `INSIDEOUT_REFRESH_TTL`) | all blank-fallback 2026-10-06 (files exist empty; catalog defaults in effect) |
| [auth/](auth/) | [jwt.yaml](auth/jwt.yaml) | jwt-secret (`INSIDEOUT_JWT_SECRET`) | filled 2026-10-05 (openssl-minted, per-environment) |
| [compose/](compose/) | [stack.yaml](compose/stack.yaml) | server-port, postgres-port, postgres-superuser-password, client-port (`SERVER_PORT` … `CLIENT_PORT`) | all blank-fallback 2026-10-06 (compose `:-` defaults in effect; stack unused here) |
| [database/](database/) | [postgres.yaml](database/postgres.yaml) | database-app-dsn (`DATABASE_URL`), database-owner-dsn (`DATABASE_OWNER_URL`) | both filled 2026-10-05 (pooler DSNs; migrate probe green) |
| [database/](database/) | [local-stack.yaml](database/local-stack.yaml) | postgres-app-password (`POSTGRES_APP_PASSWORD`) | filled 2026-10-05 (copied; compose stack unused here) — postgres-owner-password empty by design |
| [github/](github/) | [app.yaml](github/app.yaml) | gh-app-id, gh-app-private-key, gh-client-secret, gh-private-key-file (`INSIDEOUT_GH_*` + the `_FILE` pointer, kind: path) | filled; the PEM `.key` is real multi-line (normalized from the escaped form — the `_FILE` consumer reads it raw); the pointer's test asserts its target exists |
| [github/](github/) | [webhooks.yaml](github/webhooks.yaml) | gh-webhook-secret (`INSIDEOUT_GH_WEBHOOK_SECRET`) | filled 2026-10-05 |
| [github/](github/) | [tokens.yaml](github/tokens.yaml) | github-token (`GITHUB_TOKEN`) | empty by design (optional PAT) |
| [llm/](llm/) | [provider.yaml](llm/provider.yaml) | llm-api-key + companions llm-base-url, llm-model, llm-schema (`INSIDEOUT_LLM_*`) | all four filled (companions 2026-10-06); provider ping 200, boot probe: real coach |
| [llm/](llm/) | [coach.yaml](llm/coach.yaml) | ai-max-concurrent, ai-queue-wait (`INSIDEOUT_AI_*`) | both blank-fallback 2026-10-06 (defaults 4 / 15s in effect) |
| [mail/](mail/) | [resend.yaml](mail/resend.yaml) | mail-api-key + companions mail-from, public-url (`INSIDEOUT_MAIL_*`, `INSIDEOUT_PUBLIC_URL`) | all three filled (companions 2026-10-06); magic-link request accepted (200) |
| [mail/](mail/) | [dev.yaml](mail/dev.yaml) | mail-dev-body (`INSIDEOUT_MAIL_DEV_BODY`) | blank-fallback 2026-10-06 (body logging off) |

16 of 31 filled; 15 blank-fallback (config companions whose catalog or
compose default is in effect — the file exists so the kind is never
absent) plus `postgres-owner-password`/`github-token` operator-minted
empty by design. With more than one host in `hosts/`, scope the
tooling:
`INSIDEOUT_KEYS_HOST=buddys-macbook-pro-2.local python3 scripts/keys.py
init|audit|fill` (never prints values). Per-key shape tests:
`python3 -m unittest discover -s tests/keys -t .` from `scripts/`.
The root `.env` stays the only file running processes read
([docs/SETENV.md](../../../../docs/SETENV.md) is the bridge manual).
