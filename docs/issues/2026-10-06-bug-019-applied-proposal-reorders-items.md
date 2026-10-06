# BUG-019 — accepting a proposal applies its items out of order

## Symptom

An agent proposal listed four structure items in this order: Intake,
Verification, Handoff, Archive. After a human accepted it with
`--apply`, the roadmap positions were Intake, Handoff, Verification,
Archive. Seen 2026-10-06 on the live sim server. The titles all
landed; only the order changed.

## Status

Fixed 2026-10-06. `proposal_items` gained a `position` column
(migration `20261006120000`), set to the order the items were given,
and the list orders by it. `TestProposal_ApplyKeepsItemOrder` proposes
Intake, Verification, Handoff, Archive and reads them back in that
order. The sim database was migrated; Chen's already-applied nodes
were left as they landed.
