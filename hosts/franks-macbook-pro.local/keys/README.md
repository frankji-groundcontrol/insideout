# Keys — insideout credentials on this host

Everything lives under this one `keys/` folder: subfolders by functional
module, YAML manifests inside them (a module may split its configuration
across several YAMLs or keep it integrated in one), and each module's
`.secrets/` holding the actual `<name>.key` files (mode 0600,
git-ignored, `ls -a` to see them). Manifests are metadata only — values
never appear in tracked files.

| Module | Manifest | Keys (env var) |
| --- | --- | --- |
| [auth/](auth/) | [jwt.yaml](auth/jwt.yaml) | jwt-secret (`INSIDEOUT_JWT_SECRET`) |
| [database/](database/) | [supabase.yaml](database/supabase.yaml) | database-app-dsn (`DATABASE_URL`), database-owner-dsn (`DATABASE_OWNER_URL`) |
| [database/](database/) | [local-stack.yaml](database/local-stack.yaml) | postgres-app-password, postgres-owner-password (`POSTGRES_*_PASSWORD`) |
| [github/](github/) | [app.yaml](github/app.yaml) | gh-app-id, gh-app-private-key, gh-client-secret (`INSIDEOUT_GH_*`) |
| [github/](github/) | [webhooks.yaml](github/webhooks.yaml) | gh-webhook-secret (`INSIDEOUT_GH_WEBHOOK_SECRET`) |
| [github/](github/) | [tokens.yaml](github/tokens.yaml) | github-token (`GITHUB_TOKEN`, optional) |
| [llm/](llm/) | [provider.yaml](llm/provider.yaml) | llm-api-key (`INSIDEOUT_LLM_API_KEY`) |
| [mail/](mail/) | [resend.yaml](mail/resend.yaml) | mail-api-key (`INSIDEOUT_MAIL_API_KEY`) |

10 of 12 keys are filled; `postgres-owner-password` and `github-token`
are operator-minted and currently empty. Audit and fill:
`python3 scripts/keys.py audit|fill` from the repo root (never prints
values).

The root `.env` stays the only file running processes read; this tree is
the out-of-git store behind it (`scripts/env.sh init` is the bridge —
see [docs/SETENV.md](../../../../docs/SETENV.md)).
