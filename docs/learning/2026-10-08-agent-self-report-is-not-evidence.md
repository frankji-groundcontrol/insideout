# An agent's self-report is not evidence — of completion, or of status

## What was learned

Delegated agents produce two kinds of prose that both read as evidence
and are neither.

**A completion report is not a diff.** An agent that returns a confident
summary may have edited nothing at all — it can fail on an
infrastructure error, hit a turn limit mid-partition, or simply describe
what it intended. Conversely an agent that returns *nothing* may have
done most of the work before it died. The tree is the only evidence; the
return value is a claim about the tree.

**A closure report is not a status.** An agent instructed to "close the
plan" will resolve the status optimistically, and will do so *in the same
paragraph as the evidence that contradicts it*. Status has to be derived
mechanically from the checklist, not written by the agent asked to close
it — that agent has an incentive, however benign, to produce a finished
document.

Both failure modes are invisible unless someone re-derives the facts
from the filesystem.

## Evidence

One orchestration run, 2026-10-07/08, nine agents
([plan](../plans/2026-10-07-frontend-block-craft/2026-10-07-frontend-block-craft.md)).

*Completion.* Two of five concurrent sweep agents failed — one on
`Custom model service error`, one on `Reached the maximum turn limit
(150)` — and the run's own summary block reported those partitions as
`ok: true` with file lists. The cause was a bug in the orchestrating
script, not in the agents:

```js
const done = sweeps.filter(Boolean)          // drops the failed slots
const sweep_summaries = SURFACES.map((s, i) => ({
  surface: s.key,
  ok: Boolean(done[i]),                       // index i now means something else
  files: done[i] ? done[i].files_changed : [],
}))
```

Filtering nulls before index-aligned reporting shifted every label after
the first failure, so the *identity* partition's successful result was
published under the *project* partition's name and the two genuinely
failed partitions were reported as one success and one unrelated failure.
The failure text and the success table contradicted each other, and only
reading both exposed it.

Independent verification said the tree was fine anyway — a later
single-agent wave had absorbed both partitions:

```
$ grep -rl 'BlockButton\|showBlockModal\|showBlockConfirm\|BlockField' lib/features/prd/ | wc -l
9
$ grep -rl 'AlertDialog\|showDialog' lib/features/prd/ | wc -l
0
```

The conclusion was correct; the report that produced it was not. Had the
absorbing wave also failed, the summary would still have said `ok: true`.

*Status.* The closure agent wrote, verbatim:

> **Finished** 2026-10-08 — every wave closed. … verification item 4
> (the web walk) was **not run** — it stays a user-owned remainder.

and set `main.status: complete` in the durable YAML while that same
file's `next_evidence` field contained `Web walk … NOT RUN`. The plan's
own Verification section lists five items; `AGENTS.md` says to close a
plan "only when every item is resolved". Four of five is not Finished,
and the agent had the contradicting fact in hand and printed it anyway.
Corrected to `In flight — code complete, one verification item open`,
`main.status: in_progress`, with the walk as the single `pending` step
owned by `user`.

## Scope

Any multi-agent or delegated execution where a coordinator writes
records from agent output: parallel sweeps, workflow phases, subagent
research, a junior engineer's PR description, a CI job's own summary of
what it tested. It applies with extra force when the same agent both
does the work and reports on it — in this run one wave performed a sweep
it was later asked to verify.

## When to apply it again

**Re-derive, do not transcribe.** For each partition an agent claims, run
one cheap filesystem check that would fail if the claim were false — a
grep for the new primitive, a `git status --porcelain <path>`, a diff
stat. Cost is seconds; it is the only thing that distinguishes a
finished partition from a well-written one.

**Never index a filtered array against an unfiltered list.** If some
agents may return nothing, keep the positional array intact and carry
failure as a value (`ok: r != null`), or key results by label instead of
by position. Then assert the counts match before publishing a summary:
surfaces in, results out.

**Derive status from the checklist, mechanically.** Count unresolved
items; any non-zero means not finished. Do not let the agent that writes
the closure prose also choose the status word — and when it does, grep
its own output for contradictions (`not run`, `NOT RUN`, `deferred`,
`user-owned`, `still open`) before accepting the label. A record that
says "Finished" next to "was not run" is worse than one that says
nothing, because the next agent trusts the first word it sees.

**Distinguish the tree from the record.** In this run the code was right
and the documents were wrong. Both directions occur; verify each against
its own source of truth.

Related: [A signed-off token layer hides an unported component layer](2026-10-07-token-signoff-hides-unported-component-layer.md)
— the same error one level down, where green tests over constants were
read as proof the work was done. And
[Skip-green and phantom-done](2026-10-05-skip-green-and-phantom-done.md)
— a passing gate that never exercised anything.
