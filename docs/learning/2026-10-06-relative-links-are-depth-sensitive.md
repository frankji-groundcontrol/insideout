# 2026-10-06 — Relative links are depth-sensitive: resolve them mechanically

## What was learned

The same documentation target linked from files at different directory
depths needs a different relative prefix per file, and eyeballing it is
unreliable. Two of ten touched files shipped broken relative links in
their first draft: `skills/insideout-setup/references/modules.md` (one
level deeper than its sibling `SKILL.md`) wrote `../../docs/…` where
`../../../docs/…` was required, and a changelog wrote `usage/README.md`
where `../usage/README.md` was required. A mechanical resolver caught
both before anything was committed.

## Evidence

[Install-skill changelog](../changelogs/2026-10-06-install-skill.md),
verification section; the check is a ~20-line Python pass that
extracts `](target)` pairs per file and calls `os.path.exists` on
`normpath(join(dirname(file), target))`, skipping absolute URLs. It
runs in seconds over every file a change touched.

## Scope

Applies to every doc edit that adds relative links — skill files,
changelogs, plans, issue records — especially when a new file lives one
directory deeper or shallower than its neighbors, or when the same
target is linked from more than one depth in the same change.

## When to apply

Before claiming any docs-only change done: run the resolver over the
touched files (or the whole `docs/` + `skills/` tree) instead of
trusting visual inspection. Deep-link anchors (`#step-0--…`) still need
a human read — the resolver only proves the file exists, not the
heading.
