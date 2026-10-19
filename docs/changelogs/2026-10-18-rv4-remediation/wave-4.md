# Wave 4 — integrated 2026-10-20

Local branch `rv4/integration` only. Nothing was pushed, merged to `main`,
or deployed.

## What landed

21 clusters, 174 cherry-picks, no conflicts, then two pre-gate fixups.
Head `97cf4c06`. That is 499 commits above `main` @ `d53035a`.

The first 18 clusters were accepted by the earlier verifier rounds
(reports `verdict-w4-*-rN.json`). The Claude run stopped on an API 503
with three clusters unfinished. This resume finished them and re-checked
red against green before the cherry-pick:

| Cluster | Head | Check |
| --- | --- | --- |
| `w4-rv-deploy-truth` | `79ebba46` | `check_proxy_hops_rv4.py` exits 1 at `e5d00743` and 0 at the head |
| `w4-copy-sweep` | `39348693` | `check_first_run_copy.py` exits 1 at `39e540b5` and 0 at the head |
| `w4-rv-mcpserver` | `f2af0bd9` | `TestRvMcp` fails at `26dd530f` and passes at the head on private Postgres |

Those three checks were a coordinator re-run, not a separate mutation
verifier. Scope of each diff stayed inside the cluster's owned files.

## Pre-gate fixups

- `dd773379` — `coverage_session_test.go` stops expecting the reuse-grace
  replay to clear the winner's cookies (`rv4-FR-F100`).
- `97cf4c06` — the real-API MCP contract passes `confirm=true` to
  `auth_logout` (`rv4-FR-F176`).

## Gates on `97cf4c06`

- `go test ./... -race -count=1` with `DATABASE_URL` on a private Postgres
  18 cluster: exit 0. The same suite had failed once, only on
  `TestCLIAndMCPAgainstTheRealAPI/MCP_process_that_signs_itself_in`; the
  second fixup was re-run as the full suite.
- `flutter analyze --no-fatal-infos`: 14 infos, exit 0.
- `flutter test` and `TZ=Asia/Shanghai flutter test`: both passed (575).
- 19 `scripts/tests/check_*.py` guards, `check-redaction.sh`, and
  `check-docs-recorded.sh`: exit 0.
- Port 55499 was torn down afterwards.

Workflow files were not executed on a runner. Those properties stay
pending the first CI run after a push.

## Register dispositions

`dispositions.py --lists` against this head, after the three verdict
files were written: 92 FIXED, 16 PARTIAL, 6 DEFERRED_OWNER, 1 NOT_REAL
(`rv4-FEAT-05`, convert half already on the base), 0 BLOCKED, 30
not-in-scope. The whole-change findings (`rv4-FR-*`) are worker rows
beside the register, not register IDs.

## Still open

Prose and index to-dos named in the worker fragments are not all applied.
The follow-up the previous coordinator called wave 4b is not started:
`ci.yml` fetch-depth 0 for the ledger-honesty arm, the findings-register
deploy-id scrub together with `QUOTES_EVIDENCE`, the agent-wiring edits
for `rv4-FR-F94` / `rv4-FR-F86`, and the wider `INSIDEOUT_TEST_STRICT`
sweep. Owner questions stay deferred (retention, history rewrite, hosted
migration cutoff, the invitation-request route, project handover, the
session-swap withhold).
