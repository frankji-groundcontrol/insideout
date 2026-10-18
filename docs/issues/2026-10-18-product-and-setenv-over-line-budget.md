# PRODUCT.md and docs/SETENV.md are over the line budget, and wave 2 grew both

**Date:** 2026-10-19
**Kind:** structural debt (line-budget split), no behaviour change
**Files:** [`PRODUCT.md`](../../PRODUCT.md) — 462 lines (460 before the
[review-4 remediation](../changelogs/2026-10-18-rv4-remediation/index.md); wave 2
added 2) and [`docs/SETENV.md`](../SETENV.md) — 368 lines (348 at the review's
`9f0b1c7`, 363 at `d53035a`; wave 2 added 5), both against the ~350-line budget.
**Found by:** the [wave-2 records pass](../changelogs/2026-10-18-rv4-remediation/wave-2.md),
which had to state the agent boundary and the serve-role check in each file and
could not split either inside a records pass.

## Why it matters

Both files are reference pages that a reader, or an agent, loads whole, and the
repository rule is single-responsibility files of about 350 lines or fewer.
`PRODUCT.md` is also read by code and by guards (below), so a split has to move
text without moving what they read. Nothing is wrong with the content; the files
are long and each new fact lengthens them.

## Target structure

**`PRODUCT.md`** has sixteen sections. Keep it as the canonical entry at about
260 lines: the thesis, who it serves, the principles, the canonical journey, the
Coach and the first workable version, the audience views, facts and privacy,
superseded assumptions and current status, and brand commitments. Move the seven
delivery-shaped sections (about 220 lines) to one file each under a new
`docs/product/` folder with its own README index: product version control,
collaboration and authority, the shared Roadmap Graph, Git, CLI, MCP and agents,
completion and learning, the minimum coherent end-state loop, and the derived
delivery defaults. Each moved section keeps its heading and leaves a
one-sentence stub with a link in `PRODUCT.md`, so a reader of the entry still
sees the whole shape.

**`docs/SETENV.md`** is a hands-on walk: layout, first-time setup, five steps,
"Prove it". Keep that walk and move the two reference blocks that are not a step
into `docs/usage/`: "The skeleton is the schema" (about 45 lines) and "Never
print a secret" with "Hard don'ts" (about 40 lines). That leaves about 285
lines.

## What the move must not break

- `scripts/tests/check_docs_truth.py` reads the superseded-assumptions paragraph
  of `PRODUCT.md` (it must keep "scaffold" and "prds.go"), and
  `scripts/tests/check_product_promise_gaps.py` requires the status section to
  link the promise-gaps record. Both sections stay in `PRODUCT.md`.
- `server/internal/agent/prompts_contract_test.go` and the clarify prompt cite
  the Coach contract in `PRODUCT.md`; that section stays. Its line citations are
  already comments, not checks.
- Dated review and plan records cite `PRODUCT.md:<line>` as of their own commit.
  They are history and are not renumbered; a live page that links a moved
  section by name is repointed.
- Keep the five `Step` headings in `docs/SETENV.md`: the pages under
  `docs/usage/keys/` and the key tutorials link to them, and `scripts/env.sh`
  names "SETENV.md step 1" in a message.
- The doc map ([docs/index.md](../index.md)) and the nearest README index list
  every new file, and `sh scripts/check-docs-recorded.sh` stays green.

## Bounded fix prompt

Split the two files as above in two pure-move commits (no wording change), each
with a changelog entry. Before the move, grep for `PRODUCT.md` and `SETENV.md`
links by section name and by anchor, and list each one with the file that will
own the text. After it, run the nine `scripts/tests/check_*.py` guards with
`python3 -I`, `sh scripts/check-redaction.sh` and a relative-link check over
every touched file, and confirm every file is at or under 350 lines.
