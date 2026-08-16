# 2026-07-30 — Env catalog + TUI, contract-scoped propagation, honest schema

`scripts/env.sh` gains two verbs — `edit` (a curses catalog of every variable
and its state) and `propagate` (generate `app/.env` from the root, scoped and
checksum-stamped) — `check` becomes component- and staleness-aware, `dev.sh`
gates on it, and `.env.example` is corrected so its *form* matches the truth it
claims to encode. Design adapted from the reference implementation in a sibling
repository; the plan and the ground-truth table are in
[docs/plans/2026-07-30-env-catalog-propagate.md](../plans/2026-07-30-env-catalog-propagate.md).

## What changed

**New — `./scripts/env.sh edit`.** A scrollable list of every variable the repo
knows about: ↑/↓ move, Enter sets, `c` clears, `/` filters to what is
outstanding, `q` quits. Five states per row — `set` (you chose it), `default`
(byte-identical to `.env.example`, so it came from the skeleton), `missing`,
`placeholder`, `unset` — plus which component reads it. Secrets are masked in
**both** directions (`••••••` on screen, `******` while typing), classified by
variable *name* before any value reaches a widget.

All of the logic lives in [`scripts/env_catalog.py`](../../scripts/env_catalog.py);
[`env_tui.py`](../../scripts/env_tui.py) only draws, and `edit --list` renders
the same rows without a terminal. A tty-only UI is otherwise untestable.

**New — `./scripts/env.sh propagate [component]`.** Generates `app/.env` from
the root file so the Nuxt component also works launched from its own directory.
Scoped to the keys `app/.env.example` declares (a backend secret cannot land in
the frontend directory), stamped with the root file's sha256, `chmod 600`, and
guarded: it refuses a path git does not ignore (checked *inside* the owning
directory) and refuses to overwrite a file it did not generate.
`propagate server` explains that the Go binary reads process env only rather
than inventing a file nothing would load.

**Changed — a placeholder in a required variable now FAILS `check`** (it was a
warning). Both skeleton values pass `config.Load` — the DSN is non-empty and
`change_me_to_a_long_random_string` is 33 characters — so a bare
`cp .env.example .env` previously exited 0 and `dev.sh` launched an environment
that could only fail later with an opaque DNS error. A check that cannot fail
in the direction of the bug is not a check. In an *optional* variable a
placeholder stays a warning.

**Changed — `check` is component- and staleness-aware.**
`./scripts/env.sh check [app|server] [--db]`. A generated copy built from an
older root `.env` is a **failure** for the component being launched — `dev.sh`
now passes its `-C` target, so a stale copy blocks that launch and names the
fix — and only a warning for the others, so a stale `app/.env` cannot block a
`-C server` test run. A hand-written copy is reported as such and never
clobbered; that is the deliberate-override escape hatch.

**Changed — `.env.example` is now a schema that tells the truth.** An
uncommented assignment means required, a commented one optional. Ten
variables were uncommented while being optional — including
`ANTHROPIC_AUTH_TOKEN` directly under a comment saying to leave it empty — and
two (`GITHUB_TOKEN`, `NUXT_API_INTERNAL_BASE`) were absent from the skeleton
while `env.sh`'s hardcoded `KNOWN_NAMES` already knew them. Only
`DATABASE_URL` and `INSIDEOUT_JWT_SECRET` remain uncommented, matching exactly
the two `config.Load` refuses to boot without. `KNOWN_NAMES` is deleted; the
name list is derived from the skeletons.

**New — `app/.env.example`.** Recreated, for a different purpose than the
Supabase-era file deleted on
[2026-07-27](2026-07-27-env-hygiene.md): it is the component's *contract*,
declaring the one variable Nuxt reads (`NUXT_API_INTERNAL_BASE`) and thereby
bounding what `propagate` may copy in.

**Split — `scripts/env.sh` into three files**, closing
[the ~350-line budget issue](../issues/2026-07-29-envsh-line-budget.md):
`env.sh` (dispatcher + `check` + `list`/`redact`, 276), `env-lib.sh` (shared
helpers, 191), `env-write.sh` (`init` + `propagate`, 318). The test file was
split the same way once it crossed the budget: `test_env_catalog.py` (read
path), `test_env_writes.py` (write path), `env_testlib.py` (harness).

## Why the generated copy matters here

`app/.env` was hand-written and pinned `NUXT_API_INTERNAL_BASE` to
`http://127.0.0.1:8080/api/v1`. Measured against the installed loader (c12
3.3.4), an exported value **wins** over that file — it assigns a dotenv key
only when `process.env` has none. So the line was live under
`cd app && pnpm dev` and would go inert under `./scripts/dev.sh -C app pnpm dev`
the moment the root declared the same key: a bistable file whose effect depends
on a fact invisible from the file itself. The reference implementation's hazard
was the mirror image (later-wins), and the fix is identical — stamp the
provenance and gate the launcher on it.

## What an adversarial review pass then found

A 34-agent review (four independent lenses — guards, leaks, catalog, docs —
each finding verified by a skeptic who had to reproduce it) raised 30 claims,
28 of which survived. The ones that mattered, all fixed here:

**Two guards that could not fail in the direction they guarded.**

- `load_env` sourced `.env` in a subshell and dumped `declare -p`, which cannot
  distinguish a value the file assigned from one the caller's shell exported.
  `check` therefore passed on a **zero-byte `.env`** whenever the operator's
  shell exported `DATABASE_URL` (direnv, CI secrets, an earlier `source .env`)
  — and `dev.sh` launched. It now clears every known name first and emits an
  explicit `unset` for each one the file does not set, so the file is the
  authority; a shadowing shell export is named in the failure message rather
  than left to be guessed at.
- The provenance stamp fingerprinted only the **source**, so a generated
  `app/.env` edited by hand still reported "in sync" — and the next
  `propagate` discarded the edit silently, because the DO-NOT-EDIT marker was
  still present. Generated files now carry a second `# body-sha:` over their
  own payload; `check` reports an edited copy and `propagate` refuses to
  overwrite it, naming both ways out.

**Three value leaks.**

- `redact_url` only substituted for `postgres://…@…`, so a libpq keyword DSN
  (`host=… password=…`) and a credential query parameter were printed
  verbatim. Any shape that cannot be safely reduced is now masked whole.
- `load_env` did not redirect the subshell's stderr, and bash echoes the
  **offending line** when it cannot parse an assignment. A value with a space
  was printed to the terminal by the very command you run when `.env` is
  broken. The diagnostic is now suppressed and replaced with the line *number*.
- `Var.display()` returned the raw value whenever the status was `placeholder`,
  on the reasoning that a skeleton value is public — but `placeholder` is a
  substring test on the *live* value, so a real credential containing
  `change_me` was rendered in the clear. The name test now always wins.
  `check --db` also passed the whole DSN to `pg_isready` as an argv element,
  visible in the process table; it now strips the userinfo first.

**Writers that disagreed with their own readers.**

- Both writers emitted bare `KEY=value`, so a value with a space produced a
  file `load_env` then rejected. Values are now quoted (bash and Python agree
  byte-for-byte).
- `set_var` and `propagate` resolved a duplicated key to the **first**
  occurrence while bash, compose and c12 all take the **last** — so editing a
  key that had been overridden at the bottom of the file wrote a value nothing
  used, and reported it saved. Both now target the last.

**Crash and false-alarm fixes.** The TUI aborted with a traceback on
`TERM=vt100/vt220/ansi` (unguarded `curs_set`/`use_default_colors`) and hung on
`TERM=dumb`; both now degrade and name `edit --list`. Ctrl-D at the exit prompt
raised `EOFError`; an unreadable `.env` raised `PermissionError`; a missing
`shasum` exited 127 with no message. `is_bundled_pg` hardcoded port 5442, so
the `POSTGRES_APP_PASSWORD` pairing went unenforced on any other mapped port —
it now honours `POSTGRES_PORT` for localhost forms only, which adds no false
alarm for a remote DSN. `propagate` now refuses a redirected
`INSIDEOUT_ENV_FILE` outright rather than writing scratch values into the real
component copies, and `edit` honours the same variable for both reads and
writes.

**Doc corrections** it caught, beyond this change's own surface:
`docs/architecture/deployment.md` claimed the server does *not* auto-migrate on
boot (`runServe` calls `store.Migrate` before listening); `README.md` still
taught the pre-`dev.sh` flow, implying a bare `cd server && go run` picks up
`.env`; `docs/usage/local-development.md` gave `./scripts/smoke.sh`, a path
that does not exist; and `ANTHROPIC_BASE_URL` was documented as having no
default when `anthropic.go` falls back to `https://api.anthropic.com`.

## Verification

No mocks. Every guard was run against a failing input as well as a passing one.

- `python3 scripts/test_env_catalog.py` (read path, 46 assertions) and
  `python3 scripts/test_env_writes.py` (write path, 37) — **83 in total, all
  passing**: the schema convention, set-vs-default, placeholder markers,
  masking by name, component attribution, inline-comment stripping, the write
  path (last-occurrence targeting, quoting, DSN round-trip, clear→`#KEY=`→set,
  permissions, the `INSIDEOUT_ENV_FILE` override) and the one piece of
  rendering logic that is testable headlessly (column clamping).
- **The required-placeholder gate** was exercised both ways: a verbatim
  `cp .env.example .env` exits 1 naming both variables; a real DSN and a
  generated secret exit 0; one placeholder of the two still blocks and names
  only that one.
- **Schema linters fail correctly** on four deliberately corrupted skeletons:
  a phantom name, a dead variable, an optional marked required, an inline
  comment after a value.
- **`check` guards fail correctly** on nine inputs: empty file, 31-character
  secret (fails) vs 32 (passes), placeholder DSN, bundled-pg with and without
  `POSTGRES_APP_PASSWORD`, mismatched password, invalid duration, missing file.
  A remote DSN produces no `POSTGRES_APP_PASSWORD` alarm.
- **`propagate`** refuses a hand-written target, rejects an unknown component
  (exit 2), explains `server`, and writes `app/.env` at `chmod 600`. With
  `NUXT_API_INTERNAL_BASE` set at the root it propagated exactly that key;
  `DATABASE_URL`, `INSIDEOUT_JWT_SECRET`, `ANTHROPIC_AUTH_TOKEN`,
  `GITHUB_TOKEN` and `POSTGRES_APP_PASSWORD` were all absent from the copy.
- **Staleness gate**: after touching the root `.env`, `check app` failed and
  `check server` warned; `./scripts/dev.sh -C app` exited 1 while
  `-C server` exited 0; re-propagating cleared it.
- **The git-ignore guard** was exercised against a directory where `.env` is
  *not* ignored, and refused.
- **Leak grep**: no secret value from the real `.env` (nor a DSN's embedded
  password) appears in the output of `list`, `redact`, `check` or `edit --list`.
- **The TUI was driven in tmux** with `capture-pane`, which unit tests cannot
  reach: it renders 17 variables with no phantom entry; Enter opens the prompt
  at row `h-1` without the `addnwstr() ERR` crash that class of write causes;
  a typed secret echoes as `******` and never appears in the cell grid; Escape
  cancels in well under a second; the header keeps counting the whole
  environment while `/` filters; after a write the cursor follows the row to
  its new sorted position; `c` writes the key back as `#KEY=`; quitting offers
  the re-propagate.
- **A 40-column terminal with bilingual hint text** renders without wrap or
  crash. curses' `n` argument counts characters, not columns, so a CJK glyph
  would otherwise occupy twice its budget — every skeleton comment here is
  bilingual, so this is the common row, not an exotic one. `put()` clips by
  measured column width.
- **Permissions**: `set_var`/`clear_var` chmod the temp file *before* the
  atomic rename, so `.env` is never observable at the umask default, and no
  `.env.tmp` survives.
- `.env` was backed up first and confirmed byte-identical afterwards (`cmp`).

## Operator notes

- **`app/.env` is now generated.** The previous hand-written file was moved
  aside during this change; its only content was the code default, so nothing
  changed behaviourally. If you keep a local override, `check` will report the
  file as hand-written on every run — that is intended, not a bug.
- **Re-run `./scripts/env.sh propagate` after editing the root `.env`**, or
  `dev.sh -C app` will refuse to start and tell you to. `env.sh edit` and
  `env.sh init` offer to do it for you.
- **One new way to be blocked:** if `DATABASE_URL` or `INSIDEOUT_JWT_SECRET`
  still holds a `change_me…` / `your-remote-host` placeholder, `check` now
  fails and `dev.sh` refuses. Any environment that was actually working is
  unaffected — a placeholder there could never have connected or signed a
  token you would trust.
- Otherwise existing `.env` files need no migration. Values that match the
  skeleton now display as `default` rather than `set` — a labelling change,
  not a validation one.
