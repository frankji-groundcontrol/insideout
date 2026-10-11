# 2026-10-11 — A verbatim quote is evidence, not a link

## What was learned

During link-graph cleanup, five evidence quotes inside dated review
records were "fixed" by rewriting the relative link targets they quoted
so the links would resolve from the review file's own directory
(`../../client/README.md` became `../../../client/README.md`,
`(docs/TODO.md)` became `(../../TODO.md)`, and so on). The rewrites
matched neither the cited source files as they exist now nor as they
existed at the reviewed commits — after the edit the quotes were no
longer evidence of anything. The adversarial review failed the round for
exactly this, while passing every genuinely new or repaired link in the
same change. In one of the five spots the quote sat inside a fenced code
block, where links do not even render: zero benefit, pure distortion.

## Evidence

The repair round of the docs-order link surgery: the four distorted
files were restored to their committed verbatim text (`git restore`,
working-tree diff had contained nothing else), the quotes verified
against their sources at the reviewed commits —
`f185e94:docs/usage/local-development.md:11`, `ecfe4d2:README.md:90`,
`877ad03:docs/index.md:50` — and the historical-record exemption that
covers them was written into
[the docs-marksman-order plan](../plans/2026-10-11-docs-marksman-order.md)
alongside the existing Nuxt `app/` citation policy.

## Scope

Any dated record that quotes another file's line verbatim — review
findings, findings registers, skeptic verdicts, changelog citations —
especially when the quoted line itself contains a relative link. It is
the complement of [relative links are depth-sensitive](2026-10-06-relative-links-are-depth-sensitive.md):
your own links must be resolved mechanically, but a quoted link belongs
to the source document and its directory, not to yours.

## When to apply

Before "fixing" any broken or odd-looking link, check whether it sits
inside a quotation of another document. If it does, verify the quote
against the cited file — or its `git show <commit>:<path>` version at
the record's date — and keep or restore it verbatim, routing the link
into the recorded broken-link exemption instead of editing the quote.
A link gate that fires on a quoted string is asking for an exemption
entry, never for a rewrite.
