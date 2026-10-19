# Review-4 remediation, wave 3: the per-ID dispositions

The per-finding table of [wave 3](wave-3.md), kept in its own file so that both
records stay inside the line budget. Each cluster's own account (what changed,
the evidence, the ceilings) is its fragment, linked from the
[folder index](index.md).

## How to read it

Status is the engineering status. Every `FIXED` has a red test that failed before
its fix and passes after it (the verifiers replayed each pair at the worker
SHAs; the integrator ran the whole suite at the head `3a1c4ac`), with two
exceptions: `rv4-TEST-03` is a coverage task whose red is the measured baseline
(API coverage 63.0% to 72.3%, plus two mutation checks), and `rv4-CLOSE-10` was
closed by the integrator's pre-gate patch. Six rows say `-` in the Red column:
those two, `rv4-TEST-04` (test-only commits) and the three `DEFERRED_OWNER`
rows (`rv4-DATA-06` is a decision record; `rv4-TEST-07` and `rv4-CLOSE-09` share
one comment). SHAs are on `rv4/integration`; the fragments cite
the worker-branch SHAs, and each integration commit is patch-identical to its
worker commit (its body names the original in the `cherry-pick -x` line).

A row that rests on a workflow file or a CI-config property is verified here only
by hand and by the guards, so it is **pending first CI run after push**: the
release gate (it needs `actions: read`), the attestation, the no-overwrite
publish, `security.yml` (govulncheck, gitleaks), dependabot, the `ci.yml` token
narrowing and Go-floor derivation, `ubuntu-install.yml`, the client web build
flag and the nginx checker. That is residue R1-17; it is not repeated in every
cell below.

The R3 IDs are the rows of the wave-3 residue table in the
[open-findings record](../../issues/2026-10-18-review-4-open-findings.md#wave-3-open-residue-integrator-2026-10-19),
which carries one line and a fix prompt for each. They are not numbered in plan
order: the test-hardening, first-run-copy and docs-truth rows come after the
others. R1 and R2 IDs are the earlier residue tables.

## Dispositions

| Finding | Cluster | Row | Status | Red | Fix | Left open |
| --- | --- | --- | --- | --- | --- | --- |
| rv4-AIDEV-02 | coach-trust | 3.1 | FIXED | `493ed84` | `7628169` | The finalize gate is kept on purpose: a skipped critic still reaches finalize, but the stream, the view and the finalize card now say no review ran. `DEFERRED_OWNER`: whether to refuse while a skip is set, or only on contention, is one condition plus its pinning test (owner decisions) |
| rv4-AIDEV-03 | coach-trust | 3.1 | FIXED | `da550d4` | `2a7a66b` | none |
| rv4-AIDEV-05 | coach-trust | 3.1 | FIXED | `36dba2d` | `facd204` | Deployment risk: a gateway that never sends `message_stop` or `response.completed` now fails every turn, and no real provider was driven (R3-02). The Responses stub in `llm_factory_test.go` lacks `response.completed` and waits about 2 s on a retry (R3-03) |
| rv4-AIDEV-06 | coach-trust | 3.1 | FIXED | `8214cba` | `3a0947e` | A dropped parallel call is silent; the Responses forced-tool critic path is unit-tested only (R3-04) |
| rv4-AIDEV-07 | coach-trust | 3.1 | PARTIAL | `9a22b75` | `8b3a9b7` | Conversation completion still hangs on the `NO OPEN QUESTIONS` marker line, not on server state; the quote floor (12 characters) and the text-support floor (0.2) are tunables calibrated on 39 hand-written pairs (R3-01) |
| rv4-CLOSE-03 | coach-trust | 3.1 | FIXED | `ff94752` | `d0ab02d` | New route `POST /api/v1/conversations/{id}/stage/decline` (human only). A second tab keeps its card and gets a 409; an agent refusal is covered by the route sweep only (R3-04) |
| rv4-LEAD-FENCE-DEDUP (R2-08) | coach-trust | 3.1 | FIXED | `f893a32` | `3f7ba17` | none; R2-08 is closed (`agent/prompt_fence.go` now delegates to `agentcontext.FenceData`) |
| rv4-REL-01 | stream-bounds | 3.2 | FIXED | `66e3434` | `0da4148` | Accepted limits (R3-08): no instance-wide stream cap, one limiter entry per user who ever posted a cursor. A refusal is 429 `APP_STREAM_LIMIT` (20 per user, 100 per project); cursor posts are 25 per second per user |
| rv4-REL-02 | stream-bounds | 3.2 | FIXED | `618859c` | `87d8ce2` | A plain route that reaches the 25 s deadline answers 500, not 503; seven streaming or long-running routes are exempt (R3-08). `healthz` answers 503 after 2 s |
| rv4-REL-03 | stream-bounds | 3.2 | FIXED | `227d3bf` | `769366d` | Membership is rechecked on the 10 s heartbeat with a 5 s bound and fails open on a database error, so a removed member can read for up to 10 s (R3-08) |
| rv4-REL-04 | stream-bounds | 3.2 | FIXED | `2eb8d3d` | `3326cec` | none. The second instance now logs a WARN, retries every 1 s for 2 minutes and logs an INFO on takeover; the ERROR comes only after the window (operator notes) |
| rv4-SEC-08 | stream-bounds | 3.2 | FIXED | `0c6ae16` | `49f6f25` | A stream now ends at access-token expiry (15 minutes), so it is usable only together with FULL-04, which reopens it (R3-07) |
| rv4-FULL-04 | stream-bounds | 3.2 | FIXED | `b98b11e` | `32f29f8` | Covered by a Dart VM test with a scripted transport; no browser drives a mid-stream 401 (R3-07) |
| rv4-COMP-04 | stream-bounds | 3.2 | FIXED | `ac40f8c` | `0aaac09` | `update_prd_section` through the agent has no cap and `get_prd` returns sections whole (R3-05); the web editors have no `maxLength` and rows stored above the cap stay whole (R3-06). The API refuses a section over 10,000 characters with a 400 |
| rv4-SEC-02 | auth-network | 3.3 | FIXED | `7545e09` | `bc42da5`; guard repair `46f252b` | `docker compose config` was not run, docker is not installed here (R3-12). Hosted must set `INSIDEOUT_TRUSTED_PROXY_HOPS=1` before it deploys (R3-13) |
| rv4-SEC-03 | auth-network | 3.3 | FIXED | `aa317b3` | `6ec3d8e`; typo `aacd86a` | Register still answers 409 for a taken address, on purpose; the owner may read SEC-03 as PARTIAL (owner decisions). The mail queue is in memory: a queued mail is lost on exit and a full queue drops the newest (R3-10) |
| rv4-REL-07 | auth-network | 3.3 | FIXED | `aa317b3` | `6ec3d8e` | Same commits as SEC-03. Measured: a known address answered after about 202 ms against 0.9 ms for an unknown one; with the queue the gap is at most 0.06 ms. Same queue limits (R3-10) |
| rv4-SEC-04 | auth-network | 3.3 | PARTIAL | `a73f232` | `48a1896` | A spent refresh token now ends the whole session family after a 10 s grace. `iss`/`aud` claims and a logout denylist in `auth/jwt.go` are open, so an access JWT outlives a family kill by up to 15 minutes (R3-09); a refused refresh still clears the auth cookies, so a browser tab that loses a rotation race can clear the cookie the winner just set (R3-11) |
| rv4-PUB-03 | release-supply | 3.4 | PARTIAL | `4e1f4b1` | `b242e09` | Pending first CI run after push: the gate on a green `ci` run for the exact SHA, the attestation and the no-overwrite publish. `bootstrap.sh` does not yet require https for `INSIDEOUT_RELEASE_BASE` or verify the attestation (R3-14). The version policy and rollback section (PM-05) moved to the proposed `w4-release-version` (R3-18) |
| rv4-PUB-04 | release-supply | 3.4 | FIXED | `bd2bcdd` | `a30f09e` | Packaging was run by hand: four tarballs holding `insideout`, `insideout-mcp`, `LICENSE` and `THIRD_PARTY_NOTICES.md`, the version stamped, four checksums. The workflow run itself is pending first CI run after push |
| rv4-PUB-07 | release-supply | 3.4 | FIXED | `347920b` | `d2620e9`; `331a5f3` | Pending first CI run after push. Seven historical gitleaks findings need an allow-list or a rotation owner (R3-16); the pre-commit redaction hook only warns (R3-17); `ci.yml` is 349 of 350 lines (R3-20, R1-01) |
| rv4-SEC-07 | release-supply | 3.4 | PARTIAL | `4191157` | `47e748c` | Every Action reference is pinned by full commit SHA (23 of 25 were tags); pending first CI run after push. Docker base images are not pinned by digest and dependabot has no docker ecosystem (R3-15); the pins are literals, so a Go or Flutter bump is a lockstep edit (R3-19); `bootstrap.sh` (R3-14) |
| rv4-PUB-09 | release-supply | 3.4 | FIXED | `00c488b`, `1d5bf61` | `d07d8b3`, `2bd00ba` | Go and Flutter archives are checksum-verified before they are unpacked, a wrong Flutter hash stops the whole block, and the CI Go floor comes from `go.mod`; pending first CI run after push. The pins are literals (R3-19) |
| rv4-PUB-02 | privacy | 3.5 | PARTIAL | `e1f9494` | `b80d01a`, `a4374fe`, `384fb9e` | `PRIVACY.md` is published and linked from the sign-in and sign-up door, but the web link copies the notice address instead of opening a tab; the opener is the proposed `w4-privacy-opener` (R3-21). No guard ties the processor list to the variables (R3-23) |
| rv4-DATA-06 | privacy | 3.5 | DEFERRED_OWNER | `-` | `c4823d9` (decision record) | Five owner questions in the retention and erasure record: the retention window, the erasure rule, self-service delete, export before erasure, and who is `<privacy-contact>` and how fast they answer. The reaper cannot purge conversations as built (R3-22) |
| rv4-COMP-08 | docs-truth | 3.6 | FIXED | `95e0817` | `c96ff59`; `2c116c8` | Every `PARTIAL` promise row has a dated issue and the docs say PostgreSQL 18. Folded `COMP-08-DECISION`: should `in_progress` to `done` need linked evidence or a short reason, enforced in `UpdateRoadmapNode`? An owner question (owner decisions) |
| rv4-FULL-09 | docs-truth | 3.6 | FIXED | `3996fea` | `228c41a` | Compose has no pass-through for `INSIDEOUT_AI_TURN_DEADLINE`, so its reference row says compose does not interpolate it (R3-35) |
| rv4-USER-08 | docs-truth | 3.6 | FIXED | `80dde55` | `201a8ed` | The client banner was not touched by any cluster; `client/lib/l10n/en_prd.dart` already says replies are echoes, not AI answers (R3-50) |
| rv4-FEAT-08 | docs-truth | 3.6 | FIXED | `dba0e13` | `d147465` | none |
| rv4-ARCH-06 | docs-truth | 3.6 | FIXED | `ce648d1` | `56eb57e` | New `docs/architecture/cli-mcp-and-client-config.md`; `backend.md` corrected. ARCH-05's point that a fork's binaries aim at the upstream hosted API stays an owner follow-on (R3-38) |
| rv4-PM-03 | docs-truth | 3.6 | FIXED | `72d696f` | `1ca978b` | none; HANDOFF and the board state nine open product-promise slices |
| rv4-PM-09 | docs-truth | 3.6 | PARTIAL | `3356c20` | `4318627` | The stale Flutter 3.47.0 warning is gone and guarded. Archiving finished rows of the plans board is not done, because the guards read specific board rows and need an edit first (R3-36) |
| rv4-CLOSE-04 | docs-truth | 3.6 | FIXED | `b027c5a` | `0970300`; `745f60a` | Ten docs-only ledger rows relabelled Recorded and the counts stated. The ledger guard knows root-level documents by extension only, and the promise register still has DOCS-ONLY rows without a dated issue (R3-37) |
| rv4-CLOSE-04-BOARD | docs-truth | 3.6 | FIXED | `80a69cc` | `4d7ea78` | Repair row from round 1: the plans board and the review-3 README restate the recomputed counts. The guard's restated arm is a ceiling (at most 69), not equality with 63 |
| rv4-PM-06 | docs-truth | 3.6 | FIXED | `b6bc872` | `156f32d`; pre-gate `4060d64` | The worker reported `PARTIAL`: 42 broken links sat in dated records it did not own. The integrator's pre-gate patch repaired them and set the dated ceilings to 0. Over-budget docs stay on their modularity issue (R3-38) |
| rv4-CLOSE-10 | docs-truth | 3.6 | FIXED | `-` | pre-gate `4060d64` | No red commit: the wrong sentence ("existing rows default to en") sat in an unowned file and the migration says `DEFAULT 'zh'`. The patch also recomputed the rv3 index header and the rv3 track outcome |
| rv4-USER-09 (R1-06) | docs-truth | 3.6 | FIXED | `583839d` | `9299dc6` | README half: `setup --local` needs a checkout and `promote-admin` creates the account. The `--hosted` blurb in `cli_setup.go` and the "invite code" word in `cli.go` stay R1-07 (R3-46) |
| rv4-PM-08 | docs-truth, first-run-copy | 3.6, 3.8 | FIXED | `f662ac5`, `7449a58` | `04877ae`, `544c5c8` | One combined disposition: the README half (docs-truth) and the installers and bootstrap messages (first-run-copy) both state macOS and Linux and lead with the prebuilt one-liner. The `--hosted` blurb is R1-07 (R3-46); bootstrap https and attestation are PUB-03 (R3-14) |
| rv4-TEST-02 | test-hardening | 3.7 | FIXED | `7855dfc` | `12bbefd` | A test binary that is killed leaves `signup_mode` changed until the value is put back (R3-45) |
| rv4-TEST-03 | test-hardening | 3.7 | FIXED | `-` | `9afc391` | No red commit: a coverage task. API coverage 63.0% to 72.3%; 15 of 17 mutations caught and 2 equivalent. Packages still under 50%: GitHub sync and webhook, project guide, conversation messages, roadmap-draft loader (R3-42). Pinned, not fixed: an owner who is not a workspace admin cannot give a project away (API 500, `projects_update` has USING and no WITH CHECK) and nothing guards the last admin (R3-43) |
| rv4-TEST-04 | test-hardening | 3.7 | PARTIAL | `-` | `96362a2` | The CLI and the MCP tools are driven through the real API, but 17 of 52 tools are not called and the route-literal check was not written (R3-40) |
| rv4-TEST-06 | test-hardening | 3.7 | PARTIAL | `c82a3b9` | `855d621` | `INSIDEOUT_TEST_STRICT` refuses a database that is not on this machine, but nothing sets it yet (`scripts/dev.sh`, CI); 31 test files keep their own `DATABASE_URL` preamble; the check is a host check, not a `*_test` name marker (R3-39) |
| rv4-DATA-07 | test-hardening | 3.7 | FIXED | `2b90c56` | `5f2dda2` | none. 18 of 41 foreign keys had no supporting index and now every one has one (migration `20261018210000_index_foreign_keys.sql`) |
| rv4-CLOSE-08 | test-hardening | 3.7 | PARTIAL | `677af98` | `c7c792d` | The breaker-row tests are serialized, not cured: `CheckCircuit` and `RecordCircuitResult` are hardwired to the default breaker row, and three agent test files read the row without the lock (R3-41) |
| rv4-TEST-07 | test-hardening | 3.7 | DEFERRED_OWNER | `-` | `5e281d1` | A comment only. The replay cutoff stays at `20261006120000` until the owner reads `SELECT max(filename) FROM insideout.schema_migrations` on the hosted database (R3-44) |
| rv4-CLOSE-09 | test-hardening | 3.7 | DEFERRED_OWNER | `-` | `5e281d1` | The same owner question. `check_upgrade_cutoff.py` was not written, and moving the cutoff alone turns the upgrade lane red (R3-44) |
| rv4-CLOSE-02 | first-run-copy | 3.8 | FIXED | `16ec858` | `4088b72`, `c40199a` | `idea convert --locale` takes `en` or `zh` in the CLI and the MCP tool (one variadic `IdeaConvert`); the default comes from `LC_ALL`, `LC_MESSAGES`, `LANG`, else `en`, and is a tunable (R3-49) |
| rv4-FEAT-05 | first-run-copy | 3.8 | PARTIAL | `16ec858` | `4088b72`, `c40199a` | Shares its commits with CLOSE-02. The export headings in `server/internal/export/render.go` are still English (R3-47) |
| rv4-USER-10 | first-run-copy | 3.8 | FIXED | `59d6f42` | `46a1a92` | The workspace code and the account invitation have different names. Three client tests outside the cluster changed the word they look for (R3-50). An invitation that can carry a workspace is an API and schema slice (R3-48) |
| rv4-USER-12 | first-run-copy | 3.8 | PARTIAL | `173c429` | `386fe18` | A closed server's register 403 says registration is closed. The contact route is `DEFERRED_OWNER`; the invitation field is still shown when the mode is closed (R3-48) |
| rv4-LEAD-SETUP-BLURB (R2-06) | first-run-copy | 3.8 | FIXED | `e6ab582` | `7ba574e` | New guard `scripts/tests/check_first_run_copy.py`. The `--hosted` blurb in `cli_setup.go` and "invite code" in `cli.go` are R1-07 (R3-46) |
| rv4-QUAL-04 | context-deadlines | 3.9 | PARTIAL | `5bf3bd9` | `4ead866`; test `98f6b3a` | The caller's context now reaches `apiclient`, the MCP handlers and the CLI (exit 130 on Ctrl-C after a 400 ms grace); the CLI half was unproven until the round-1 repair `98f6b3a`, a test only. `runHealthCheck` in `cli_setup_check.go` still builds its own 60 s context and its sign-in check has none (R3-24) |
| rv4-QUAL-06 | context-deadlines | 3.9 | FIXED | `ac02f14` | `76e4556` | none |
| rv4-REL-06 | context-deadlines | 3.9 | FIXED | `cdb5fa4` | `707a01b` | Quick calls get 20 s and AI calls 5 minutes. A cut call reads the raw "context deadline exceeded" (R3-26); the AI handlers never read the request body, so a client that leaves is not seen (R3-25) |
| rv4-FULL-07 | context-deadlines | 3.9 | FIXED | `ae5030d` | `587c0e3` | A node expand now gets the model's deadline too. Same body-read gap (R3-25) |
| rv4-FEAT-04 | github-evidence | 3.10 | FIXED | `d87a489` | `8f83c0f` | Commit sync reads with the repo's App installation token and the server no longer reads `GITHUB_TOKEN` for it. `setup --check`, two scripts and a skip message still ask for that key (R3-27); the web client has no text for the new error codes (R3-28) |
| rv4-FEAT-07 | github-evidence | 3.10 | FIXED | `f70410a` | `e50d882` | none. Only a successful `deployment_status` is release evidence, so the GitHub App must deliver that event (operator notes) |
| rv4-COMP-05 | github-evidence | 3.10 | PARTIAL | `994a71d`, `4eb6ea8` | `27fede5`; pins `ffd55db`, `649844b` | A repo binds to one project, and only once GitHub shows it to the server (migration `20261018180000_github_repo_binding.sql`). `DEFERRED_OWNER`: what proves the linker controls the repo (owner decisions). Renamed or transferred repos, and duplicate bindings that already exist (R3-29) |
| rv4-SEC-06 | github-evidence | 3.10 | FIXED | `1aebc35` | `e7356b8` | `FetchGuideFile` refuses dot-only names and a stored refused repo link answers 400. The web client shows generic text for the new codes and for a 403 read as a rate limit (R3-28) |
| R1-12 (firstLine half) | github-evidence | 3.10 | FIXED | `1bcbc99` | `e9ead0e` | A commit subject or PR title is cut between characters, at 120. The message-cap half (`conversations.go`, 10,000 bytes) stays open as R1-12 |
| rv4-COMP-03 | client-assets | 3.11 | FIXED | `2e4ac83`, `1859c1a`, `c83ab86`, `525d089` | `b11e200`, `0e8092c`, `b0b260c`; chore `666c3f8` | CanvasKit and a font mirror (872 files, 23 MB) are served from this origin and the CSP names no host. Pending first CI run after push: the web build flag and the nginx checker. `loadWebFonts()` waits up to 10 s for CSS faces CanvasKit ignores (R3-30); nothing compares the mirror to the SDK's font table and nginx sets no long cache for `/fonts/` (R3-31); no hosted build was loaded in a browser (R3-32) |
| rv4-COMP-06 | client-assets | 3.11 | FIXED | `b99500a` | `d7f78f3` | Manifest level only: no Android SDK here, so no APK was built, and a `dart:io` socket may not consult the cleartext config (not proven); a Dart https guard in `session.dart` is not written (R3-32) |
| FEAT-02-CLIENT-APPLY (R2-01) | client-apply | 2.3, 2.4 | FIXED | `6096862` | `0d33077` | The feed's own apply is gone; the server applies once. Leftovers: an unused `projectId`, the unread `ProposalItem` and `ProjectUpdate.items`, the "N nodes added" key, and a failed apply is a 500 (R3-33) |
| rv4-DATA-03 read half (R2-02) | users-read-grant | 2.6 | FIXED | `8f57d77` | `c57073b` | Migration `20261018200000_users_read_grants.sql` takes `users.password_hash` out of the app role's reads; the rollback `GRANT` is in the fragment. The hosted role was not inspected, and the migration's `DO` assertion fails if `PUBLIC` or an inherited role can still read the column. Ceilings (R3-34) |

Not counted, and not fixed: `rv4-PM-04` (red `0de150c`, fix `5320251`) and
`rv4-PM-05` (red `7ac78ce`, fix `3b54d25`). The release-supply worker stamped the
version in `install.sh` and the Dockerfile and rewrote `docs/usage/releasing.md`,
none of them files it owned; the verifier returned the cluster, and both were
reverted (`c201f92`, `12bc405`). They move to the proposed wave-4
`w4-release-version`, with R1-04, R1-05 and R1-16 (R3-18).

Judgement calls a reader should know about:

- `rv4-SEC-03` is `FIXED` with the register 409 kept on purpose. A taken address
  still answers 409 at sign-up, so an enumeration of that one endpoint remains;
  what changed is that the mail round trip no longer tells a known address from
  an unknown one. An owner who reads that as partial can change the row.
- `rv4-AIDEV-02` is `FIXED` with the gate kept on purpose, for the same reason:
  the register's own recommendation was to announce a skipped critic, and that is
  what shipped. The stricter reading is one condition and one pinned test away.
- `rv4-PM-06` and `rv4-CLOSE-10` are closed by the integrator's pre-gate patch,
  not by a worker commit, because the files were unowned. `rv4-PM-08` is one
  disposition for two halves written by two clusters.
- Six of the thirteen clusters needed a repair round before their verifier
  accepted them (see the Verifier column of [wave 3](wave-3.md#clusters)).
