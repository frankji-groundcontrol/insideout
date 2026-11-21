# Keys — insideout credentials on this host

This host's values live under this `keys/` folder: one subfolder per
functional module, each holding a `.secrets/` with the actual
`<name>.key` files (mode 0600, git-ignored, `ls -a` to see them). The
YAML manifests that name every key are shared by all hosts, under
[`hosts/_manifests/keys/`](../../_manifests/keys/) (a module may split
its configuration across several YAMLs or keep it integrated in one).
Manifests are metadata only — values never appear in tracked files. A mapped kind may be blank (its catalog
or compose default is in effect) but never absent: `keys.py init`
creates the empty placeholders.

| Module | Manifest | Keys (env var) | State |
| --- | --- | --- | --- |
| app/ | [runtime.yaml](../../_manifests/keys/app/runtime.yaml) | addr, cookie-secure, cors-origins, dev-cors, access-ttl, refresh-ttl (`INSIDEOUT_ADDR` … `INSIDEOUT_REFRESH_TTL`) | access-ttl/refresh-ttl filled 2026-10-06 (this host's env overrides); the rest blank-fallback |
| auth/ | [jwt.yaml](../../_manifests/keys/auth/jwt.yaml) | jwt-secret (`INSIDEOUT_JWT_SECRET`) | filled |
| compose/ | [stack.yaml](../../_manifests/keys/compose/stack.yaml) | server-port, postgres-port, postgres-superuser-password, client-port (`SERVER_PORT` … `CLIENT_PORT`) | server-port/postgres-port filled 2026-10-06 (this host's env overrides); the other two blank-fallback |
| database/ | [postgres.yaml](../../_manifests/keys/database/postgres.yaml) | database-app-dsn (`DATABASE_URL`), database-owner-dsn (`DATABASE_OWNER_URL`) | both filled |
| database/ | [local-stack.yaml](../../_manifests/keys/database/local-stack.yaml) | postgres-app-password, postgres-owner-password (`POSTGRES_*_PASSWORD`) | app filled; owner empty by design |
| github/ | [app.yaml](../../_manifests/keys/github/app.yaml) | gh-app-id, gh-app-private-key, gh-client-secret, gh-private-key-file (`INSIDEOUT_GH_*` + the `_FILE` pointer, kind: path) | first three filled; pointer blank by design (this host uses the env-embedded lane) |
| github/ | [webhooks.yaml](../../_manifests/keys/github/webhooks.yaml) | gh-webhook-secret (`INSIDEOUT_GH_WEBHOOK_SECRET`) | filled |
| github/ | [tokens.yaml](../../_manifests/keys/github/tokens.yaml) | github-token (`GITHUB_TOKEN`, optional) | empty by design |
| llm/ | [provider.yaml](../../_manifests/keys/llm/provider.yaml) | llm-api-key + companions llm-base-url, llm-model, llm-schema (`INSIDEOUT_LLM_*`) | key filled; companions filled from this host's `.env` when present (offline coach otherwise) |
| llm/ | [coach.yaml](../../_manifests/keys/llm/coach.yaml) | ai-max-concurrent, ai-queue-wait (`INSIDEOUT_AI_*`) | blank-fallback 2026-10-06 (defaults 4 / 15s in effect) |
| mail/ | [resend.yaml](../../_manifests/keys/mail/resend.yaml) | mail-api-key + companions mail-from, public-url (`INSIDEOUT_MAIL_*`, `INSIDEOUT_PUBLIC_URL`) | key filled; companions filled from this host's `.env` when present |
| mail/ | [dev.yaml](../../_manifests/keys/mail/dev.yaml) | mail-dev-body (`INSIDEOUT_MAIL_DEV_BODY`) | blank-fallback 2026-10-06 (body logging off) |

With more than one host in `hosts/`, scope the tooling:
`INSIDEOUT_KEYS_HOST=franks-macbook-pro.local python3 scripts/keys.py
init|audit|fill` from the repo root (never prints values). Per-key
shape tests: `python3 -m unittest discover -s tests/keys -t .` from
`scripts/`.

The root `.env` stays the only file running processes read; this tree is
the out-of-git store behind it (`scripts/env.sh init` is the bridge —
see [docs/SETENV.md](../../../docs/SETENV.md)).
