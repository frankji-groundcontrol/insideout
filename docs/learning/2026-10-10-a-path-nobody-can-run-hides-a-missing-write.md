# 2026-10-10 — A path nobody can run hides a missing write

## Context

`insideout setup --local --postgres bundled` and its shell twin
`scripts/setup.sh` write `DATABASE_URL` as the app role and then run the first
migration. The app role may not migrate, so the run ends with
`connected as "insideout_app"; use DATABASE_OWNER_URL`. The path needs Docker,
which neither the 2026-10-07 live walk nor the later waves had, and its
helpers were covered only by pure unit tests (`bundledDSN`), so the missing
`DATABASE_OWNER_URL` write was never seen. A reviewer reading the code found
the suspicion; running the nearest real piece confirmed it.

## Lesson

"Needs Docker" is not a reason the path stays unexecuted: stub only the
Docker call (a no-op `docker` on `PATH`) and keep everything else real,
including a real PostgreSQL behind the DSN. That reproduced the failure
exactly, before and after the fix, with the binary built from each commit.
Unit tests of the helpers around an unrun path prove the helpers, not the path.

## When to apply

- Any flow with an external prerequisite you cannot run on this host: list the
  prerequisite's observable effects, fake only those, run the rest end to end.
- When a worker reports "by reading the code this would fail", reproduce it
  before trusting or dismissing it.
