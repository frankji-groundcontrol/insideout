# 2026-10-05 — Host key store (hosts/) with audit/fill tooling

**Scope:** credential management for this repository, following the
agents_home + setup-piper reference schemes (pattern only — no coupling).
Docs, scripts, and the new `hosts/` tree; no service code changed,
nothing to deploy. Plan:
[`docs/plans/2026-10-05-hosts-keys/`](../plans/2026-10-05-hosts-keys/2026-10-05-hosts-keys.md).

## What changed

- **`hosts/franks-macbook-pro.local/keys/`** — one `keys/` root holding
  five functional-module folders, each with tracked YAML manifests (split
  or integrated per module) naming all 12 credential kinds —
  `secrets: <name>: {var, kind: value|path, ...}` — beside the module's
  git-ignored `.secrets/<name>.key` value files (mode 0600):
  - `auth/jwt.yaml` — JWT secret.
  - `database/supabase.yaml` — app + owner DSNs; `database/local-stack.yaml`
    — the disposable compose stack's two postgres passwords (split because
    they belong to a different installation).

    *Superseded 2026-10-05 (same day): the manifest was renamed
    `postgres.yaml` — the kind is the role-scoped Postgres DSNs, and Supabase
    is merely today's host ([buddy host keys](2026-10-05-buddy-host-keys.md),
    slice 4).*
  - `github/app.yaml` — App id + private key + client secret;
    `github/webhooks.yaml` — webhook secret; `github/tokens.yaml` —
    optional operator PAT.
  - `llm/provider.yaml` — LLM provider key; `mail/resend.yaml` — Resend key.
  `hosts/README.md`, per-host README + `host.yml`, and a keys README
  module index document the scheme.
- **Values:** 10 of 12 keys filled by moving the curated `.key` files
  into the module `.secrets/` dirs (`mv`/`chmod 600`, values never
  displayed). Empty by design: `postgres-owner-password` (unset locally;
  bundled-compose only) and `github-token` (optional PAT).
  `.gitignore` gains `.secrets/`; `git ls-files hosts/` shows metadata only.
- **`scripts/keys.py`** — `key_map` walks `hosts/<host>/keys/*/*.yaml`
  (block and flow entry styles), `audit [env-file]` (per-key filled/empty,
  env comparison, unmapped secret-shaped env names) and `fill <env-file>`
  (empty-only writes at 0600). Never prints values.
  **`scripts/test_keys.py`** — 7 unit tests: manifest parsing across
  files and modules, no-value-leak guarantees, empty-only fills,
  placeholder/missing skips, unmapped detection, CLI end-to-end. CI
  env-suites job now runs it
  (`test_env_catalog test_env_writes test_keys`).

  *Superseded 2026-10-05 (same day): `scripts/test_keys.py` was retired for
  the `scripts/tests/keys/` tree, which CI's env-suites job discovers instead
  ([buddy host keys](2026-10-05-buddy-host-keys.md), slice 10).*
- **`docs/SETENV.md`** — pointer paragraph: `.env` stays the only file
  processes read; the hosts tree is the out-of-git credential store.

## Verification

- `python3 -m unittest test_env_catalog test_env_writes test_keys` →
  47 passed, 0 failed.
- `python3 scripts/keys.py audit` → 12 kinds across 5 modules,
  10 filled / 2 empty, paths resolving to `keys/<module>/.secrets/*.key`.
- No consumer changed: `.env` untouched, `env.sh check` green, server and
  client suites not affected (no Go/Dart changes).

## Operator notes

- Visibility: each module's `.secrets` is a dotfolder —
  `ls -a hosts/<host>/keys/<module>/` or `find hosts -name '*.key'`
  (documented in the keys README).
- Rotation still means updating `.env` and the Railway `server` variables
  to match; the store is the backup layer. The Resend key **was rotated
  2026-10-05** (same day as this change): `.env` and Railway already held
  the new value, and the store's `mail/resend.yaml` key was synced from
  `.env` (comparison and write without printing the value); live
  `/healthz` answered 200 after.
