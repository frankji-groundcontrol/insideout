# A truncated gate snapshot can mislabel its sections

**Date:** 2026-10-11
**Source:** [docs-order final audit D, repair round 1](../changelogs/2026-10-11-docs-order-marksman.md)

## The lesson

The repair ask pasted the gate's detail through a filter that kept any
line containing "ORPHAN" or starting with two spaces — so the
MISSING-README-dirs rows and the broken-link rows of `graph.py`'s
output landed under the pasted "ORPHANS" header. Read as-is, the round
looked like it must fix ~30 orphan folders and a list of dead links;
read from the script, the gate line itself (`GATE orphans=0
md_broken=0`) was already green and the only true gap was the other
gate: coverage `missing=8 unlink=103`.

## Why it happens

Gate scripts print several sections with uniform indentation; any
line-prefix filter that captures one section's rows captures them all
and silently drops the section headers that gave those rows meaning.
The summary line and the detail sections are separate facts — a pasted
excerpt is not the gate's verdict.

## When to apply

- Repairing from a pasted gate excerpt: re-run the gate script (or
  read its JSON) and classify findings by the script's own section
  semantics before editing anything.
- Writing a workflow that forwards gate detail: forward the summary
  line plus the exact named section(s) needed, not an indentation
  filter over the whole output.

## Evidence

D round 1 (dwfrun-c214ad44): the ask's "graph 详情" merged the
`== MISSING README dirs ==` and `== BROKEN md links ==` rows under
ORPHANS. Re-running `python3 /tmp/insideout-gap/docs/graph.py` read
`orphans=0 md_broken=0`; `readme_coverage.py` read `missing=8
unlink=103`. The repair fixed only the coverage gap (8 folder READMEs,
101 record links, 2 plan links) and every gate ended green — including
`GATE orphans=0 md_broken=0` re-verified on the settled tree.
