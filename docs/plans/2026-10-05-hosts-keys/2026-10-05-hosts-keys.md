# 2026-10-05 — Host-keys scheme inside insideout

**Status:** Finished 2026-10-05 (CI green on `4b1ff89`; Resend key rotated and all three homes synced same day). **References:** the agents_home repo's
`hosts/<host>/keys/` scheme and the setup-piper repo's grouped manifests
(`secrets: name → {var, kind}` YAMLs + shared `.secrets/`). Per operator
direction: **all credentials under one `keys/` folder per host, module
subfolders inside it, YAML manifests within modules (split or integrated
as the module warrants), `.secrets/` holding the key files** — never one
folder per key. This plan brings that scheme **into this repository** —
both references are pattern sources only, nothing couples the repos.

**Goal (observable):** `hosts/franks-macbook-pro.local/keys/` holds five
module folders (auth, database, github, llm, mail) with eight tracked
YAML manifests naming all 12 insideout credential kinds; the 10 known
values sit in git-ignored 0600 `.key` files under the owning module's
`.secrets/`; `scripts/keys.py audit|fill` manages them without ever
printing a value; `scripts/test_keys.py` runs in CI.

## Checklist

- [x] `hosts/README.md` (scheme), `hosts/franks-macbook-pro.local/`
      (`host.yml` metadata + README), `keys/README.md` module index;
      module folders with manifests: auth/jwt, database/supabase +
      database/local-stack, github/app + github/webhooks + github/tokens,
      llm/provider, mail/resend; `.gitignore` gains `.secrets/`.
- [x] Move the 10 filled `.key` values into the module `.secrets/` dirs
      (`mv`, values never displayed); two kinds stay empty by design
      (owner compose password, optional PAT).
- [x] `scripts/keys.py` — `key_map` parses `keys/*/*.yaml` manifests
      (block and flow entry styles), `audit` (filled/empty, optional
      env-file comparison, no values), `fill` (empty-only `.key` writes
      from an env file, 0600, no values) — plus `scripts/test_keys.py`
      (7 tests incl. multi-manifest and multi-module fixtures).
- [x] CI: add `test_keys` to the env-suites job.
- [x] Docs: SETENV.md pointer to the key store, dated changelog, board row;
      `python3 -m unittest` 47/47, committed `4b1ff89` (explicit staging),
      pushed, CI green.

## Decisions

- The live consumer stays `.env` (`env.sh`/`dev.sh`/compose). The hosts
  tree is the credential store/backup layer; `keys.py fill` is the bridge
  env→keys, and `env.sh init` remains the bridge keys→env (human paste).
- Modules: `auth` (JWT), `database` (Supabase DSNs; separate manifest for
  the disposable local-stack compose passwords — split because they are
  different installations), `github` (App identity / webhook verification
  / operator PAT as separate manifests), `llm` (provider key),
  `mail` (Resend).
- Kinds are the 12 from the 2026-10-05 inventory (two DSNs, JWT, LLM,
  Resend, GH webhook/App id/PEM/client-secret, two compose passwords,
  optional PAT). `ANTHROPIC_AUTH_TOKEN` stays unmapped (config_test pins
  it as ignored legacy).
- An earlier iteration of this plan used one folder per key kind; the
  operator rejected it (see reference note above) and the tree was
  restructured before anything was committed.
- No service code changes → nothing to deploy; CI only exercises the new
  script test.
