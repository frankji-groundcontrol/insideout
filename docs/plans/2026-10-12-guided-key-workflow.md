# Complete guided key setup and verification (2026-10-12)

Goal: a newcomer can follow installation → key walkthrough → each of the
31 mapped key references → repeatable verification, with prerequisites,
expected results, failure remedies, and an explicit optional/default lane.

Preserve the earlier uncommitted remote-auth slice. This task updates the
documentation and repairs secret-bearing test diagnostics; it does not rotate
live credentials, change deployed settings, or add a new configuration UI.

## Execution

Live coordination state: [track YAML](2026-10-12-guided-key-workflow.track.yaml).

- [x] Review the plan against the real commands and consumers before editing.
- [x] Write the shared walkthrough and connect installation, SETENV, usage,
      environment, root README, and the installation skill.
- [x] Expand all 31 key docs with actionable setup, safe test commands or
      browser actions, expected outcomes, troubleshooting, and correct links.
- [x] Fix key validation diagnostics red-first so failed assertions cannot
      print credentials; handle the observed terminal-newline failure honestly.
- [x] Exercise the smallest real database/JWT path using a disposable cluster;
      verify documented commands and clearly separate checked commands from
      provider flows requiring an operator's account.
- [x] Review once, repair observed defects, run link/coverage/privacy/test gates.
- [x] Update dated changelog, learning, issue disposition, and indexes.

## Engineering review contract

Keep the common environment/store steps in one tutorial; key files link down
to exact procedures and remain the reference authority. Avoid shell argv
credentials, response bodies containing tokens, and test assertions that
render secret operands. A 401 on an unauthenticated route proves routing only,
not database health or a usable credential. Provider verification must prove
the capability the app needs (mail delivery, a coach turn, signed webhook,
private-repository access). Stored-but-unused credentials get placement-only
acceptance, with no invented live consumer. Runtime settings must test their
observable effect rather than just their file's shape.

## Acceptance evidence

No mocks were accepted as live verification. An isolated real PostgreSQL 18
cluster with synthetic roles applied all 38 migrations and passed `TestAuthz`;
the role gate now detects a missing owner `BYPASSRLS` before migration. The
49-test key suite, 48 environment-catalog checks, 37 environment-write checks,
docs-truth, redaction, and diff checks all pass. Provider-backed capability
checks remain operator-owned and are described with their prerequisites in the
testing tutorial. No real `.env` file was printed or inspected.
