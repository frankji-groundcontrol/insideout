# Plans and task board

This is the authoritative status board for concurrent repository work. Update
the table whenever a task starts, changes state, becomes blocked, or reaches a
checkpoint. Each linked plan owns its detailed checklist and decisions;
`docs/HANDOFF.md` only explains how the next agent should resume.

## Current board

| Priority | Task | Status | Next action | Blocker / note |
| --- | --- | --- | --- | --- |
| P0 | [Review 5 remediation (six waves)](./2026-11-21-rv5-remediation/2026-11-21-rv5-remediation.md) | **In flight** 2026-11-21 — waves 0 to 2 done ([wave 0](../changelogs/2026-11-21-rv5-remediation/wave-0.md), [wave 1](../changelogs/2026-11-21-rv5-remediation/wave-1.md), [wave 2](../changelogs/2026-11-21-rv5-remediation/wave-2.md)): green tip, the slice committed, the build path and the idea body fixed, the coach transcript live, the task tree for roles built on every surface; owner decisions answered ([record](../issues/2026-11-21-review-5-open-findings.md)) | Wave 3: surface parity (server contract, then CLI/MCP and web, then the parity test). | The branch is pushed after each wave; no CI is run (owner's request) and verification is local. `main` is not pushed. [Changelog](../changelogs/2026-11-21-rv5-remediation/index.md). |
| P0 | [Multirole review 4 + remediation handoff](2026-10-18-multirole-4.md) | **Waves 0 to 4b on origin/main** — review delivered 2026-10-18 (145 findings; [report](../review/2026-10-18-multirole-4/README.md)); waves 0–4 at `e6a3d14`, wave 4b and the adopted decisions pushed 2026-11-21 at `4d186da9` ([sync record](../changelogs/2026-11-21-wave-4b-sync-and-push.md)). | Watch the first CI runs on `4d186da9`: `security` is red on [govulncheck toolchain findings](../issues/2026-11-21-govulncheck-go-toolchain.md). Operator consoles stay the owner's ([adopted settings](../issues/2026-10-20-rv4-decisions-adopted.md)). | Before the first deploy, confirm the hosted `DATABASE_URL` is the app role, set `INSIDEOUT_TRUSTED_PROXY_HOPS=1`, and inspect the hosted app role for migration `20261018200000`. |
| P1 | [Product promise slices (nine PRODUCT.md promises the product does not yet keep)](../issues/2026-10-11-product-promise-gaps.md) | **Pending** — nine slices recorded 2026-10-11, none started; the enumeration carries the recommended order | Take slice 1, [Product Branch, Merge and the Decision Log](../issues/2026-10-11-product-branch-merge-decision-log.md) (the keystone); open a dated plan on this board before editing. | Product scope, not defects: the owner sets priority. Each slice record carries the verbatim promise, today's tree state and a bounded fix prompt. |

**Status meanings:** In flight = active work; Pending = not started; Finished
= recorded scope and verification complete in the local worktree; Completed =
a checkpoint commit exists; Blocked = work cannot safely progress. Because
several tasks share documentation files, do not use `git add .`; review and
stage the appropriate hunks for one task at a time.

Unscheduled limitations and debt stay in [`docs/TODO.md`](../TODO.md) and
[`docs/issues/`](../issues/README.md). Promote one to this board when it is
selected for work; backlog is not in-flight work.

## Finished work

The board holds only In flight, Pending and Blocked rows (and a row whose
next action is still open). When a row reaches Finished, Completed, Fixed,
Closed, Delivered or Superseded, move it verbatim to the month file for its
plan's date under [`index/`](index/README.md), adding one `../` to its
relative links: [November 2026](index/2026-11.md),
[October 2026](index/2026-10.md), [September 2026](index/2026-09.md),
[August 2026](index/2026-08.md), [July 2026](index/2026-07.md).
