# 2026-07-29 — env.sh over the ~350-line modular-file budget

> **Resolved 2026-07-30.** Split as predicted below, when the file next needed
> structural change (adding the `edit` and `propagate` verbs). The result is
> `scripts/env.sh` (dispatcher + `check` + `list`/`redact`, 227 lines),
> `scripts/env-lib.sh` (shared helpers, 110) and `scripts/env-write.sh`
> (`init` + `propagate`, 278) — all under budget, all `bash -n` clean and
> bash 3.2 compatible. The `KNOWN_NAMES` constant named in the target
> structure was not moved but **deleted**: it had already drifted from the
> skeletons it duplicated, so the names are now derived from them. See
> [the changelog entry](../changelogs/2026-07-30-env-catalog-propagate.md).
>
> The original record follows, unedited.

`scripts/env.sh` is now 365 lines — over the project's ~350-line soft limit
([CLAUDE.md](../../CLAUDE.md), "Modular files"). It got there by accretion: the
`check` verb grew a defaults-in-effect report (`report_defaults`), an empty
`GITHUB_TOKEN` visibility block, and a file-identity line on top of the
interactive `init` flow that already made up the bulk of the file.

## Why not split now

The file is still single-responsibility (one tool: set up + validate `.env`)
and reads top-to-bottom as helpers, then `check`, then `init`, then dispatch.
Splitting it into sourced helper files purely to hit a line count would be
false modularity — more files, same responsibility, and a sourced-file
indirection bash 3.2 makes clumsy. Recording the debt and splitting when the
file next needs structural change is the smaller total diff.

## Target structure (when next touched)

- `scripts/env.sh` — thin dispatcher: arg parse, `usage`, source the helpers,
  run `do_init` / `do_check`.
- `scripts/env-lib.sh` (sourced, not executable) — `load_env`, `mask_secret`,
  `redact_url`, the `ok/warn/FAIL/info` markers, `exact_flag`,
  `report_defaults`, the shared `KNOWN_NAMES`, plus the logic that currently
  lives inline in the verbs (the TTL duration regex, the `apply_changes`
  write-back, and the URL-password awk splice).
- `do_init` / `do_check` move into the lib too, or into a second sourced file
  if the lib itself gets chunky.

## Bounded fix prompt

"Split `scripts/env.sh` (~365 lines) into a thin dispatcher plus a sourced
`scripts/env-lib.sh` holding the helpers and verbs, per
[docs/issues/2026-07-29-envsh-line-budget.md](2026-07-29-envsh-line-budget.md).
Keep behavior byte-identical: re-run the `INSIDEOUT_ENV_FILE` battery (empty /
fully-set / partial / missing files, plus the leak grep) and `bash -n` on
both files; keep both under ~350 lines and bash 3.2 compatible. Record the
split in a changelog entry and close this issue."
