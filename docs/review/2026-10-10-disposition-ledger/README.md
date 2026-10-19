# 2026-10-10 — Review-register disposition ledger

What happened to every finding the three whole-tree reviews filed that no
issue record, plan, or changelog traced. The registers:

- [Multirole review 1 (2026-10-04)](../2026-10-04-multirole/01-findings-register.md)
- [Multirole review 2 (2026-10-04)](../2026-10-04-multirole-2/01-findings-register.md)
- [Frontend Cobalt review (2026-10-06)](../2026-10-06-frontend-cobalt/01-findings-register.md)
- [Multirole review 3 (2026-10-10)](../2026-10-10-multirole-3/01-findings-register.md)
  — executed directly as four remediation waves; rows are traced in
  [04-multirole-3.md](04-multirole-3.md).

## Method

Each untraced finding was triaged against the code at the wave base (fixed,
parked, bounded, owner step, or not a defect); every FIXED and NOT-A-DEFECT
call was then independently re-checked by a second agent against the code, and
the PARTIAL re-checks were re-fixed test-first in round 2. Fixes land with the
red test committed first; independent verifiers reproduced red/green and
mutation-checked each batch before it was integrated. The wave changelogs are
[review-sweep-server.md](../../changelogs/2026-10-10-close-open-gaps/review-sweep-server.md)
and
[review-sweep-client.md](../../changelogs/2026-10-10-close-open-gaps/review-sweep-client.md).

## Statuses

| Status | Meaning |
| --- | --- |
| Fixed in this wave | Proved red, fixed, verified by an independent verifier in this sweep |
| Fixed before this wave | The triage found the closing change already on `main` |
| Fixed (code half) | The code half landed and is verified; the remaining half needs an owner action (register 4) |
| Fixed | Register 4: a commit changed code, config or a script (or the owner closed the live step), and the row names it |
| Recorded | Register 4: the fix commit changed only `docs/` — the gap or claim is written down, not closed; the row says where it stays open |
| Satisfied by the report | No code was owed — the report or plan itself answers the row (register 4) |
| Not a defect | Re-proved at the base and did not reproduce for the stated reason |
| Parked feature | A product decision parks it (PRODUCT.md / a plan names it) |
| Owner | Only the owner can close it (credential, deploy, or product call) |
| UNRESOLVED | Should not exist in the final ledger; the wave is not done while it does |

Register 4's Fixed/Recorded split and its `Counts` line are held by
`scripts/tests/check_ledger_honesty.py`.

## Files

- [01-multirole-1.md](01-multirole-1.md) — register 1 rows
- [02-multirole-2.md](02-multirole-2.md) — register 2 rows
- [03-frontend-cobalt.md](03-frontend-cobalt.md) — frontend register rows
- [04-multirole-3.md](04-multirole-3.md) — register 3 rows (four remediation waves)
