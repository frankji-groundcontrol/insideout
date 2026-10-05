# 2026-10-05 — Buddy host key store (key-by-key)

**Goal:** bring this machine (`buddys-macbook-pro-2.local`) into the
per-host key scheme as a first-class host, one key at a time, following
the six-part convention (where / propagated / set / tested / used /
install guide) the agents_home family doc established. The user drives
the order; each key lands with its manifest + `.secrets/` value and this
checklist tracks progress.

**Scope:** hosts tree, its READMEs, and records only. No application
code; `keys.py` already supports multi-host via `INSIDEOUT_KEYS_HOST`.
Nothing deploys.

## Checklist

- [x] Host scaffold: `hosts/buddys-macbook-pro-2.local/` (host.yml,
      README.md, keys/README.md) + row in `hosts/README.md`.
- [x] Usage installation part: `docs/usage/keys.md` — the six-part
      reference (mirrors agents_home's `docs/usage/keys/` convention,
      one family doc since every key shares one flow) with the per-host
      state table; indexed in `docs/usage/README.md`; cross-linked from
      `docs/SETENV.md`.
- [x] Practice record: `docs/practices/2026-10-05-setting-up-a-key.md`
      — the full per-key setup sequence (trigger / six-part frame /
      steps / verification / failure signals), indexed in the practices
      README.
- [x] Usage docs split per key (user call, superseding slice 2's
      single family doc): `docs/usage/keys.md` → `docs/usage/keys/`
      with a README index (shared flow + per-host state table) and one
      standalone six-part file per manifest/category (jwt, postgres,
      local-stack, gh-app, gh-webhooks, github-token, llm-provider,
      mail-resend). Live pointers updated (usage README, SETENV,
      practice doc); slice-2 changelog text left as history.
- [x] `keys/auth/jwt.yaml` manifest; value filled from the local `.env`
      via `INSIDEOUT_KEYS_HOST=buddys-macbook-pro-2.local python3
      scripts/keys.py fill` (empty-only, 0600, never printed). User
      minted the secret with openssl.
- [x] database/postgres (renamed from supabase 2026-10-05 — the kind is
      Postgres DSNs, Supabase is just today's host): both DSNs filled
      from the local `.env`; manifest renamed on both hosts, values
      unaffected (module-shared `.secrets/`, keyed by name).
- [ ] llm: provider key (env already set).
- [ ] mail: Resend key (missing locally — copy from MBP's agents_home
      store when user calls it).
- [ ] github: app id + PEM + client secret + webhook secret (missing
      locally — copy from MBP's agents_home store).
- [ ] database/local-stack: two compose passwords (mint when the
      disposable stack is next used).
- [ ] github/tokens: optional PAT (stays empty by default).
- [ ] Close out: keys/README.md state table final, changelog, HANDOFF,
      commit + push + MBP fast-forward.

## Decisions

- Folder name `buddys-macbook-pro-2.local` matches agents_home's host
  for the same machine; `ssh_alias: MBP2` likewise.
- Manifests land key-by-key with the user (not front-loaded), so the
  tracked tree always reflects verified state; the store's six parts
  are satisfied per key by: manifest+`.key` (1), `dev.sh`/`env.sh`
  bridge (2), `keys.py fill` + `env.sh edit` (3), `env.sh check` +
  boot/migrate probes (4), `server/internal/config` (5), SETENV.md (6).
- `hosts/README.md` gains the multi-host note: audit/fill need
  `INSIDEOUT_KEYS_HOST` once two hosts exist.
