# 25 · Closure audit

**Lens:** review-3 High closure verification, review-3 Medium sampling, 2026-10-04 High spot-check, red-test mutation checks, CI wiring of guards, ledger accuracy, L1 sibling search  
**Score:** **7.4 / 10**  
**Bottom line:** Most review-3 High closures are real: 14 of 17 have the fix at the cited location and a mutation-killable test. The weak points are the accounting and four unwired guard scripts: about 11 'Fixed' ledger rows are docs-only commits, and four check_*.py scripts cited as red tests run nowhere in CI. FEAT-01, USER-01 and REL-01 are only partially closed.

Tree: `main` @ `9f0b1c7`. Every High this lens filed went to an adversarial skeptic cluster; where the skeptic changed a severity or corrected a claim it is marked **[Correction]** below. The full register is [01-findings-register.md](01-findings-register.md).

## Skeptic verdicts on this lens's Highs

| ID | Filed | Verdict | Final | Reason |
|---|---|---|---|---|
| CLOSE-01 | High | DOWNGRADED | Medium | **[Correction]** Facts hold, severity is overstated. A repo-wide grep outside docs/ finds no reference to check_go_floor, check_docs_truth, check_contribution_surface or check_product_promise_gaps in .github/, config/git-hooks/, scripts/*.sh or any Go test; they appear only in their own Usage lines and in docs. The only python3 -I guards CI … |

---

## Scope

Role: closure-audit (CLOSE-NN). Tree: clean detached worktree at main @ 9f0b1c7. I read the review-3 register (83 rows: 17 High, 33 Medium, 33 Low) and the disposition ledger `docs/review/2026-10-10-disposition-ledger/04-multirole-3.md`. I checked all 17 Highs at the quoted file:line. For every claimed red test I confirmed it exists, is not skipped, and fails when the fix is reverted. I did this by mutating a scratch copy of `server/` against a private Postgres, which is now stopped. I also sampled 11 Mediums and spot-checked the 2026-10-04 Highs. The L1 time bomb (`TestProjectQuietFlagged`) is the coordinator's and is not re-filed. Hosted-deploy claims (SEC-01 rollout) cannot be verified offline and are marked as such.

## Scores

| Aspect | Score | Why |
|---|---|---|
| Code closure of Highs | 8.5 | 14 of 17 verified true with mutation-killable tests; three are partial or unverifiable |
| Test honesty | 7.5 | The cited Go tests really fail on revert; four guard scripts cited as "red" run nowhere in CI |
| Ledger accuracy | 6.0 | About 11 "Fixed" rows are docs-only commits; some prose contradicts the code |
| Regression resistance | 7.0 | Upgrade-replay lane is strong but its cutoff is stale |
| **Overall** | **7.4** | Mostly honest closures; the weak spots are in how the ledger counts and in guards nobody runs |

## What is good

- `server/internal/agent/critic_merge.go` with `critic_capture_test.go`: the AIDEV-01 fix (`mergeRankedCritique` at `critic.go:122`) is real. Reverting it to `merged := findings` fails the test.
- `server/scripts/upgrade-replay.sh` with `upgrade-seed.sql`: TEST-01 is closed by a real replay of deployed-state data through the new migrations, not a fresh-schema run.
- `server/internal/agent/template_offline_test.go` plus `client/test/coach_panel_offline_test.dart`: the offline disclosure (FEAT-02) is covered on both server and client, and `sse.go` carries `offline` in `message_start`.
- `server/internal/agent/coach_complete_test.go`: the `endsWithMarkerLine` completion check kills the `Contains` mutation.
- `server/internal/api/single_replica_test.go`: the singleton guard exists and is tested for the case it was written for.

## Findings (worst first)

**CLOSE-01 (High) - Guard scripts cited as "red" are not wired into any automation.**
`scripts/tests/check_go_floor.py`, `check_docs_truth.py`, `check_contribution_surface.py` and `check_product_promise_gaps.py` pass when run by hand. None of them is referenced by `.github/workflows/`, `config/git-hooks/` or `scripts/*.sh` (grep returned nothing). Only `check_smoke_wiring.py` is wired (via `smoke.yml`). Evidence: `.github/workflows/ci.yml:316: run: sh check-redaction.sh` and `ci.yml:210: - run: python3 -I scripts/check_nginx_reresolve.py` are the only script guards CI runs. The ledger counts these scripts as the "red test" for their rows. Scenario: someone lowers the Go floor or breaks a docs claim; every gate stays green and the owner reads "closed with a guard". Fix: add one CI step that runs all `scripts/tests/check_*.py` with `python3 -I`. diff_size: about 6 lines of YAML. first_check: `grep -rn check_go_floor .github config scripts`.

**CLOSE-02 (Medium) - FEAT-01 is closed on the web only; CLI and MCP never send a locale.**
`server/internal/apiclient/prd.go:23: if err := c.do(http.MethodPost, "/ideas/"+ideaID+"/convert", map[string]any{}, &out); err != nil {`. The empty body means CLI- and MCP-converted ideas get the stored default (`zh`) coach language regardless of the idea's language. Scenario: an English idea converted through MCP gets a Chinese-default conversation. Fix: add a `locale` argument to the CLI/MCP convert tools and pass it through. first_check: `grep -rn locale server/internal/apiclient server/internal/mcp`.

**CLOSE-03 (Medium) - USER-01/AIDEV-07 are only partially closed: decline is not durable.**
`client/lib/features/prd/prd_page.dart:214: // Decline stays local — the endpoint expresses confirm only`. Confirm and the parked `resolve_finding` resolutions are server-side (`stage_confirm_apply.go`), but a decline is a client-only state. Scenario: the user declines a stage, reloads, and the decline is forgotten while findings stay parked. Fix: add a decline endpoint or record decline on the stage row; until then mark the rows Partial.

**CLOSE-04 (Medium) - The ledger header overstates closure.**
It reports "73 closed by code (70 Fixed + 3 code-halves)", yet these "Fixed" rows have fix commits that touch only docs: CQ-02 (5af2c94), FEAT-05 (3bd86f0), FEAT-06 (71d884d), FULL-03 (d4598ac), SEC-04 (4a7cb5b), USER-04 (58ec57b), COMP-01 (1c26951), COMP-04, PM-03, REL-04, REL-05 (787c8ee). Example: `58ec57b` is "docs(issues): tokenized audience-share link recorded (USER-04)" with 2 doc files changed; REL-05's fix states "nothing runs backup-db.sh on any schedule (no cron, no CI job)". Recording a gap is legitimate disposition, but it is not "closed by code". Scenario: the owner believes backups and audience sharing are handled. Fix: relabel these rows "Recorded / Owner decision" and recompute the header. first_check: `git show --stat <sha>` for each docs-only row.

**CLOSE-05 (Medium) - The singleton guard is a one-shot check.**
`server/internal/api/single_replica.go` takes the advisory lock at start. If the lock connection drops, nothing re-checks and a second replica can start later. REL-01's test covers only the start-time collision. Fix: hold the lock on a dedicated connection and re-acquire or exit on loss. diff_size: about 30 lines.

**CLOSE-06 (Low) - AIDEV-09 is copy-only.** The closure changed wording, not the trust-layer behavior the finding described. Mark Partial.

**CLOSE-07 (Low) - FEAT-04 has no CLI/MCP exposure.** `POST /api/v1/roadmap/{nid}/evidence` and `/validate` exist on the API and web only, so agents using MCP cannot validate roadmap nodes.

**CLOSE-08 (Low) - TEST-06 still shares one breaker row.** `server/internal/agent/breaker_pins_test.go:32: ... FROM insideout.ai_circuit_breaker WHERE id = 'anthropic'`. The wiring tests stay sequential by design but are not isolated.

**CLOSE-09 (Low) - Stale replay cutoff.** `server/scripts/upgrade-replay.sh:33: DEPLOYED_THROUGH=20261006120000`. Later migrations are replayed from the seed, so the lane no longer models what is actually deployed.

**CLOSE-10 (Low) - Changelog contradicts migration.** `server/db/migrations/20261011110000_conversation_locale.sql:9: ADD COLUMN locale text NOT NULL DEFAULT 'zh';` against `docs/changelogs/2026-10-11-rv3-remediation/index.md:35: existing rows default to `en``. Existing rows are `zh`.

**CLOSE-11 (Low) - Marker equality is brittle.** `endsWithMarkerLine` matches `NO OPEN QUESTIONS` exactly, so a model adding punctuation or a trailing note silently fails to complete the coach flow. Accept a trimmed, case-insensitive match.

## Lead verification

| Lead | Verdict | Evidence |
|---|---|---|
| L1 time bomb | Extended: no sibling | Go tests inject clocks or use relative times; Dart fixtures are now-relative. No other wall-clock-dependent test found. |
| L2 guard scripts unwired | Confirmed | CLOSE-01; four of five scripts unreferenced. |
| L3 docs-only "Fixed" | Confirmed | CLOSE-04; 11 rows with docs-only commits. |
| L11 locale coverage | Extended | Web fixed; CLI/MCP gap (CLOSE-02) and 'zh' vs 'en' default (CLOSE-10). |
| L16 decline durability | Confirmed | CLOSE-03. |
| L17 singleton guard | Confirmed | CLOSE-05. |
| L18 breaker isolation | Confirmed | CLOSE-08. |
| 2026-10-04 Highs (AIDEV-09, USER-09, USER-11, REL-13) | Holds, except AIDEV-09 is copy-only | USER-09 drop confirm, USER-11 error mapping and the rightmost-XFF trust in `auth_throttle.go` / `client/nginx.conf` are in place. |

## Closure table

| Finding | Claimed commit | Code verified | Test verified | Verdict |
|---|---|---|---|---|
| AIDEV-01 | per ledger | yes | yes (mutation killed) | True |
| AIDEV-05 | per ledger | yes | yes | True |
| AIDEV-02 | per ledger | yes | yes | True |
| ARCH-01 | per ledger | yes | yes | True |
| FEAT-01 | per ledger | web only | yes | Partial (CLOSE-02) |
| USER-03 | per ledger | yes | yes | True |
| FULL-01 | per ledger | yes | yes | True |
| REL-01 | per ledger | start-time only | yes | True for start; Partial (CLOSE-05) |
| FEAT-02 | per ledger | yes | yes (Go + Dart) | True |
| TEST-01 | per ledger | yes | yes | True (cutoff stale, CLOSE-09) |
| TEST-02 | per ledger | yes | yes | True |
| SEC-01 | per ledger | tests yes | yes | True in code; deploy unverifiable offline |
| USER-01 | per ledger | confirm only | yes | Partial (CLOSE-03) |
| FEAT-04 | per ledger | API only | yes | Partial (CLOSE-07) |
| COMP-01 | 1c26951 | docs only | none | Recorded, not code-fixed |
| FEAT-03, USER-02, PM-01 | per ledger | yes | yes | True |
| Mediums sampled (REL-02, REL-03, SEC-02, SEC-03, SEC-06, SEC-07, AIDEV-04, AIDEV-06, AIDEV-10, AIDEV-11, AIDEV-12) | per ledger | yes | yes, pass on private PG | True |
| REL-05, CQ-02, FEAT-05, FEAT-06, FULL-03, SEC-04, USER-04, COMP-04, PM-03, REL-04 | docs-only | no code | none | Recorded only (CLOSE-04) |

## Recommendations

1. Wire every `scripts/tests/check_*.py` into `ci.yml` (CLOSE-01).
2. Relabel the docs-only "Fixed" rows and recompute the ledger header (CLOSE-04).
3. Add `locale` to the CLI/MCP convert path (CLOSE-02).
4. Persist stage decline server-side, or mark USER-01/AIDEV-07 Partial (CLOSE-03).
5. Make the singleton guard hold its lock continuously (CLOSE-05).
6. Advance `DEPLOYED_THROUGH` to the real deployed migration and add it to the release checklist (CLOSE-09).
7. Correct the changelog default-locale sentence and loosen marker matching (CLOSE-10, CLOSE-11).
8. Expose roadmap evidence/validate through CLI and MCP (CLOSE-07).

## What this role would do first

Wire the Python truth scripts into CI. It is the smallest change (a few YAML lines) and it turns a dozen "passes by hand" claims into guards that actually run. Then relabel the ledger so that every row marked Fixed means a code change plus a red test.

## Stop trusting

- Any "red" citation that names a `check_*.py` script (other than the smoke-wiring check): it never runs in automation.
- "Fixed" on the docs-only rows listed in CLOSE-04, especially REL-05 (backups are not scheduled) and USER-04.
- FEAT-01 as fully closed: CLI and MCP still skip locale.
- USER-01/AIDEV-07 as human-authored end to end: decline is local.
- REL-01 as continuous protection: it checks at start only.
- AIDEV-09 as a trust-layer fix: it is copy only.
- TEST-06 as de-shared: the breaker row is still shared.
- The changelog statement that existing rows default to `en`: the migration sets `zh`.
