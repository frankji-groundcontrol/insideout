# 40 · Remediation plan

Tree: `main` @ `9f0b1c7`. Ordered by impact over effort: start at row 1.1 and
work down. Each wave ends with a runnable check; do not start the next wave
until the check passes. IDs point at [01-findings-register.md](01-findings-register.md);
the evidence, quote and skeptic verdict for each live in the role files.

Rules for every row: red test first when the row names one, real PostgreSQL
for anything database-shaped (`./scripts/dev.sh -C server go test ...`), one
commit per row, and a changelog entry per wave (the commit guardrail warns
without one).

Diff sizes are the reviewers' estimates, checked by the skeptics where a High
was involved. `dir` means the reviewer gave a direction and a first check, not
a patch, so size the row with a short spike first.

The completeness critic ([30](30-completeness-critic.md)) added `COMP-01` to
`COMP-08`; they sit in rows 1.16 to 1.18, 3.2, 3.6, 3.10 and 3.11. It also
recommends restoring `DATA-01`, `USER-02` and `AIDEV-01` to High. The skeptic
verdicts stand in the register, but the order here already treats them as
priorities: `USER-02` is row 1.2, `DATA-01` is row 2.6, and `AIDEV-01` is
fenced by row 2.5 after `AIDEV-04`.

## Wave 0 · Today, no code risk (privacy and owner decisions)

The repo has been public since 2026-10-11; these do not get cheaper.

| Step | IDs | File | Change | Size |
|---|---|---|---|---|
| 0.1 | PUB-01, PM-11 | `docs/changelogs/2026-08-25-invitation-only-signup.md:3` and the other four occurrences, `docs/changelogs/2026-10-11-owner-deploy-and-mail-truth.md:129` | Replace the real hosted-admin and owner-domain addresses with `<admin-email>` and `<owner-email>`; add a rule to `scripts/check-redaction.sh` that rejects any address not under `example.*` or `noreply`; decide whether to rotate the admin address. | ~6 doc lines + ~8 guard lines |
| 0.2 | PUB-05 | `.gitignore:72` | Delete the dead exception; in the Supabase dashboard confirm the legacy project exposes no schema through the Data API; rotate or disable the legacy publishable key. | dir (owner, dashboard) |
| 0.3 | PUB-10, PUB-12 | `hosts/*/host.yml:6` and the nine docs that quote it, `docs/changelogs/2026-08-18-flutter-web-host.md:30` | Scrub checkout paths, the OS user and the ssh alias to placeholders; decide whether `hosts/` ships as an example tree with the real one ignored; reduce the provider ids and deployed host in dated records to placeholders. | dir |
| 0.4 | PUB-06 | git history | **Owner decision, no action by default.** History carries institutional and machine-name author identities. Record accept or rewrite in `docs/issues/`; a rewrite means a force-push of `main` and needs explicit owner approval. | dir |

**Check:** `sh scripts/check-redaction.sh` passes with the new rule, and
`git grep -nEi '[A-Za-z0-9._-]+@[A-Za-z0-9-]+\.(edu|com|org)' -- docs | grep -viE 'example\.|noreply|<'` prints nothing.

## Wave 1 · One-commit fixes (about one engineer-day in total)

Independent of each other; land them in this order. Row 1.1 is the front door.

| Step | IDs | File | Change | Size |
|---|---|---|---|---|
| 1.1 | USER-01 | `server/internal/store/invitations.go:193`, `server/cmd/insideout/main.go` | Make `promote-admin` upsert the user (prompting for a password) or promote the first registrant when `users` is empty; add the red test on a fresh invitation-only database first. | ~25 lines |
| 1.2 | USER-02, FULL-03, PM-04 | `server/cmd/insideout/main.go:42` | Before `config.Load()`, handle `help`, `-h`, `--help`, `version` and unknown verbs: print the verb list from `cli.go`, exit 0 (help, version) or 2 (unknown). Stamp a version with `-ldflags` in `release.yml`. | ~30 lines |
| 1.3 | L1 / ARCH-04, QUAL-05 | `server/internal/api/projects.go:80-84`, `roadmap.go:70`, `projects_test.go:51` | Inject a clock (`Server.now`) into the view builders so `TestProjectQuietFlagged` stops depending on the real date; it has been red on `main` since 2026-10-16. Add a guard test against raw `time.Now()` in `internal/api` view builders. | ~30 lines |
| 1.4 | PM-02, TEST-08 | `.github/workflows/ci.yml:4` | Add a weekly `schedule` trigger running `go test ./... -race -shuffle=on`; change `docs/HANDOFF.md` to cite a dated SHA of the last green run instead of "the head run is green". | 3 lines |
| 1.5 | TEST-01, PM-05 | `.github/workflows/release.yml:5` | Run `go test ./... -race` as the first step of the release job, or gate on a green `ci` run for the tag SHA. Nothing else in `release.yml` changes in this wave. | ~6 lines |
| 1.6 | PM-01, CLOSE-01, TEST-05, PUB-08 | `.github/workflows/ci.yml:312-316` | Add one step to the python job that runs the five `scripts/tests/check_*.py` guards (`check_go_floor`, `check_docs_truth`, `check_contribution_surface`, `check_product_promise_gaps`, `check_smoke_wiring`) with `python3 -I`. All five exit 0 on `main` today. | 6 lines |
| 1.7 | USER-06 | `README.md:170` | Put the id after the flags in the `prd update` example (Go's `flag` stops at the first positional). | 1 line |
| 1.8 | REL-05 | `docker-compose.yml:47` | `stop_grace_period: 35s` on the server service; note it in `docs/architecture/deployment.md`. | 1 line |
| 1.9 | SEC-01 | `server/internal/store/auth_tokens.go:95` | Fold hash compare and attempts increment into one atomic `UPDATE`; reserve the throttle slot before verification. Skeptic measured 11 to 350 effective guesses under a concurrent burst instead of 5. | ~20 lines |
| 1.10 | USER-07 | `server/internal/api/admin.go:133` | Return `mailSent` from create-invitation and show "not emailed, copy this link" when false. | ~8 lines |
| 1.11 | PM-12, PM-07 | `.github/workflows/ubuntu-install.yml:63`, `ci.yml:240` | Tighten the Go regex to 1.26+; set the analyzer ratchet to the real baseline. | 2 lines |
| 1.12 | USER-09, USER-11 | `scripts/bootstrap.sh:127`, `client/lib/features/project/project_dialogs.dart:38` | Stop advertising `setup --local` from the prebuilt installer; relabel the Edit-project description field. | 5 lines |
| 1.13 | QUAL-02, QUAL-03 | `server/internal/api/auth_cooldown.go:79`, `server/internal/apiclient/prd.go:83` | Replace `cooldownClientIP` with `clientIP` and extend the shared test with the `"9.9.9.9, "` case; escape every apiclient path segment and query value (36 sites) with a hostile-input test. | a few lines + a sweep |
| 1.14 | SEC-06 | `server/internal/github/github.go:50` | Reject `.` and `..` repo segments, store the canonical repo URL, and stop using one global token for every project member's repo. | ~15 lines |
| 1.15 | PM-10, AIDEV-10, QUAL-01, FULL-06, DATA-05 | `server/go.mod:22`, `server/internal/mcpserver/product_tools.go:196`, `server/internal/agentcontext/agentcontext.go:177`, `scripts/env.sh:220`, `server/internal/store/project_updates.go:85` | Mark the direct dependency direct and add a `go mod tidy` diff check to CI; declare `items` in the `propose` tool schema; sort `blankSections` so `openQuestions` order is stable; treat an empty `DATABASE_OWNER_URL` as an error for the bundled Postgres; make the keyset cursor survive a deleted cursor row. | ~35 lines in total |
| 1.16 | COMP-02 | `client/lib/chrome/time_ago.dart:3`, `client/lib/features/project/roadmap_node_menu.dart:203` | Send the picked deadline as a date (`yyyy-MM-dd`, no `toUtc()`) or render it with `.toLocal()`; one Dart test under `TZ=Asia/Shanghai` that picks 2026-10-18 and expects 2026-10-18. Reproduced: it renders 2026-10-17. | ~10 lines |
| 1.17 | COMP-01 | `server/internal/api/ideas.go:92` and 15 sibling sites (`projects.go`, `workspaces.go`, `prds.go`, `roadmap.go`, `roadmap_validation.go`, `agent_routes.go`, `prd_commits.go`) | Count characters with `utf8.RuneCountInString` (the pattern already in `project_updates.go:49`) through one shared `tooLong(s, n)` helper; test with 67 Chinese characters (201 bytes) on each field. | ~30 lines |
| 1.18 | COMP-07 | `scripts/backup-db.sh:29` | `umask 077` before the dump and a `chmod 600` check; say in the usage doc that the dump holds credential hashes. | 2 lines |

**Check:** on a clean checkout, `(cd server && go build ./... && go vet ./... && go test ./...)`
is green including `TestProjectQuietFlagged`; `TZ=Asia/Shanghai flutter test` for the
deadline test is green and a 67-character Chinese idea title is accepted; under `env -i` with no database,
`insideout --help` and `insideout version` exit 0 and print no config error;
`for f in scripts/tests/check_*.py; do python3 -I "$f" || echo FAIL "$f"; done`
prints nothing; `grep -c 'go test' .github/workflows/release.yml` is at least 1.

## Wave 2 · Structural fixes with a red test first (about one week)

These close the Highs that are not one-liners.

| Step | IDs | File | Change | Size |
|---|---|---|---|---|
| 2.1 | ARCH-01, ARCH-02, ARCH-05, SEC-05 | `server/internal/clientconfig`, `cmd/insideout/cli.go`, `cli_roadmap.go:206`, `cmd/insideout-mcp/main.go:22` | One `clientconfig.Resolve(flagAPI)` returning `(base, token)`, used by the CLI, every `--api` flag and the MCP binary; bind the stored token to its base; stop swallowing config-load errors; delete the `envOr` copies and the `scripts/mcp-config.sh` shadow. Red test: each verb honours `--api` over config. | ~40 lines |
| 2.2 | FULL-02, USER-04, USER-03, USER-05, FULL-05 | `server/internal/apiclient/apiclient.go:39`, `server/internal/setup/flow/flow.go:151`, `docs/usage/installation.md:56` | Persist the refresh token and refresh once on 401 in `apiclient`; have plain `login` write the 0600 config and print its path; make `setup --check` send the stored credential and fail on 401; clear the web session only on 401 or 403 during refresh. | ~60 lines + ~25 |
| 2.3 | FEAT-02, FEAT-09 | `server/internal/api/agent_routes.go:254` | Add `applied_at` to `proposal_items` (one migration), apply all items in one transaction `WHERE applied_at IS NULL`, return 500 on failure; fix the Decision Log comment that says decisions append. Red test: accept-apply twice yields one set of nodes. | ~40 lines + migration |
| 2.4 | FULL-01 | `client/lib/api/client_project.dart:231`, `roadmap_ai.go:233` | Parse `children`, show the draft, `POST` the confirm route and parse the 201 list; remove client-side apply so the web UI cannot double-apply. Add a real-shape contract test. | ~25 lines |
| 2.5 | FEAT-01, AIDEV-04, AIDEV-08, AIDEV-01 | `server/internal/mcpserver/product_tools.go:120,176`, `agentcontext.go`, `agent/tools.go:148` | Introduce an agent token scope (or MCP profile) that omits `proposal_decide`, `commit`, `signup_mode`, `invite_*`, deletes and password tools, **enforced server-side**; reject a decision by the proposal's own author actor; fence and label every tool result and MCP output that carries user-writable text, starting with `get_prd`. Spike a short design first. | ~120 + ~60 + ~15 lines |
| 2.6 | DATA-01, DATA-02, DATA-03, DATA-04 | `server/db/migrations/20260720150000_row_level_security.sql:47,109`, `rls_walk_test.go`, `cmd/insideout/main.go:51` | One migration: `workspace_memberships_insert` allows `role='member'` only with a valid join code, `admin` only for the workspace creator; column-level grants (or a pinning trigger) on `users`; refuse to serve when the pool role has `BYPASSRLS` or is superuser; a write-side RLS walk beside `TestRLSWalk`. Measured: a bare self-insert as admin succeeds today; no API path reaches it. | ~15 + ~10 + test |

**Check:** `./scripts/dev.sh -C server go test ./... -race -count=1` passes against real
PostgreSQL with the new tests included; with `config.json` pointing at server A,
`insideout roadmap list --api B` reaches B; a login older than 15 minutes still
runs `whoami`; accept-apply twice leaves one set of roadmap nodes; the web
Expand with AI button creates children; `psql` as `insideout_app` cannot insert an
admin membership into another user's workspace.

## Wave 3 · Trust, reliability and docs truth (about one to two weeks)

| Step | IDs | File | Change | Size |
|---|---|---|---|---|
| 3.1 | AIDEV-02, AIDEV-03, AIDEV-05, AIDEV-06, AIDEV-07, CLOSE-03 | `server/internal/agent/tools.go:289`, `conversations.go:29`, `anthropic_parse.go:140`, `responses.go:259`, `ledger.go:243` | Coach trust layer: say when the critic was skipped and do not let a skip pass the gate silently; show parked finding resolutions on the confirm card and persist stage decline server-side; honour truncation on the tool-use path; keep parallel tool calls separate; require a minimum quote length for attested facts. | ~30 + ~60 + ~40 + ~10 |
| 3.2 | REL-01, REL-02, REL-03, REL-04, SEC-08, FULL-04, COMP-04 | `server/internal/api/server.go:64`, `healthz.go:10`, `presence_routes.go:154`, `single_replica.go:69`, `prds.go:114` | Cap concurrent SSE streams per user and per project (429), bound the `?client=` key space and add a server-side limit on presence cursor posts (today only the client throttles it); 2-3 s deadline on the `/healthz` ping and a per-request deadline for non-stream routes; recheck membership and token expiry on heartbeats; retry the singleton lock; send SSE 401s through the refresh path; cap PRD section text (runes) at write and truncate it in the coach prompt, so one user cannot send about 1 MiB per section into every turn. | ~40 + ~15 + ~20 lines |
| 3.3 | SEC-02, SEC-03, SEC-04, REL-07 | `server/internal/api/auth_throttle.go:300`, `auth_email_flows.go:302`, `store/sessions.go:71` | Trusted-proxy hop count (default `RemoteAddr`); bind the compose server port to loopback; send auth mail asynchronously so known and unknown users take the same time; revoke the session family on refresh reuse. | dir |
| 3.4 | PUB-03, PUB-04, PUB-07, SEC-07, PUB-09 | `.github/workflows/release.yml:48,66`, `ci.yml:1` | Ship `LICENSE`, third-party notices and the MPL source offer in the tarballs; stop `--clobber` overwriting a published tag; add a `permissions:` block, dependency and vulnerability scanning, and secret scanning; pin Actions by SHA; sign artifacts. | ~20 + ~10 + ~40 lines |
| 3.5 | PUB-02, DATA-06 | `server/internal/api/me.go:13`, `server/db/migrations/20260720135754_agent.sql:4` | Add `PRIVACY.md` (processors, retention, contact) linked from sign-up; decide retention and erasure in a `docs/issues/` record, then add an owner-pool erasure routine and a conversation retention window; then self-service account delete. | dir |
| 3.6 | ARCH-06, PM-03, PM-06, PM-09, CLOSE-04, USER-08, FEAT-08, FULL-09, CLOSE-10, COMP-08 | `docs/architecture/backend.md:24`, `docs/HANDOFF.md:13`, `docs/plans/README.md:12`, `PRODUCT.md:440`, `README.md:147` | Also open a dated issue for each of the five PARTIAL promise rows (50, 100, 103, 116, 126), make `check_product_promise_gaps.py` fail on a PARTIAL row with no issue, and decide whether `in_progress` to `done` needs evidence or a reason; say "PostgreSQL 18" where `README.md:147` and `local-development.md:12` claim "14+" (CI tests 18.6 only) or add a 14 lane; list `INSIDEOUT_ENV_FILE`, `INSIDEOUT_ROLELESS_URL` and `INSIDEOUT_SETUP_REPO_ROOT` in `environment.md`. Make the record system true: add the CLI, MCP and client-config surface to the architecture docs, state the nine open product slices in HANDOFF, fix the broken relative links (the coordinator's scan finds 75: 4 point at `.md` files in older review records, 71 at source paths that moved) and scope the "zero broken doc links" claim to what the gate measures, relabel docs-only "Fixed" ledger rows as "Recorded" and recompute the header. | docs only |
| 3.7 | TEST-02, TEST-03, TEST-04, TEST-06, DATA-07, CLOSE-08, CLOSE-09, TEST-07 | `server/internal/store/invitations_test.go:39`, `internal/api` | Serialize the shared `signup_mode` row (advisory lock in a shared test helper); lift `internal/api` coverage from 46.5%; add one CLI/MCP-to-real-API contract test; index the unindexed FK columns. | dir |
| 3.8 | PM-08, USER-10, USER-12, CLOSE-02, FEAT-05 | `README.md:182`, `client/lib/l10n/en_workspace.dart:13`, `server/internal/apiclient/prd.go:23` | First-run copy: prebuilt one-liner first, an invite contact route, macOS/Linux stated; name the two "invite codes" differently; pass a locale from the CLI and MCP convert path. | ~25 lines |
| 3.9 | QUAL-04, QUAL-06, REL-06, FULL-07 | `server/internal/apiclient/apiclient.go:27,87`, `server/internal/agent/coach.go:194` | Thread `context.Context` through `apiclient`, the CLI root and the MCP handlers; replace the fixed 20 s total timeout with per-call deadlines so AI draft and build calls are not cut; stop discarding store errors in terminal-state bookkeeping (about ten sites). | dir |
| 3.10 | FEAT-04, FEAT-07, COMP-05 | `server/internal/api/github_webhook.go:168`, `github_webhook_evidence.go:91`, `server/internal/store/project_repo.go:46` | Use the GitHub App credential for commit-timeline sync so private repos work; record deployment evidence only after an outcome is known; verify the binder can see the repo (GitHub App installation) before `SetProjectRepo` stores it, and stop two projects sharing one repo URL from receiving each other's events. | dir |
| 3.11 | COMP-03, COMP-06 | `client/web/index.html:25`, `client/nginx.conf:35`, `client/android/app/src/main/AndroidManifest.xml:7` | Bundle the fonts and the CanvasKit runtime with the web build and tighten the CSP to self; check first load from a network that blocks the two Google CDNs; set `usesCleartextTraffic` false in release builds (debug and loopback only). COMP-03 is unverified from here (no network in the review). | dir |

**Check:** every Medium in the register either has a closed row in this plan or sits on the
"Not in scope" list below; `sh scripts/check-docs-recorded.sh` and the five `check_*.py`
guards pass in CI; a fresh multirole review of the same tree scores no lens below 7.5.

## Not in scope yet

Leave these alone until Waves 0-2 close; each is a feature, a rewrite or a policy
decision, and doing it first spends the week on the wrong thing.

- **New product slices** (FEAT-03 CLI and MCP observation and validation verbs,
  FEAT-06 a smarter offline coach, CLOSE-07): the nine open slices stay open by design.
- **Multi-replica server** (ARCH-03, CLOSE-05, REL-04 beyond the retry): the
  single-replica contract is documented; changing it is an architecture project.
- **History rewrite** (PUB-06): wait for the owner's recorded decision in 0.4.
- **Breaking the monolithic functions** (QUAL-07): the budget is met; split them
  only when a row in Wave 2 touches the same function.
- **Low items** (AIDEV-09, AIDEV-11, ARCH-07, ARCH-08, CLOSE-06, CLOSE-11,
  DATA-08, DATA-09, DATA-10, DATA-11, FULL-08, FULL-10, PUB-11, QUAL-08, QUAL-09, QUAL-10,
  QUAL-11, QUAL-12, REL-08, REL-09, REL-10, REL-11, SEC-09, TEST-09): batch into one hygiene
  commit after Wave 3; none is user-visible.
