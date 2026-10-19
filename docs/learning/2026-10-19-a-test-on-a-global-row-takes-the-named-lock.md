# A test on a global row takes the named lock and puts the row back

**Date:** 2026-10-19
**Source:** [multirole review 4](../changelogs/2026-10-18-multirole-review-4.md), findings TEST-02, TEST-06 and CLOSE-08 (shared-database test races)
**Fixed in:** [wave 3, test hardening](../changelogs/2026-10-18-rv4-remediation/w3-test-hardening.md)
([practice](../practices/2026-07-21-verify-against-real-database.md))

## The lesson

`go test ./...` starts one test binary per package, in parallel, as separate
processes. Nothing inside a process (`t.Parallel`, a mutex) can order two tests
of different packages. What they share is the database, so the database orders
them. `store`'s `TestSignupModeSetting` sets `signup_mode` to `open` while
`api`'s `TestInvitationRoundTrip` needs `invitation_only`; each passes alone and
the pair fails depending on timing.

- **One lock per global row,** not one big lock: `signup_mode` and the breaker
  row each have their own session-level advisory lock, so unrelated tests still
  run in parallel. `internal/testlock` provides it (`testlock.SignupMode(t,
  pool)`, `testlock.Acquire`), on a connection taken out of the pool so a test
  binary that dies frees the lock when its session ends.
- **A lock only counts if skipping it is caught.** `testlock.AssertWaits` holds
  the lock itself, starts the test binary again to run one named test, and
  requires the child to park on the lock (seen in `pg_locks`). The names in each
  call are the registry a new test that touches the row is added to.
- **A live check is a toucher too.** A hand run that flips a global row outside
  the Go tests has no lock; it must restore the seeded value. One wave-3 live
  check left `signup_mode` open and the next full run failed `TestSignupModeSetting`
  until the value was put back.

## When to apply

- Writing a test that reads or writes `signup_mode`, the production
  circuit-breaker row, or any other row with one value for the whole database.
- Running a live check or a smoke script against a database other suites use:
  restore every global row you changed, in the same script.

## Evidence

The red test of TEST-02 failed on the unfixed tests because they finished at
once instead of waiting on the lock. Strict mode (`INSIDEOUT_TEST_STRICT`) makes
a pool whose host is not loopback or a socket fail the test, so the shared
instance is not hit by accident. The remaining toucher cases are listed as open
residue in the
[open-findings record](../issues/2026-10-18-review-4-open-findings.md).
