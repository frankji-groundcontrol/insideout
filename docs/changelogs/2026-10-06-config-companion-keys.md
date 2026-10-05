# 2026-10-06 — Config-companion key kinds: every value gets the triple

**Scope:** extended the one-file-per-key scheme from 12 credential
kinds to the 6 config companions that genuinely exist as specific
choices — each now has its store file, six-part doc, and per-key test.
User call: skipping the triple for config values becomes
implementation debt. Plan:
[`docs/plans/2026-10-06-config-companion-keys.md`](../plans/2026-10-06-config-companion-keys.md).
No service/client code changed beyond the keys.py placeholder fix;
nothing deploys.

## What changed

- **New key kinds (18 total):** `llm-base-url`, `llm-model`,
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
  `llm-api-key.md` and `mail-api-key.md` now cross-link their sibling
  kinds instead of inlining them; the "backed up nowhere" caveat is
  superseded. Keys README re-indexed (18 rows) with the scope rule
  stated: pure-default knobs (ADDR, TTLs, CORS, ports) stay
  catalog-only, their stated location the `environment.md` rows.
- **Six per-key tests** mirroring the new kinds (schema enum, absolute
  URLs, sender shape, and a strong pointer test asserting the `_FILE`
  target exists). Suite: **31 tests, OK on Buddy** (2 by-design skips).
- **`scripts/keys.py` placeholder-detector fix:** `<.*>` flagged any
  brackets, so a legitimate display-name sender (`Name <addr@host>`)
  counted as a placeholder — fill skipped `INSIDEOUT_MAIL_FROM` while
  `env.sh check` called it set (the tools disagreed). Regex narrowed
  to bracket tokens without `@` (`<[^@<>]*>`), test-first
  (`test_env_state_displayNameSender_is_not_a_placeholder`). This also
  surfaced that Buddy's `.env` had a pre-existing placeholder
  `MAIL_FROM` line that blocked yesterday's mirror — replaced with the
  primary host's real line (no-echo) and filled.

## Verification

- `python3 -m unittest discover -s tests/keys -t .` (scripts/): 31
  tests OK on Buddy; cross-host run on MBP after sync (see below).
- Scoped audits: Buddy 16/18 filled, 0 unmapped; `env.sh check` green
  after the MAIL_FROM replacement (the sender no longer rides the
  sandbox default).

## Operators / users

Nothing to do for deployments (companion values already ride the
provider variable store). Anyone with a checkout gets 18 audited kinds
and recovery-from-store for the full LLM quartet and mail trio.
