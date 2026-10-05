# 2026-10-06 — Config-companion key kinds: every value gets the triple

**Scope:** extended the one-file-per-key scheme from 12 credential
kinds to every mapped env var, in two phases. Phase 1: the 6 config
companions that genuinely exist as specific choices. Phase 2 (same-day
user instruction): full scan of the env catalog under the
**never-absent rule** — a kind may be blank (a catalog or compose
default in effect) but never missing; each of the 31 kinds now has its
store file, six-part doc, and per-key test. User call: skipping the
triple for config values becomes implementation debt, and "just
because they can be set in the config doesn't mean you should leave
them out." Plan:
[`docs/plans/2026-10-06-config-companion-keys.md`](../plans/2026-10-06-config-companion-keys.md).
No service/client code changed beyond the keys.py fixes; nothing
deploys.

## What changed

### Phase 1 — the 6 companions (31 kinds total with phase 2)

- **New key kinds:** `llm-base-url`, `llm-model`,
  `llm-schema` (manifest `llm/provider.yaml`), `mail-from`,
  `public-url` (`mail/resend.yaml`), `gh-private-key-file`
  (`github/app.yaml`, `kind: path`). Config companions ride the same
  store mechanics (`.secrets/` 0600, no-print fill) — like agents_home
  treats model ids and endpoints — closing the recovery hole recorded
  2026-10-05 (companion overrides existed only in `.env`).
- **Buddy filled 16/18** (companions copied from its `.env` via scoped
  fill; the two empties stay by design). Mapping the `_FILE` pointer
  silenced the audit's permanent unmapped flag — Buddy and MBP audits
  now report **0 unmapped**.
- **Six six-part docs** under `docs/usage/keys/`
  (`llm/{llm-base-url,llm-model,llm-schema}.md`,
  `mail/{mail-from,public-url}.md`, `github/gh-private-key-file.md`);
  `llm-api-key.md` and `mail-api-key.md` cross-link their sibling
  kinds instead of inlining them.
- **Six per-key tests** mirroring the new kinds (schema enum, absolute
  URLs, sender shape, and a strong pointer test asserting the `_FILE`
  target exists).
- **`scripts/keys.py` placeholder-detector fix:** `<.*>` flagged any
  brackets, so a legitimate display-name sender (`Name <addr@host>`)
  counted as a placeholder — fill skipped `INSIDEOUT_MAIL_FROM` while
  `env.sh check` called it set (the tools disagreed). Regex narrowed
  to bracket tokens without `@` (`<[^@<>]*>`), test-first
  (`test_env_state_displayNameSender_is_not_a_placeholder`). This also
  surfaced that Buddy's `.env` had a pre-existing placeholder
  `MAIL_FROM` line that blocked yesterday's mirror — replaced with the
  primary host's real line (no-echo) and filled.
- **MBP's three manifests extended** with the same companion entries
  after diagnosing that its fill had written 0 — the entries only
  existed in Buddy's manifests (fill maps per host).

### Phase 2 — full scan, never-absent rule (18 → 31 kinds)

- **13 new kinds**, each with a `fallback:` manifest field making the
  legal blank explicit: `app/runtime.yaml` (addr, cookie-secure,
  cors-origins, dev-cors, access-ttl, refresh-ttl), `llm/coach.yaml`
  (ai-max-concurrent, ai-queue-wait), `mail/dev.yaml`
  (mail-dev-body), `compose/stack.yaml` (server-port, postgres-port,
  postgres-superuser-password, client-port — the docker-compose
  interpolation knobs, in their own `compose/` module).
- **`keys.py init`** (test-first): creates an empty 0600 `.key`
  placeholder for every mapped kind — never touches a filled file,
  idempotent. Buddy: 15 placeholders created (the 13 + the two
  operator-fill kinds that had no file); its `.env` sets none of the
  13, so all remain blank-fallback = defaults in effect.
- **Store invariants strengthened:** a host present on the machine
  (≥1 `.key`) must have EVERY mapped kind's file; empty is legal only
  with a manifest `fallback`/`optional`/`operator_fills` field.
- **13 six-part docs** (fallback-first framing: "blank = catalog
  default X; the file exists so the kind is always findable") +
  **13 per-key tests** (exists + shape-if-non-empty: Go durations,
  port ranges, exact `0`/`1` toggles, positive int, non-placeholder).
  Usage keys README rewritten (31 rows, never-absent rule stated; the
  phase-1 "pure-default knobs stay catalog-only" sentence superseded
  and removed); tests README + both host READMEs re-indexed.

## Verification

- `python3 -m unittest discover -s tests/keys -t .` (scripts/): 31
  tests OK after phase 1; **46 tests OK on Buddy** after phase 2
  (+13 per-key, +1 init tool, +1 never-absent invariant); cross-host
  run on MBP after sync (see plan).
- Scoped audits: Buddy 31 kinds, 0 unmapped, 16 filled + 15
  legal-blank; `env.sh check` green after the MAIL_FROM replacement.

## Operators / users

Nothing to do for deployments (companion values already ride the
provider variable store; the 13 phase-2 kinds run on defaults until
deliberately set). Anyone with a checkout gets 31 audited kinds and
recovery-from-store for the full LLM quartet and mail trio; on a new
host, `keys.py init` materializes every placeholder so no kind is
absent before values land.
