# 2026-10-06 — Config-companion key kinds (every value gets the triple)

**Goal:** extend the one-file-per-key scheme from 12 credential kinds to
every mapped env var, in two phases. Phase 1: the 6 config companions
that "genuinely exist" as specific choices (`llm-base-url`,
`llm-model`, `llm-schema`, `mail-from`, `public-url`,
`gh-private-key-file`). Phase 2 (user instruction, same day): full scan
of the env catalog — the **never-absent rule**: a kind may be *blank*
(a catalog or compose default is in effect) but never *missing*; every
mapped kind gets its store file, six-part doc, and per-key test. Each
phase: store file (manifest entry + `.key`), doc under
`docs/usage/keys/<module>/`, test under `scripts/tests/keys/<module>/`.

**Why:** user call 2026-10-06 — skipping the triple for config values
becomes implementation debt; the LLM quartet proved a key without its
companions is untestable and unrecoverable (overrides existed only in
`.env`). agents_home's convention (model ids and endpoints as key kinds)
is the reference — ported structurally this time. Phase 2's never-absent
rule supersedes this plan's original "pure-default knobs stay
catalog-only" scoping (user: "just because they can be set in the
config doesn't mean you should leave them out… do a full scan").

**Scope:** manifests (both hosts), store fills + `init` placeholders,
13+6 docs, 13+6 tests, index tables, cross-link updates, records. No
Go/client code, nothing deploys.

## Checklist

### Phase 1 — the 6 companions

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
- [x] MBP manifests extended with the same 6 companion entries (its
      fill had written 0 — entries only existed in Buddy's manifests).

### Phase 2 — full scan, never-absent rule (31 kinds)

- [x] Full scan of `.env.example` + `server/internal/config/config.go`
      readers + docker-compose interpolation: 13 missing kinds (9
      runtime: addr, cookie-secure, cors-origins, dev-cors, access-ttl,
      refresh-ttl, ai-max-concurrent, ai-queue-wait, mail-dev-body;
      4 compose: server-port, postgres-port,
      postgres-superuser-password, client-port).
- [x] `keys.py init` test-first: creates empty 0600 placeholders for
      every mapped kind, never touches a filled file, idempotent.
- [x] Store invariants strengthened: a present host must have every
      mapped kind's `.key` (never-absent); empty is legal only with a
      manifest `fallback`/`optional`/`operator_fills` field.
- [x] Four new manifests on BOTH hosts with `fallback:` fields:
      `app/runtime.yaml` (6), `llm/coach.yaml` (2), `mail/dev.yaml`
      (1), `compose/stack.yaml` (4).
- [x] Buddy `init` + scoped `fill` (13 placeholders created; env sets
      none of them, so all stay blank-fallback — correct).
- [x] 13 six-part docs (fallback-first framing); usage keys README
      rewritten (31 rows, never-absent rule stated, superseded
      "catalog-only" sentence removed); tests README re-indexed;
      both host READMEs updated.
- [x] 13 per-key tests (exists + shape-if-non-empty). Suite: 46 tests
      OK on Buddy.
- [x] Push + MBP sync; MBP init (19 placeholders: 13 new + 6
      companions) + fill (6 written: access-ttl, refresh-ttl,
      mail-from, public-url, server-port, postgres-port — this host's
      env overrides); suite **46 OK on MBP** after marking its four
      blank-lane companion entries `optional: true` (offline coach +
      env-embedded PEM lane) and making the phase-1 shape tests
      blank-tolerant (fix commit `40635cf`).
- [x] Records: changelog `2026-10-06-config-companion-keys.md`
      extended with phase 2; HANDOFF state line; board row closed
      after the MBP run.

## Decisions

- Config companions ride the SAME store mechanics (`.secrets/` 0600,
  no-print fill) — like agents_home's model-id keys. They are not
  secrets; the store is their out-of-git backup, fixing the
  recovery hole documented 2026-10-05.
- `gh-private-key-file` uses `kind: path`; its test asserts the target
  file exists — mapping it also silences the audit's permanent
  unmapped flag.
- Never-absent (phase 2): every mapped kind gets an empty `.key`
  placeholder when its default is in effect — the file is the
  findability contract; `fallback:` in the manifest is what makes a
  legal blank explicit and testable. This supersedes the original
  "catalog-only knobs" scoping.
- Compose ports/passwords live in a `compose/` module (they configure
  docker-compose interpolation, not the server process); the
  bootstrap-only superuser password is documented as a local-dev
  default, not a production credential.
