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
- [x] Restructured to mirror the store exactly + reference direction
      fixed (user call): module folders `auth/ database/ github/ llm/
      mail/` with filename = manifest `setting` (github/app.md, not
      gh-app.md); every per-key file's part 6 now reads "this file is
      the detail" with peer links only; SETENV.md Steps 1/2/3/5 link
      down into the per-key files. Link-check over all affected files:
      0 broken.
- [x] Filenames = key names, one file per key (user call): 12 files
      named exactly like their `.key` files (`jwt-secret.md`,
      `database-app-dsn.md`, … `gh-app-private-key.md`, …); the three
      multi-key manifest docs split into per-key files with sibling
      cross-links; README table 12 rows; SETENV Step 1/2/3/5 links
      retargeted. Link-check: 16 files, 0 broken.
- [x] `keys/auth/jwt.yaml` manifest; value filled from the local `.env`
      via `INSIDEOUT_KEYS_HOST=buddys-macbook-pro-2.local python3
      scripts/keys.py fill` (empty-only, 0600, never printed). User
      minted the secret with openssl.
- [x] database/postgres (renamed from supabase 2026-10-05 — the kind is
      Postgres DSNs, Supabase is just today's host): both DSNs filled
      from the local `.env`; manifest renamed on both hosts, values
      unaffected (module-shared `.secrets/`, keyed by name).
- [x] llm: provider key — filled from the local `.env` (67 B, same as
      MBP's record); boot probe: config loaded, offline-coach notice
      absent (the listen bind on :8080 failed — that port is held by
      an unrelated `api` process on this machine; use `INSIDEOUT_ADDR`
      to run a local server here).
- [x] mail: Resend key — copied from MBP's store; `.env` mirrored
      (with `INSIDEOUT_MAIL_FROM` + `INSIDEOUT_PUBLIC_URL` companions);
      magic-link request accepted (200).
- [x] github: app id + PEM + client secret + webhook secret — copied;
      `.env` mirrored; PEM consumed via `INSIDEOUT_GH_PRIVATE_KEY_FILE`.
- [x] database/local-stack: postgres-app-password copied;
      postgres-owner-password empty by design (mint on next stack use).
- [x] github/tokens: optional PAT — empty by design.
- [x] Testing as unit suites, organized like the keys (user call):
      `scripts/tests/keys/` — module folders, one file per key named by
      key name (hyphens kept; a `load_tests` hook loads them since
      Python can't import hyphenated modules), plus `test_tool.py`
      (keys.py behaviors incl. multi-host scoping) and `test_store.py`
      (real-tree invariants, CI-safe). 24 tests; CI env-suites job
      updated to run the discovery.
- [x] Real defects the tests caught and fixed: (1) the copied PEM was
      the `\n`-escaped single-line form — the `_FILE` consumer reads
      raw, so boot warned "private key unusable — guide loads will be
      unauthenticated"; normalized to true multi-line (1702→1675 B),
      warning gone on re-boot. (2) DSNs are pooler-form
      (`insideout_app.<project-ref>` userinfo) — tests now encode that
      shape instead of assuming bare-role DSNs.
- [x] Live round-trips completed: `/me` no-token → 401; garbage bearer
      → 401 (JWT verify path); bad login → 401 (app-DSN query under
      `insideout_app`); migrate → up-to-date (owner DSN); boot without
      offline-coach notice (LLM). Port note: `:8080` is held by an
      unrelated process on this machine — probes used `:8099`.
- [x] Close out: state tables final, changelog slices 9–10, HANDOFF,
      commit + push + MBP fast-forward (+ suite run on MBP as the
      cross-host proof).

## Decisions

- Folder name `buddys-macbook-pro-2.local` matches agents_home's host
  for the same machine; the manifest's ssh alias is a placeholder (rv4-PUB-10).
- Manifests land key-by-key with the user (not front-loaded), so the
  tracked tree always reflects verified state; the store's six parts
  are satisfied per key by: manifest+`.key` (1), `dev.sh`/`env.sh`
  bridge (2), `keys.py fill` + `env.sh edit` (3), `env.sh check` +
  boot/migrate probes (4), `server/internal/config` (5), SETENV.md (6).
- `hosts/README.md` gains the multi-host note: audit/fill need
  `INSIDEOUT_KEYS_HOST` once two hosts exist.
