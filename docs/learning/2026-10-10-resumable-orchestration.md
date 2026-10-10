# A frozen orchestration is recoverable if the state is on disk

2026-10-10, the review-register sweep.

The orchestrating session froze mid-run with four agent workflows in flight
(five workers part-way through their batches, nine verifiers not yet started,
one batch launched seconds before the freeze). The resume was lossless
because of four design choices the sweep had already made:

1. **Every worker owns a git worktree and a branch.** Work in progress was
   committed work; the freeze cost at most an uncommitted edit or two, which
   the resuming worker re-read and finished.
2. **Reports and verdicts are files, not chat.** `report-<cluster>.json` and
   `verdict-<cluster>-r<n>.json` under the scratch root meant a fresh
   orchestrator could learn what had finished by listing a directory, not by
   replaying a transcript.
3. **The contract lives in files the prompts only reference** (worker rules,
   per-cluster briefs, env files). Re-promping a new agent with the same
   contract was copy-paste, not archaeology.
4. **Per-batch verification, not end-to-end trust.** Because each batch had
   its own adversarial verifier, resuming meant "finish the unfinished
   batches and verify the unverified ones" — no partially-verified mixture.

The transcript archaeology (reading subagent journals to place each frozen
worker's last action) was the only part that should not have been needed;
writing a one-line progress note per batch into the scratch tree would have
made the resume mechanical. Cost of the freeze: about one hour of recovery.
Cost without the on-disk state: the whole 24-batch wave.
