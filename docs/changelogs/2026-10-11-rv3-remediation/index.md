# 2026-10-11 — Review-3 remediation: four verified waves

Execution of the [review-3 remediation plan](../../review/2026-10-10-multirole-3/40-remediation-plan.md)
([plan record](../../plans/2026-10-10-rv3-remediation.md)): all 83 register
rows dispositioned — 73 closed by code (70 Fixed + 3 code-halves), 3 satisfied
by the report, 1 parked, 6 owner-blocked. Every finding's trace is in the
[disposition ledger](../../review/2026-10-10-disposition-ledger/04-multirole-3.md).

Each cluster worked red-test-first in its own worktree; an independent
adversarial verifier reproduced red and green, re-ran the gates and
mutation-checked before acceptance. All 32 clusters were accepted in round 1.

| Record | Waves | State |
| --- | --- | --- |
| [server.md](server.md) | 1, 2, 4 | landed — API, store, agent, three migrations |
| [client.md](client.md) | 2, 3, 4 | landed — coach panel, capture, polish, validation leg |
| [docs-and-infra.md](docs-and-infra.md) | 1, 3, 4 | landed — CI lanes, truth checks, doors, promise records |
| [verification.md](verification.md) | all | gates green on a fresh database; one allowed skip |

## Operator notes

- **Seven migrations are now pending on the hosted database**: the four
  `20261010*` files plus `20261011100000_circuit_breaker_row_id`,
  `20261011110000_conversation_locale`, `20261011140000_roadmap_validation`.
  All seven apply together on a fresh clone of the migrated base
  ([verification.md](verification.md)); the first hosted apply is the owner's
  SEC-01 step, now rehearsed by CI's upgrade-replay lane on production-shaped
  seed data.
- `20261011100000` **drops and recreates** `ai_check_circuit` /
  `ai_record_circuit_result` with a row-id parameter — a plain `CREATE OR
  REPLACE` would have left an ambiguous overload (see the
  [learning record](../../learning/2026-10-11-create-or-replace-with-a-new-signature-overloads.md)).
  Nothing calls the old zero-arg forms.
- The conversation locale (`20261011110000`) and roadmap validation
  (`20261011140000`) columns are additive; existing rows default to `en` and
  no validation state respectively.
- Nothing here is deployed. Shipping remains the owner's explicit step
  ([HANDOFF](../../HANDOFF.md)); the six owner rows are enumerated in the
  [promise register](../../review/2026-10-10-multirole-3/50-promise-register.md).

*Superseded 2026-10-11: shipped — `main` @ `b6c34c4` deployed to both
services; the seven migrations applied at boot and the hosted database
reports "already up to date" ([owner-deploy
record](../2026-10-11-owner-deploy-and-mail-truth.md);
[HANDOFF](../../HANDOFF.md)).*
