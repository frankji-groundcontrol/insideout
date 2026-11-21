# Practice: Probe a real database without touching the tree

**Date**: 2026-11-21

## Trigger

A review or an investigation needs to learn what the store or an API handler
actually does (who is refused, what a call writes), the checkout holds
someone's unfinished work, and no file may be added to it.

## Sequence / guardrail

1. Start a private throwaway PostgreSQL cluster on a high local port with
   `initdb` and `pg_ctl` from the installed PostgreSQL, create the two roles
   the way `.github/workflows/ci.yml` does (an owner role, and an app role
   with `NOBYPASSRLS`), and apply the migrations with
   `go run ./cmd/insideout migrate`. Never point a probe at the database in
   `.env`.
2. Write the probe as a Go test file outside the repository, in the package
   under test, reusing that package's own test helpers (`newTestStore` and
   `mkUser` in `server/internal/store/authz_test.go`). Record what happens
   with `t.Logf`; a probe reports, it does not assert.
3. Write an overlay file that maps a path inside the package to the probe:
   `{"Replace":{"<repo>/server/internal/store/zz_probe_test.go":"/tmp/<dir>/probe_test.go"}}`.
4. Run `go test -overlay /tmp/<dir>/overlay.json ./internal/store/ -run <Probe> -v -count=1`
   with `DATABASE_URL` set to the app role of the private cluster.
5. To read rows the app role cannot see, open a second connection with the
   owner role inside the probe; the store's own pool is subject to row-level
   security.
6. Stop the cluster and delete its directory when the work is done.

## Verification

`git status --short` shows no new file after the run, and the probe's log
lines appear under `--- PASS` in the `-v` output (not `SKIP`).

## Failure signals

- The probe is skipped: `DATABASE_URL` did not reach the test.
- A count read through the store's pool is always zero: the read ran as the
  app role with no user set.
- A `zz_*_test.go` file shows up in `git status`: the probe was written into
  the tree instead of being overlaid.

## Related

- [Verify database-dependent work against real PostgreSQL](./2026-07-21-verify-against-real-database.md)
- First used in [focused review 5](../review/2026-11-21-focused-review-5/README.md) (rows R1, R3 to R5, R11).
