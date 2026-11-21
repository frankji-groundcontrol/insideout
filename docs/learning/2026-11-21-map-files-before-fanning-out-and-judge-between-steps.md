# Map files before fanning out, and judge between steps

**Date:** 2026-11-21
**Source:** the orchestration of the review 5 remediation
([changelog](../changelogs/2026-11-21-rv5-remediation/index.md),
[plan](../plans/2026-11-21-rv5-remediation/2026-11-21-rv5-remediation.md))

## The lesson

Waves 0 to 3 ran each cluster as a fixed workflow: a worker, then an
adversarial verifier, then up to two repair rounds, with no decision in
between. Waves 3 to 5 then switched to direct dispatch, after the owner asked
why the waves ran as fixed loops when nothing was orchestrating them. Three
practices made the difference.

- **Map each cluster's files against what is already running, then start
  everything that does not overlap.** The plan ordered the waves 3, 4, 5. A
  file map showed that half of wave 4 and half of wave 5 shared no file with
  wave 3, so those clusters ran beside it:
  - setup-engine and rule-guards beside all of wave 3;
  - doc-records at the same time;
  - agent-split once wave 3's server branch was accepted, on that branch;
  - the server half of the renames beside wave 3's CLI, MCP and web stages.

  Only the clusters that shared files with wave 3 waited for it (the MCP and
  CLI splits and the web controllers), and the link rewrite waited for
  everything.
- **Read each report and diff yourself, and fix small findings in place.** In
  the fixed loop, one repair round changed no code: it fixed a usage-doc
  wording problem and two nits, and cost a full extra repair and verify cycle.
  Each repair and verifier agent re-ran the whole gate suite. Under direct
  review, the coordinator settled the same kind of finding in a few minutes:
  - an index row that two parallel branches had each changed;
  - a test file that a parallel branch added at the old layout;
  - a long-function list that the splits had made stale.
- **Keep a guard that runs beside other work a ratchet with paste-ready
  output.** The function-length guard was written while other clusters were
  still growing and splitting functions. It fails on an unlisted long function,
  on a listed one that grew, and on an entry whose function is gone or now
  short. Its failure message prints the entries ready to paste. Re-running it
  after each integration kept the list exact. The first run after wave 3
  merged caught two functions that wave 3 had grown beside it.

## Evidence

- The fixed-loop wave 2 follow-up: 6 agents, about 816,000 tokens and 53
  minutes for two small clusters. Its server repair round changed no code: it
  fixed one usage-doc wording problem and two nits.
- Waves 3 to 5 under direct dispatch: 10 clusters. The coordinator settled
  every cross-branch overlap at integration. Full gates ran once per
  integration, not once per agent.

## When to apply it

Use a fixed workflow only when the script itself makes decisions or fans out
wide. Otherwise, before starting a wave:

1. List each cluster's files.
2. Start every cluster that overlaps nothing still running.
3. Review each result yourself.
4. Run the gates once on the integrated tree.

Brief any guard that runs beside other work to print paste-ready entries, and
re-run it after every merge. See also
[one owner per shared resource](./2026-10-19-one-owner-per-shared-resource-in-a-multi-worker-wave.md).
