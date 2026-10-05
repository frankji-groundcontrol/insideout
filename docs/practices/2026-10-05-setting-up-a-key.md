# 2026-10-05 — Setting up a key on a host (full sequence)

*Extended 2026-10-06 to every mapped kind (config companions and
runtime knobs), the `init` placeholder step, and blank-lane legality
([changelog](../changelogs/2026-10-06-config-companion-keys.md)).*

## Trigger

Any time a kind must land on a machine in the
[`hosts/<host>/keys/`](../../hosts/README.md) store: a new credential,
a config companion or runtime knob, a new host, or a refill after
rotation. **Never-absent rule:** every kind mapped in a host's
manifests has a `.key` file on that host — blank means "default in
effect", absent means the scheme is broken. Worked examples: the
jwt-secret slice on `buddys-macbook-pro-2.local`
([changelog](../changelogs/2026-10-05-buddy-host-keys.md)); the
full-scan extension to 31 kinds
([changelog](../changelogs/2026-10-06-config-companion-keys.md)).

## The six-part frame

Every key setup satisfies the same six parts, and each key's six parts
live in its **own doc** at
`docs/usage/keys/<module>/<key-name>.md` — filename = the key name,
exactly like the `.key` file, module folder = the manifest's module
([index](../usage/keys/README.md)). The parts: **where** (manifest +
`.key`), **propagated** (`.env` stays the only consumer), **set**
(`env.sh` + scoped `fill`), **tested** (audit + the key's probe),
**used** (boot role), **install guide** (router docs link *down* to
the per-key file; the detail never routes up). The sequence below is
parts 1–4 operationalized; 5–6 live in the per-key doc, not re-derived
per setup.

## Repeatable sequence

1. **Preconditions.** The host folder exists (`host.yml`, READMEs —
   scaffold per the scheme if not). The value source is known from the
   key's own doc, `docs/usage/keys/<module>/<key-name>.md`. Never
   print a value at any step.
2. **Manifest first.** Create
   `hosts/<host>/keys/<module>/<setting>.yaml` — metadata only
   (`<name>: {var, kind, note}`); values never appear in tracked files.
   Declare blank-legality in the entry itself: a `fallback:` field
   (with the catalog/compose default) when blank = that default, or
   `optional:`/`operator_fills:` when blank is a deliberate lane on
   that host — judged by the field's **presence**, not truthiness
   (`fallback: ''` is a legal blank). A new module folder gets its row
   in the host's `keys/README.md`. The module name also fixes the
   per-key doc's folder (step 7).
3. **Value into the live consumer.** `./scripts/env.sh edit` (or
   `init`) writes the root `.env` interactively and never prints. If
   the value is already there, the scoped audit (step 5) reports
   `env=set` without revealing it. PEM exception: keep the multi-line
   key in the `.key` file and point
   `INSIDEOUT_GH_PRIVATE_KEY_FILE` at it — no single-line env bridge.
   **A copy between consumers is a transformation, not a byte copy:**
   the PEM's env form is `\n`-escaped single-line, the `_FILE`
   consumer reads the file raw — the store keeps the canonical
   multi-line form, and deriving the env form is an escape. Same
   discipline for any dual-form value.
4. **Init placeholders, then scoped fill.**
   `INSIDEOUT_KEYS_HOST=<host> python3 scripts/keys.py init` — an empty
   0600 `.key` placeholder for **every** mapped kind (idempotent;
   never touches a filled file), so no kind is absent even where a
   default is in effect. Then
   `INSIDEOUT_KEYS_HOST=<host> python3 scripts/keys.py fill` —
   copies env values into **empty-only** `.key` files (dir 0700, file
   0600). Existing values are kept, never overwritten. A kind the env
   doesn't set stays blank — that is the declared fallback in effect,
   not an error.
5. **Verify.**
   - Scoped audit: `INSIDEOUT_KEYS_HOST=<host> python3 scripts/keys.py
     audit` shows `key=filled env=set` (or `key=empty` with a
     declared fallback).
   - Perms: `.secrets` dir 0700, `.key` 0600, size plausible for the
     kind (e.g. 64 B for `openssl rand -hex 32`).
   - Isolation: the *other* host's scoped audit is unchanged — fill
     cannot cross hosts.
   - Suite: `cd scripts && python3 -m unittest discover -s tests/keys
     -t .` — per-key shape/placement plus store invariants green:
     never-absent on every present host, emptiness only with a
     declared `fallback`/`optional`/`operator_fills` field (hosts
     without a `.secrets/` skip cleanly). After syncing another
     host, run it **there** too: the cross-host run is what catches
     per-host state (escaped storage forms, 0755 dirs, wrong-case
     folders on case-insensitive filesystems, undeclared blank lanes).
   - `git status` shows only the tracked manifest/README files — a
     `.secrets` path appearing there means `.gitignore` is broken;
     stop before committing.
6. **Test the key itself.** `./scripts/env.sh check` (whole `.env`,
   masked), then the key's probe from part 4 of its doc
   `docs/usage/keys/<module>/<key-name>.md` (migrate for the owner
   DSN; a bad login → 401 for the app DSN; a garbage bearer → 401 for
   the JWT path; no offline-coach notice for the LLM key; a
   magic-link request for mail; signed deliveries answered 200 for the
   webhook secret). **A boot that only warns about an unusable key is
   a failure, not a pass** — assert the warning's *absence* in the
   boot log, never just that the process started.
7. **Records.** If the key kind is new: create its per-key doc
   (six parts, filename = key name, module folder, fallback-first
   framing for config companions — "blank = catalog default X; the
   file exists so the kind is always findable"), add its row to the
   `docs/usage/keys/README.md` table, and add the SETENV-step
   down-link when a step covers it. Update the host `keys/README.md`
   state table and the per-host state table in
   `docs/usage/keys/README.md`; tick the plan checklist; extend the
   changelog; keep the relative-link check at 0 broken. Commit
   **tracked metadata only**, push, and fast-forward the other hosts'
   clones — then run `init` + `fill` + the suite on each of them.

## Failure signals

- Fill reports `skip (env missing|placeholder)` — the value never
  reached `.env`; run `env.sh edit` first.
- The tool refuses with "several hosts … set INSIDEOUT_KEYS_HOST" —
  scoping omitted once a second host exists; this refusal is what
  prevents cross-host value mixing.
- Audit `key=filled` but wrong size/perms — someone hand-wrote the
  file; re-check 0600/0700 and the kind's expected shape.
- A `.secrets` path in `git status` — do not commit; fix the ignore
  rule first.
- Everything filled but the probe fails — the value is set but stale
  or wrong (e.g. compose password ≠ the DSN's embedded password), or
  it is the wrong **storage form** for its consumer (escaped env PEM
  in a raw-read `_FILE` slot); rotate per part 3 of the key's doc or
  transform to the consumer's form
  ([why](../learning/2026-10-05-boot-warning-is-not-a-pass.md)).
- Boot succeeds but the log carries a degradation Warn about the key
  ("private key unusable", offline coach) — not a pass; grep for the
  warning's absence.
- The suite fails only on the other host — per-host state (dir modes,
  wrong-case folder, storage form, or a companion kind legitimately
  blank there while its manifest entry lacks `optional`/`fallback`);
  fix on that host — declare the lane in metadata, never fill a fake
  value — and rerun there
  ([why](../learning/2026-10-06-blank-lane-legality-lives-in-metadata.md)).
- A mapped kind with no `.key` file on a host that has others — run
  `keys.py init`; absence is the one state the scheme never allows.
- A per-key doc missing, lumped into a group file, or named unlike the
  key — the scheme's **shape** is part of the contract (filename =
  key name, one file per key, router docs link down); fix before
  committing ([why](../learning/2026-10-05-port-reference-schemes-structurally.md)).
