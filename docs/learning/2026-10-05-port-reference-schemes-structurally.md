# Port a reference scheme structurally, not by approximation

## What was learned

When importing a convention from a reference repository (agents_home's
`docs/usage/keys/`), port its **structure faithfully** — folder
layout, file-per-unit granularity, file-naming rule, and reference
direction — instead of an approximation that "covers the same
information". The approximation still contains every fact, yet it is
wrong: users navigate by the scheme's shape, and any deviation hides
the file they are looking for.

## Evidence

Setting up the per-key usage docs (2026-10-05) took **three user
corrections in one session** for one class of mistake:

1. Lumped all 12 keys into one family doc (imported agents_home's one
   *exception* — its `insideout.md` — instead of its stated rule "one
   file per key"). User: "Shouldn't each key have its own .md file?"
2. Kept flat filenames and made every detail file point up into
   SETENV.md for guidance. User: "SETENV.md should refer to the
   detailed md files… didn't you find the absurdity?" plus "create
   folders and md files just like the way you set up hosts folder."
3. Named files after the manifest setting (`github/app.md`). User:
   "the .md file name must use the key name just like .key file."

Final shape (slices 5–7 of
[`docs/changelogs/2026-10-05-buddy-host-keys.md`](../changelogs/2026-10-05-buddy-host-keys.md)):
module folders mirroring `hosts/<host>/keys/`, 12 files named exactly
like their `.key` files, router docs (SETENV steps, keys README)
linking *down* into per-key detail.

The restructure also surfaced two latent defects of the approximation
— relative links one directory too deep, and a claimed SETENV step
that does not exist — both caught by a mechanical link check, not by
writing.

## Scope

Any port of an external convention: credential stores, doc trees,
script layouts, CI shapes. Applies the moment "the reference repo
already solved this" is the justification.

## When to apply it again

Before building, write down the reference's four rules explicitly
(granularity, naming, folder structure, who-links-to-whom) and mirror
them verbatim; deviate only with a recorded reason the user has
approved. Then run a mechanical check (link resolver, name-vs-target
diff) — structural mismatches are cheap to detect and expensive to be
corrected on.
