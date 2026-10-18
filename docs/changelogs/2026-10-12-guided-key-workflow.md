# Complete guided key setup and verification workflow

Installation now routes through two central tutorials: [key setup](../usage/key-setup-tutorial.md)
walks from `.env` and database roles through the host key store, and [key
testing](../usage/key-testing-tutorial.md) separates shape, reachability, and
capability evidence. The installation guide, SETENV, usage index, README, and
installation skill link to the same flow. All 31 key references retain the
same six sections and now include an actionable set step, a safe verification
step, expected evidence, and a failure remedy.

The key store now declares valid empty lanes for offline coach, mail dev mode,
and disabled GitHub App/webhook integrations. Per-key tests accept only an
empty optional value (with one editor-added final newline); malformed non-empty
values still fail. Test assertions no longer render secret operands, and a
regression test covers that diagnostic boundary.

Own-URL setup now probes the required role contract before migrating. An
existing `insideout_owner` without `BYPASSRLS` stops with a safe remediation
before any migration or environment write, avoiding the PostgreSQL recursive
policy error. The documented owner/app contract is `BYPASSRLS` for the
NOSUPERUSER owner and `NOBYPASSRLS` for the runtime app role.

Verification:

- `python3 scripts/test_env_catalog.py` — 48 passed
- `python3 scripts/test_env_writes.py` — 37 passed
- key suite — 49 passed
- `go test -C server ./internal/setup/... ./cmd/insideout -count=1`
- setup health checks now treat the default `messages` schema as shape-only;
  the legacy `openai` compatibility probe is opt-in
- disposable PostgreSQL 18 with synthetic owner/app roles — all 38 migrations
  applied and `TestAuthz` passed
- disposable setup with an owner lacking `BYPASSRLS` — stopped before migration
  with an opaque remediation
- docs truth, redaction, and `git diff --check` passed

Provider-backed coach, mail, and GitHub proofs still require the operator's
own accounts and isolated test resources; the tutorials identify those
prerequisites and side effects explicitly.
