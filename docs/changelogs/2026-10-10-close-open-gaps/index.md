# 2026-10-10 — Close the open gaps

Burn-down of every open issue record and board item
([plan](../../plans/2026-10-10-close-open-gaps/2026-10-10-close-open-gaps.md)).
Each child file records one wave: what changed, how it was verified, and what
an operator needs to do.

| Wave | Record | State |
| --- | --- | --- |
| CI and host fixes | [ci-and-hosts.md](ci-and-hosts.md) | landed; CI green on `8ee97fc` |
| Backend file splits | [backend-splits.md](backend-splits.md) | landed |
| Setup: database roles and the bundled owner DSN | [setup-roles-and-bundled-owner.md](setup-roles-and-bundled-owner.md) | landed |
| Backend fixes | [backend-fixes.md](backend-fixes.md) | landed |
| CI gaps and the Ubuntu install skill | [env-ci-and-skill.md](env-ci-and-skill.md) | landed |
| Client file splits | [client-file-splits.md](client-file-splits.md) | landed |
| Client fidelity fixes | [client-fidelity-fixes.md](client-fidelity-fixes.md) | landed |
| Web walk and phone layouts | [web-walk-and-phone-layouts.md](web-walk-and-phone-layouts.md) | landed |
| The commit card shows what a version recorded | [commit-card-details.md](commit-card-details.md) | landed |
| Web walk, second pass: both themes, the keyboard, two more defects | [web-walk-keyboard-and-themes.md](web-walk-keyboard-and-themes.md) | landed |
| The app's nginx follows its API upstream | [nginx-upstream-resolution.md](nginx-upstream-resolution.md) | landed; live after an `app` deploy |
| Review-register sweep: the server | [review-sweep-server.md](review-sweep-server.md) | landed; see the [disposition ledger](../../review/2026-10-10-disposition-ledger/README.md) |
| Review-register sweep: the client | [review-sweep-client.md](review-sweep-client.md) | landed |

Nothing here is deployed. The hosted bundle and server still predate the
2026-10-08 slices; shipping stays an explicit owner step
([HANDOFF](../../HANDOFF.md)).

*Superseded 2026-10-11: the deploy shipped — both services run `main` @
`b6c34c4` / `6b1f24f` and the hosted database reports "already up to date"
([owner-deploy record](../2026-10-11-owner-deploy-and-mail-truth.md);
[HANDOFF](../../HANDOFF.md)).*
