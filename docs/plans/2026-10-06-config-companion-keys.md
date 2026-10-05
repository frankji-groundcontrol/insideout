# 2026-10-06 — Config-companion key kinds (every value gets the triple)

**Goal:** extend the one-file-per-key scheme from 12 credential kinds to
the 6 config companions that "genuinely exist" as specific choices:
`llm-base-url`, `llm-model`, `llm-schema`, `mail-from`, `public-url`,
`gh-private-key-file`. Each gets its store file (manifest entry + filled
`.key`), its six-part doc under `docs/usage/keys/<module>/`, and its
per-key test under `scripts/tests/keys/<module>/`. Pure-default knobs
(ADDR, TTLs, CORS, ports) stay catalog-only with their location stated
in `environment.md` — nothing in between.

**Why:** user call 2026-10-06 — skipping the triple for config values
becomes implementation debt; the LLM quartet proved a key without its
companions is untestable and unrecoverable (overrides existed only in
`.env`). agents_home's convention (model ids and endpoints as key kinds)
is the reference — ported structurally this time.

**Scope:** manifests (both hosts), store fills, 6 docs, 6 tests, index
tables, cross-link updates, records. No Go/client code, nothing deploys.

## Checklist

- [x] Buddy manifests extended: `llm/provider.yaml` (+3), `mail/resend.yaml`
      (+2), `github/app.yaml` (+1, kind: path); scoped fill writes 6
      (7 with the MAIL_FROM fix after the placeholder-detector repair).
- [x] Six docs: `llm/{llm-base-url,llm-model,llm-schema}.md`,
      `mail/{mail-from,public-url}.md`, `github/gh-private-key-file.md`.
- [x] Six tests mirroring them; suite green on Buddy (31 tests).
- [x] READMEs re-indexed (usage keys, tests tree, Buddy host); `llm-api-key.md`
      and `mail-api-key.md` cross-updated to sibling docs (yesterday's
      "backed up nowhere" caveat superseded).
- [x] keys.py placeholder detector fixed test-first (display-name
      senders are not placeholders); Buddy's stale MAIL_FROM
      placeholder line replaced with the primary host's real value.
- [ ] Push + MBP sync; MBP fill for its env-present companions; suite
      green on MBP.
- [x] Records: changelog `2026-10-06-config-companion-keys.md` + index;
      HANDOFF state line; board row closed after the MBP run.

## Decisions

- Config companions ride the SAME store mechanics (`.secrets/` 0600,
  no-print fill) — like agents_home's model-id keys. They are not
  secrets; the store is their out-of-git backup, fixing the
  recovery hole documented 2026-10-05.
- `gh-private-key-file` uses `kind: path`; its test asserts the target
  file exists — mapping it also silences the audit's permanent
  unmapped flag.
- Catalog-only knobs keep `environment.md` as their stated location;
  this plan does not create files for them.
