# One owner per shared resource, and a brief checked against file ownership

**Date:** 2026-10-19
**Source:** the orchestration of the review-4 remediation, waves 2 and 3 ([remediation index](../changelogs/2026-10-18-rv4-remediation/index.md))

## The lesson

A remediation wave runs one worker per cluster, in parallel, with a verifier
after each. What went wrong or nearly wrong was never the code; it was two
things sharing something one of them did not own.

- **One owner per shared resource.** The planner wrote its own port table into
  every cluster note while the launcher assigned ports by plan index, and the
  database helper wipes the data directory of its port. Two owners of one port
  would have destroyed a worker's database. The launcher's table is the only
  source; planner-authored ports are stripped from the notes.
- **A dependency on several predecessors needs a real merged base.** "Branch
  from the last predecessor" silently dropped the others from the worker's tree.
  A base-builder step merges the accepted heads (`--no-ff`, a build and vet
  check, a conflict marks the cluster blocked) and every later step starts from
  that commit.
- **Gate commands must not call scripts that load `.env`.** `scripts/dev.sh`
  exports the root `.env`; a planner gate that used it would load that file into
  a worker's run. Gates are rewritten onto the private environment prefix before
  a worker sees them.
- **A cluster's brief must not ask for an edit outside its owned files.** The
  carry-forward text told the release worker to change `docs/usage/releasing.md`,
  `scripts/install.sh` and `server/Dockerfile`, none of them owned. The verifier
  accepted everything technical and still returned the cluster for repair; the
  edits were reverted and moved to a proposed wave 4. The privacy brief named
  `README.md`, `docs/issues/README.md` and `docs/usage/*`, and its first round
  added three opener files outside its owned paths, deleted in round 1. A lint
  that lists every repo path a brief names and the cluster's owned files do not
  cover flags both.
- **A verifier-granted ownership extension is invisible to the next verifier.**
  The prompt of the next round carries the static owned files; it worked once
  because the verifier derived the extension from the worker's report.

## When to apply

- Before launching a wave: run the brief-versus-ownership lint, and read the
  port and path tables for two owners of one thing.
- Before an expensive launch: run the generated workflow script offline with a
  fake `agent()` (happy path, a blocked predecessor, a merge conflict). It caught
  nothing here, and proved the base, port and merge-order logic in seconds.

## Evidence

Recorded in the coordinator notes of waves 2 and 3 and in the cluster fragments
of the [remediation folder](../changelogs/2026-10-18-rv4-remediation/index.md);
the privacy cluster's round 1 and the release cluster's repair round are the
two cases of the brief problem.
