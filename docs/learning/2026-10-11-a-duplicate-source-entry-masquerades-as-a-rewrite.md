# 2026-10-11 — A duplicate source entry masquerades as a rewrite

## What was learned

Auditing a "pure move" by normalized comparison, a bullet that matches no
source entry looks like a rewrite — but it may be a verbatim copy of a
*different occurrence* of the same entry in the source. The old changelogs
README listed `2026-07-26-backend-optimization-pass.md` twice: a shorter
entry in date position and a fuller one (with the deferred-findings
bug-book link) appended later. The README split collapsed the duplicate by
copying the fuller entry verbatim into `index/2026-07.md`. The fix round's
comparison flagged that bullet as a fourth "rewrite" (the human review had
found only three real ones) and reverted it to the shorter first entry —
silently downgrading the description and re-introducing ambiguity about
which occurrence was canonical. Re-running the comparison after the
"fix" showed the long source bullet newly missing, which exposed the
duplicate; the fuller text was restored and the collapse recorded in the
issue's resolution notes.

## Evidence

`git show HEAD:docs/changelogs/README.md` carries both entries (the short
one and the long one under the same link); the normalized bullet
comparison (`python3` script over the four `index/*.md` files vs the
README's Records section) before and after the mistaken revert shows the
missing-source entry flipping from the short text to the long text — the
signature of a duplicate, not of a rewrite. Recorded in the
[line-budget issue's resolution notes](../issues/2026-10-11-changelogs-readme-line-budget.md).

## Scope

Any verbatim-move or copy audit: README splits, folder moves, generated
indexes, changelog migrations. Also applies in reverse — a "duplicate
link" reported by a coverage check may hide two intentionally different
descriptions.

## When to apply

Before calling a moved bullet rewritten, match it against **every**
occurrence of its target in the source, not a 1:1 first-match map. When
the source lists the same target twice with different text, choose which
occurrence survives explicitly, prefer the richer/later one, and record
the collapse where the move is authorized (the issue or plan) so the next
audit does not "restore" the other variant.
