# 2026-10-05 — Buddy host key store (key-by-key, jwt first)

**Scope:** bring `buddys-macbook-pro-2.local` (the secondary dev
workstation) into the per-host key scheme as a second host, one key at a
time following the six-part convention (where / propagated / set /
tested / used / install guide) that the agents_home family doc
established. Hosts tree + records only; no application code, nothing
deployed. Plan:
[`docs/plans/2026-10-05-buddy-host-keys.md`](../plans/2026-10-05-buddy-host-keys.md).

## What changed (slice 1 — jwt)

- **`hosts/buddys-macbook-pro-2.local/`** — new host folder: `host.yml`
  (metadata only; name and `ssh_alias: MBP2` match agents_home's record
  for the same machine), `README.md`, `keys/README.md` (live state
  table), and the first manifest `keys/auth/jwt.yaml`. Manifests land
  key-by-key as each key is actually set here, so the tracked tree
  always reflects verified state.
- **Value filled** from the local `.env` with the sanctioned tool,
  scoped to the new host: `INSIDEOUT_KEYS_HOST=buddys-macbook-pro-2.local
  python3 scripts/keys.py fill` — wrote `.secrets/jwt-secret.key`
  (dir 0700, file 0600, values never printed). The secret was
  openssl-minted by the user earlier this session; per-environment by
  design.
- **`hosts/README.md`** — second host row + the multi-host note:
  audit/fill need `INSIDEOUT_KEYS_HOST` once two hosts exist.

## Verification

- Scoped fill printed `1 written, 0 kept, 0 skipped`; scoped audit shows
  `INSIDEOUT_JWT_SECRET key=filled env=set`; file is 64 B at mode 0600
  inside a 0700 dir — consistent with `openssl rand -hex 32`.
- Isolation proven: `INSIDEOUT_KEYS_HOST=franks-macbook-pro.local audit`
  still reports every MBP key `missing` on this machine (its values live
  on that machine); the scoped fill cannot touch another host's paths.
- `git status` shows only the tracked manifest/readme files — the
  `.secrets/` dotfolder is git-ignored as designed.

## What changed (slice 2 — usage installation part)

- **`docs/usage/keys.md`** — the six-part reference for the key store
  (where / propagated / set / tested / used / install guide), mirroring
  agents_home's `docs/usage/keys/` convention; one family doc rather
  than a file per key because every key here shares one flow. Includes
  the 12-kind table with value sources and per-host state, the scoped
  audit/fill loop, per-key test probes, and boot-time usage roles.
- Indexed as a row in `docs/usage/README.md`; cross-linked from the
  `docs/SETENV.md` hosts paragraph (install guide ↔ store reference).

## What changed (slice 3 — practice record)

- **`docs/practices/2026-10-05-setting-up-a-key.md`** — the repeatable
  full sequence for landing any key on any host, distilled from the jwt
  slice: preconditions, manifest-first, `env.sh` as the value bridge,
  scoped `keys.py fill`, the four-point verification (audit / perms /
  cross-host isolation / git-blind check), the per-key probe, records,
  and the failure signals (skip-when-placeholder, the multi-host
  refusal that prevents value mixing, git-visible `.secrets`). Indexed
  in `docs/practices/README.md`. Updated later the same day
  (`5fe22eb`) to reflect the final doc scheme: per-key doc path
  `usage/keys/<module>/<key-name>.md`, doc-creation step in Records,
  and a shape failure signal (doc missing/lumped/misnamed) linking the
  learning record. Refreshed again after the testing slice
  (`e07773d`): the shape suite moved into Verify with a cross-host
  rerun rule, Test-the-key gained the warning-absence rule and the
  bad-login/garbage-bearer probes, step 3 states copy-is-a-
  transformation for dual-form values, and two failure signals were
  added (boot Warn ≠ pass; suite failing only on the other host).

## What changed (slice 4 — Postgres DSNs + supabase→postgres rename)

- **`database/postgres.yaml` on both hosts** — the manifest was
  `supabase.yaml`; renamed because the *kind* is the role-scoped
  Postgres DSNs while Supabase is merely today's host of the instance
  (user call, 2026-10-05). Values unaffected: `.key` files live in the
  module's shared `.secrets/`, keyed by name, not by manifest filename
  — MBP's filled values survive the rename untouched.
- **Buddy: both DSNs filled** from the local `.env` via the scoped
  `keys.py fill` (`2 written, 1 kept` — the kept one is jwt,
  empty-only semantics observed). Files 161/163 B at 0600 — the same
  byte sizes MBP's store records for the same DSNs.
- **Docs surfaces updated**: both hosts' `keys/README.md` tables
  (postgres.yaml row; Buddy state row), `docs/usage/keys.md` DSN rows
  (source now "managed Postgres DSN (Supabase today)"; Buddy column
  filled), plan checklist.

## Verification (slice 4)

- Scoped audit after the rename: all 3 mapped kinds `key=filled
  env=set` (the glob reads `database/postgres.yaml` fine); git status
  shows no `.secrets` paths.
- `./scripts/env.sh check`: 4 ok, 0 warnings, 0 failures.
- Owner-DSN live probe:
  `./scripts/dev.sh -C server go run ./cmd/insideout migrate` →
  `migrate: nothing to do, already up to date` — real connection,
  real authentication, currency confirmed.

## What changed (slice 5 — usage docs split per key)

- **`docs/usage/keys.md` → `docs/usage/keys/`** — slice 2's single
  family doc was the exception in agents_home's convention, not the
  rule; the user called it out (couldn't find a standalone file for a
  key). Now one standalone six-part `.md` per manifest/category —
  `jwt`, `postgres`, `local-stack`, `gh-app`, `gh-webhooks`,
  `github-token`, `llm-provider`, `mail-resend` — mirroring the store's
  manifests 1:1, plus a folder README holding the index, the shared
  set/fill/audit flow (not repeated per file), and the per-host state
  table.
- Live pointers repointed (usage README row, SETENV cross-link, the
  practice doc's four references); slice 2's changelog text stays as
  the historical record of what existed then.

## What changed (slice 6 — store-mirroring layout + reference direction)

- **`docs/usage/keys/` restructured to mirror the store** (user call):
  module folders `auth/ database/ github/ llm/ mail/`, filename = the
  manifest's `setting` — `github/app.md` replaces the flat
  `gh-app.md`, etc. One README indexes it exactly the way the hosts
  keys READMEs index manifests (module-grouped table + shared flow +
  per-host state).
- **Reference direction inverted**: per-key files no longer route
  readers into SETENV.md — each part 6 now states "this file is the
  detail" with peer links (environment.md rows, consumer walkthroughs)
  only; [SETENV.md](../SETENV.md) Steps 1/2/3/5 link **down** into the
  per-key files as the per-key authority. Also corrected while at it:
  the flat files' part-6 links were one directory too deep (pointed at
  a nonexistent root-level SETENV.md), and mail/resend wrongly claimed
  a SETENV step (mail has none).
- Verification: relative-link check across all 12 affected files —
  0 broken links.

## What changed (slice 7 — filenames = key names, one file per key)

- **12 per-key files, filename = the key name** (user call, matching
  the `.key` files exactly): singles renamed (`auth/jwt-secret.md`,
  `github/gh-webhook-secret.md`, `github/github-token.md`,
  `llm/llm-api-key.md`, `mail/mail-api-key.md`); the three multi-key
  manifest docs split — `database/postgres.md` →
  `database-app-dsn.md` + `database-owner-dsn.md`,
  `local-stack.md` → the two password files, `github/app.md` →
  `gh-app-id.md` + `gh-app-private-key.md` + `gh-client-secret.md` —
  each standalone with a six-part structure and a sibling cross-link.
- keys/README table now one row per key (module | file | env var |
  manifest); SETENV Step 1/2/3/5 down-links retargeted to the
  per-key files.
- Verification: relative-link check across 16 files — 0 broken.

## What changed (slice 8 — LLM provider key)

- **`llm/provider.yaml` on Buddy** + fill from the local `.env` (`1
  written, 3 kept`; 67 B — the same size MBP's store records for this
  key). Store now 4/12 on Buddy; the only unmapped env secret left is
  `DASHSCOPE_API_KEY` (another tool's key, deliberately unmapped).
- Verification: scoped audit `key=filled env=set` ×4; boot probe —
  config and coach init completed with **no** offline-template notice
  (it logs before the listen attempt), proving the key is read and
  accepted. The listen itself failed on `:8080`, held by an unrelated
  `api` process on this machine — irrelevant to the probe; a local
  server here needs `INSIDEOUT_ADDR` on a free port.
- State tables updated (Buddy keys README, usage keys README).

## What changed (slices 9–10 — store completed + per-key test suite)

**Slice 9 — the remaining keys landed on Buddy (10/12; +2 by design):**
mail, gh-app-id, gh-app-private-key, gh-client-secret, gh-webhook-secret,
postgres-app-password copied from MBP's agents_home store via scp
(values never displayed); manifests created (mail/resend, github/app,
github/webhooks, github/tokens, database/local-stack); `.env` mirrored
no-echo from MBP for the six values + `INSIDEOUT_MAIL_FROM` +
`INSIDEOUT_PUBLIC_URL` + `INSIDEOUT_GH_PRIVATE_KEY_FILE` (PEM consumed
from the store, not embedded). `env.sh check` green; scoped audit 10
filled / 2 by-design empty.

**Slice 10 — the testing, as unit suites organized like the keys**
(user call): `scripts/tests/keys/` mirrors the store and the usage
docs — module folders, one file per key, filename = key name
(hyphens included; a `load_tests` hook in the package `__init__`
loads them by path because Python cannot import hyphenated module
names — and replaces plain discovery, which silently skips them).
`test_tool.py` (keys.py behaviors + multi-host scoping) and
`test_store.py` (real-tree invariants: 0600/0700, non-empty unless
operator-filled, git-blindness, host.yml naming; CI-safe) sit at the
root with a README index. `scripts/test_keys.py` retired; CI
env-suites now discovers the new tree. 24 tests, 22 pass + 2 by-design
skips on Buddy.

**Real defects the tests surfaced (and their fixes):**
- The copied PEM was the `\n`-escaped single-line env form.
  `github.LoadPrivateKey` reads the `_FILE` variant raw, so boot only
  **warned** — "private key unusable — guide loads will be
  unauthenticated" — a silent degradation my earlier boot checks
  counted as a pass. Normalized the `.key` to true multi-line PEM
  (1702→1675 B); re-boot shows the warning gone. Note for the other
  repo: MBP's agents_home store still holds the escaped form — fine
  while MBP embeds the env var, normalize before ever pointing
  `INSIDEOUT_GH_PRIVATE_KEY_FILE` at it.
- The DSNs are pooler-form (`insideout_app.<project-ref>` userinfo,
  `pgbouncer=true` setups) — the per-key tests now encode that shape
  instead of assuming bare-role DSNs.

**Live round-trips completed (part 4 of every landed key):**
`/api/v1/me` no-token → 401; garbage bearer → 401 (JWT verification
path); bad login → 401 (app-DSN query under `insideout_app` against
the real database); `migrate` → up to date (owner DSN); boot without
the offline-coach notice (LLM); magic-link request → 200 (mail key
accepted; inbox delivery remains the human check); boot without the
private-key warning (GitHub trio). Probes ran on `:8099` — `:8080` is
held by an unrelated process on this machine (`INSIDEOUT_ADDR` to run
a local server here).

## Cross-host proof (closing the loop)

The suite ran on MBP as the second host and caught **three real
MBP-local issues**, all fixed in place: (1) its store copy of the App
PEM was also the escaped single-line form — normalized (1702→1675 B;
its `.env` keeps the env-embedded escaped form, which is that lane's
correct format); (2) its `keys/*/.secrets` dirs were 0755 — chmod 700;
(3) on MBP's case-insensitive filesystem the buddy host folder existed
on disk as `Buddys-MacBook-Pro-2.local` while git tracks lowercase —
two-step renamed to match. Plus one test fix valid everywhere: the
multi-host CLI invocations interleaved `--root` between two
positionals, which Python ≤3.9 argparse rejects (`81a9935`). Final
state: **24 tests, OK on both hosts** (Buddy skips the two by-design
keys; MBP runs them against its sanctioned empty placeholders).

## What changed (slice 11 — usage docs surface the testing)

- **`docs/usage/keys/README.md`** gained a dedicated **Testing**
  section: the two required layers — the automated shape/placement
  suite (`python3 -m unittest discover -s tests/keys -t .` from
  `scripts/`, what it asserts, when to run it, run it on every host)
  and each key's live probe (part 4), with the boot-warning-is-not-a-
  pass rule spelled out. The duplicate one-liner after the state table
  was folded into it.
- **`docs/usage/local-development.md`** Testing section now covers the
  Python suites alongside Go/Flutter: the env batteries and the key
  suites (same two batteries CI's env-suites job runs), what the key
  suites cover, and the suite-vs-probe relationship.
- Usage README keys row mentions the two-layer testing. Link-check
  across the touched surfaces: 0 broken.

## What changed (slice 12 — llm six-part completed by evidence)

- **`docs/usage/keys/llm/llm-api-key.md`** parts 3–4 rewritten from
  the session's live findings: part 3 now states the key is one of
  **four values that must agree** (key + base URL + model + schema,
  with the wire-format meanings), and part 4 became a three-rung
  testing ladder — notice-absent (key read) → authenticated
  provider ping, status-only (key+endpoint+format+model agree;
  verified live: 200 against the relay) → one real coach reply —
  plus the tunnel caveat (a `000` on a relay behind ngrok means
  retry, not a bad key; observed as a transient drop this session).
  Six-part coverage for the llm module was otherwise complete; the
  companions are catalog config, deliberately not store keys.

## Operators / users

Nothing to do elsewhere. Remaining keys land in later slices per the
plan checklist (DSNs + LLM are env-set locally; mail and the GitHub
keys copy from the filled store on MBP; compose passwords mint on
next use; the PAT stays optional-empty).
