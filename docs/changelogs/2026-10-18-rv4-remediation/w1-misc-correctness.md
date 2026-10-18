# Wave 1, misc correctness — review-4 remediation

Branch `rv4/w1-misc-correctness`, `1dba3fe..66963b4` (10 commits: five red/fix
pairs), then this fragment. Findings are cited as `rv4-<ID>` because review IDs
restart per review. Row 1.15 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md),
without its CI half: the `go mod tidy -diff` step in `ci.yml` belongs to
`w1-ci-gates`. Each red commit fails on its own and its fix turns it green; the
database-backed one ran against a private PostgreSQL 18 with no skip. The
branch already existed at the wave base, so the worker reused it.

## What changed

- **`go.mod` marks the directly imported TUI library as direct** (rv4-PM-10,
  red `01b9351`, fix `bf9ea46`). `github.com/charmbracelet/huh` is imported by
  `internal/setup/ui` but sat in the `// indirect` block. A new test-only
  package, `internal/modtidy`, runs `go mod tidy -diff` (offline first, then
  with the ambient proxy; it fails loudly if tidy cannot run) and fails when
  the `go.mod` half of the diff is not empty. The fix moves the requirement to
  the first block. The `go.sum` half is only logged, see below.
- **The `propose` MCP tool declares the `items` it reads** (rv4-AIDEV-10, red
  `a8d623d`, fix `5a9dda6`). The handler always consumed an `items` array, but
  `tools/list` advertised only `project_id`, `kind`, `summary` and `detail`, so
  a schema-driven client could not know structured proposals exist while the
  CLI's `--item` flag worked. The schema now declares an optional array of
  objects: `title` (required, at most 200 characters) and `parentHint`
  (optional); the description states the server's limit of 10. The handler is
  unchanged.
- **The coach's open questions have one order and one blank rule** (rv4-QUAL-01,
  red `90772df`, fix `29b8cdc`). `agentcontext` listed `openQuestions` by
  ranging over a map, so the same PRD gave a different order on every call, and
  it called a section blank only when it was `""`. `readiness` already treated
  absent and whitespace-only sections as blank. `readiness.BlankSections` is
  now the single implementation (fixed order: background, constraints, goals,
  nonGoals, requirements, risks, stories, users) and `agentcontext` calls it.
  Two behaviour changes follow on purpose: a whitespace-only section is now an
  open question, and a brainstorming context for a PRD with no sections at all
  now lists all eight (it listed none, because only keys present in the map
  were looked at). The existing brainstorming test accepted any of six keys
  first; it now pins the exact ordered list.
- **`env.sh check` fails an empty owner DSN on the bundled postgres**
  (rv4-FULL-06, red `d278d4b`, fix `e384224`). The bundled postgres'
  `DATABASE_URL` is the `insideout_app` role, which cannot apply migrations, so
  compose died at `migrate` while the checker printed an info line and exited
  0. When `DATABASE_OWNER_URL` is empty and `is_bundled_pg` holds, the check now
  fails and names the cause and the fix. Any other database keeps the info
  (there the app DSN may already be the owner). `scripts/dev.sh` runs this
  check first, so it now refuses to launch in that state. Three tests drive the
  real script against a throwaway env file: the bundled `@postgres` host and
  local port 5442 fail (and the failure output carries no password), the same
  with an owner DSN passes, and a remote host or another local port stays an
  info.
- **The timeline cursor survives a deleted row** (rv4-DATA-05, red `5b5b210`,
  fix `66963b4`). `ListProjectUpdates` resolved its bare-id cursor with a
  subselect on `project_updates`, so deleting the row the cursor came from made
  the keyset comparison NULL and the next page came back empty: a client
  mid-scroll silently lost all older history. The cursor is now a value,
  `UpdateCursor{CreatedAt, ID}`, taken from `ProjectUpdate.Cursor()` and
  consumed by the new `ListProjectUpdatesBefore`, which compares against the
  pair and needs no lookup. `ListProjectUpdates` keeps its signature and
  delegates: it resolves the id to a position first, scoped to the project, and
  answers `ErrNotFound` for an id that is no longer a row of that timeline
  (deleted, or from another project) instead of an empty page that looks like
  the end of history. The red is a build failure, since the new API does not
  exist yet; the behaviour on the unchanged code was captured with a throwaway
  test: after the delete, page 2 returned 0 rows and no error where 2 were
  expected. The one production caller (the first page in `GetProject`) passes
  no cursor, and the REST API has no cursor parameter yet, so nothing else
  moves. A future endpoint should hand the client the pair, not a row id.

## Left for the integrator

These sit outside this cluster's files and are the reason rv4-PM-10 is
`PARTIAL`:

- **`go.sum` needs 18 more lines.** `go mod tidy -diff` has two halves, and
  after the fix only the `go.sum` half is left: hashes for the test-only
  dependencies of `huh` and its neighbours. The cluster did not own `go.sum`,
  so a ready patch (additions only; applied to a scratch copy, tidy then
  prints nothing and exits 0) is attached to the worker report. Apply it in
  the same integration step as the `w1-ci-gates` change to `ci.yml`, otherwise
  that new `go mod tidy -diff` step is red on its first run. Afterwards
  `TestGoModIsTidy` can assert the whole diff instead of only `go.mod`.
- **`docs/architecture/backend.md`** must list `modtidy/` in its Layout tree:
  `scripts/tests/check_docs_truth.py` fails with "backend.md's Layout tree omits
  internal/modtidy/" until then, and that is the only reason it fails here. A
  one-line patch (verified in a scratch copy: the guard passes with it) is
  attached to the worker report.
- The shared indexes, the open-findings record and the plan track for row 1.15
  are the integrator's, as in wave 0.

## Verification

Run in this worktree at the branch head; the database-backed runs used a
private PostgreSQL 18 under `env -i` with its own DSNs (no value is spelled
here).

- `gofmt -l` prints nothing; `go build ./...` and the scoped `go vet` pass.
- `go test -race -count=1` over `mcpserver`, `agentcontext`, `readiness` and
  `modtidy`: pass. The store tests `-run 'ProjectUpdates|Cursor'`: three
  tests pass, zero skips, also under `-race`.
- A mutation check on the DATA-05 test: dropping the project scope from the id
  lookup turns the other-project assertion red again.
- Python: the key suites (52 tests; the 33 skips are the existing "no key store
  or host on this machine" cases, none of them new), `test_env_catalog.py` (48)
  and `test_env_writes.py` (37) pass. Of the six `scripts/tests/check_*.py`
  guards, five pass and `check_docs_truth.py` fails for the Layout line above.
- `go mod tidy -diff` prints only the `go.sum` half (see above); it prints
  nothing once the patch is applied.
- Whole server suite, run once: every package passes except
  `TestProjectQuietFlagged` in `internal/api`, which fails identically at the
  wave base (a clock bug owned by `w1-clock-and-runes`) and is untouched here.
- Not run here: Flutter, and CI. Nothing is pushed; **pending first CI run
  after push**.
