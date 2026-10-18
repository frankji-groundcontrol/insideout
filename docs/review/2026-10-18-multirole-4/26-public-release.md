# 26 · Public release

**Lens:** secrets in tree and history, personal data and author identity, hosts/ manifests, license and font compliance, release and supply-chain integrity, CI permissions and pinning, SECURITY.md accuracy, privacy and public-service hygiene, install skill trust, redaction guard coverage  
**Score:** **6 / 10**  
**Bottom line:** No secret in tree or history (independently confirmed), but the public repo exposes real hosted-admin and owner-domain email addresses in docs, institutional and machine-name author identities in history, an ungated and unsigned release path with no license notices in the binaries, no security automation, and no privacy notice for the hosted service.

Tree: `main` @ `9f0b1c7`. Every High this lens filed went to an adversarial skeptic cluster; where the skeptic changed a severity or corrected a claim it is marked **[Correction]** below. The full register is [01-findings-register.md](01-findings-register.md).

## Skeptic verdicts on this lens's Highs

| ID | Filed | Verdict | Final | Reason |
|---|---|---|---|---|
| PUB-01 | High | DOWNGRADED | Medium | **[Correction]** The facts hold on this tree. A repo-wide grep for non-example addresses finds two full alumni-domain addresses (docs/changelogs/2026-08-25-invitation-only-signup.md:3 and docs/plans/2026-08-25-invitation-only-signup.md:13, the account promoted to sole admin), one full address on the owner's own domain … |

---

## Scope

Public-release and exposure audit of `/private/tmp/insideout-review4` (main @ 9f0b1c7), a repo that went public on 2026-10-11 with release v0.1.0 and a live hosted deployment. Covered: tracked files (`git ls-files`, 1,500), full history across all refs (secret-shape scan of every added line, deleted-file names, author/committer identity counts), `hosts/`, `.gitignore`, `scripts/`, the four workflows, SECURITY/CONTRIBUTING/LICENSE, bundled fonts, the web client's third-party loads, and the install skill. Read-only; no network; no Postgres started. Credential values, email addresses, and the deployed hostname are redacted below (`<email>`, `<prod-host>`, `<machine-host>`, `<project-ref>`).

## Score

| Aspect | Score | Why |
| --- | --- | --- |
| Secrets in tree and history | 9.5 | No key, token, PEM or DSN-with-password shape in any added line of any ref; `hosts/` manifests are names and kinds only; values live under ignored `.secrets/`. |
| Personal data and identity | 4.0 | Real e-mail addresses of the hosted admin and owner-domain account sit in dated docs; the redaction guard only knows one address; history carries institutional and machine-name author emails. |
| License and asset compliance | 6.5 | Fonts handled well (OFL file shipped, PuHuiTi purged from every ref). Release binaries ship with no license or notices. |
| Supply chain and release integrity | 5.0 | Unsigned, same-origin checksums, ungated tag trigger, `--clobber`, tag-pinned actions, no dependency or vuln automation. |
| Public-service hygiene (privacy, disclosure) | 4.5 | No privacy notice, no account deletion, third-party loads on every web visit, thin SECURITY.md. |
| **Overall** | **6.0** | Secret discipline is real; identity exposure and release trust are the gaps. |

## What is good

- Secret hygiene held through publication. I scanned every added line of `git log --all -p` for private-key, `ghp_`, `github_pat_`, `re_`, `sk-`, JWT, `sb_secret_`, `sbp_`, AWS shapes: 0 hits. The two PEM hits are prose (`server/internal/github/appauth.go` comment, a review doc). All 90 DSN-with-password hits are placeholders (`U:P@localhost`, `db.invalid`, `your-remote-host`).
- `hosts/*/keys/**/*.yaml` carry only `var`, `kind`, `note`, `optional`; no string of 32+ characters appears in `hosts/` (checked), and `.gitignore:175` ignores `.secrets/`. `scripts/keys.py:166-193` creates dirs 0700 and files 0600 and never prints values.
- PuHuiTi purge is real: no `.woff/.woff2/.ttf/.otf` other than the two Noto files appears in any ref's history, and `docs/changelogs/2026-07-27-puhuiti-reference-only.md` records why. Native fonts ship `client/assets/fonts/OFL.txt` with copyright lines; README's third-party section discloses the Google Fonts load.
- `scripts/check-redaction.sh` is a blocking CI step (`ci.yml:315`) that derives the host from `default_base.go` so it cannot match itself.
- `release.yml` and `smoke.yml` declare `permissions`; the one third-party action is `subosito/flutter-action` and it runs in a job with no secrets. `smoke.yml` secrets are only reachable from schedule/dispatch, never PRs.
- `bootstrap.sh` verifies the tarball hash before extracting and refuses on missing/mismatched lines.

## Findings

### PUB-01 (High) Real e-mail addresses in public docs; the guard is blind to them
`docs/changelogs/2026-08-25-invitation-only-signup.md:3`: "The user registered `<email>`, asked for it to be the only" (same address at `docs/plans/2026-08-25-invitation-only-signup.md:13`). It is the sole-admin account of the live hosted instance on an institutional alumni domain. `docs/changelogs/2026-10-11-owner-deploy-and-mail-truth.md:130`: "`<email>` (a member on the owner's domain; its password is" names a second live account on the owner's company domain. `docs/HANDOFF.md:36-38` spells the first name part of two more real addresses ("`frank.ji@…`", "`dingjue.ji@…`"). Confirmed by running grep; local parts are real-looking, not placeholders. `CLAUDE.md` forbids personal emails in docs, yet `scripts/check-redaction.sh:36-37` only blocks one dogfood address.
Failure scenario: an attacker now knows the admin login identifier of a public service with password reset, invitations and mail enabled: targeted phishing, credential stuffing, reset abuse. History keeps the old text.
Fix: replace the five occurrences with `<admin-account>` / `<owner-domain-account>`, add a non-example-address rule to `check-redaction.sh` (allow `example.*`, `*.test`, `*.invalid`, `resend.dev`, `insideout.local`, `test.local`); decide whether the admin address should be rotated. diff_size: ~6 doc lines plus ~8 lines in the guard.

### PUB-02 (Medium) No privacy notice, data-handling statement, or account/data deletion on a public hosted service
`server/internal/api/me.go:12`: `mux.HandleFunc("GET /api/v1/me", ...)` with only `PATCH` beside it; the only `DELETE` routes are idea, project, update, roadmap node, workspace and member. No `DELETE /me` or export route exists; grep for privacy/terms/隐私 in `client/lib` returns nothing; README has no privacy text. The coach sends idea and PRD text to a third-party LLM, mail goes through Resend, and `client/web/index.html:25` loads `https://fonts.googleapis.com/css2?...` with CanvasKit from gstatic (CSP at `client/nginx.conf:35` allows both), so every visitor's IP reaches Google.
Failure scenario: a visitor or EU user cannot learn what leaves the system or remove their account; a regulator or press reader finds a public multi-user service with no notice.
Direction: add a short `PRIVACY.md` linked from the sign-up screen (processors: LLM provider, Resend, Google Fonts; retention; contact), then a self-service account delete. First check: list every outbound call in `server/internal/agent` and `mail` to make the processor list complete.

### PUB-03 (Medium) Release trust is single-origin and ungated
`.github/workflows/release.yml:66`: `gh release upload "$GITHUB_REF_NAME" "$dist"/* --clobber`. The workflow triggers on any `v*` tag, builds without running tests or checking the tag is on main (`release.yml:3-5`), publishes tarballs and `checksums.txt` from the same job, and `scripts/bootstrap.sh` verifies against that same-origin file via a mutable `releases/latest/download/` URL. Tags are annotated but unsigned.
Failure scenario: anyone able to push a tag (or replay a run) replaces both tarball and checksum; every new `curl | bash` installs it. Checksums only catch corruption.
Direction: gate on `needs` of CI success for the tagged SHA and `git merge-base --is-ancestor`; drop `--clobber` for already-published tags; add `actions/attest-build-provenance` and `gh attestation verify` in bootstrap (optional if `gh` present). First check: `gh api repos/.../rulesets` to see whether tag protection exists (cannot verify offline).

### PUB-04 (Medium) Release binaries carry no license or third-party notices
`.github/workflows/release.yml:48`: `tar -czf "$dist/insideout-$os-$arch.tar.gz" \` packs only `insideout insideout-mcp`. The binaries embed MIT/BSD code (`pgx`, `golang-jwt`, `mcp-go`, `charmbracelet/*`, `x/crypto`) whose licenses require the notice in binary redistribution, and MPL-2.0 s3.2 requires telling recipients how to get Source for the Executable Form. No source file carries an MPL header either.
Fix: copy `LICENSE` plus a generated `THIRD_PARTY_NOTICES` (`go-licenses report`) into each archive and add the source URL to the notice. diff_size: ~10 workflow lines.

### PUB-05 (Medium) Supabase project ref and publishable key are in public history; a dead `.gitignore` exception re-allows the file
`.gitignore:72`: `!.env.supabase`. History commit `c64078a` added `app/.env.supabase` with a `https://<project-ref>.supabase.co` URL and a 46-character `sb_publishable_` key (shape-checked, value not read); `docs/NUXT_MIGRATION.md` carried the URL too. No code reads `.env.supabase` any more. The 2026-10-11 changelog (line 146) calls the key "public by design", but that was true for the old Nuxt app that spoke PostgREST directly to a schema built for it (`supabase/migrations/*_public_rpc_bridge_*` in deleted paths).
Failure scenario: if the Data API still exposes the legacy schema or any `public` RPC on that shared instance, the committed ref plus key reaches it from anywhere; a stray `.env.supabase` with real DSNs is also committable.
Direction: delete line 72; in the Supabase dashboard confirm exposed schemas exclude legacy ones, then rotate or disable the legacy publishable key. First check: dashboard, Settings, API, "Exposed schemas".

### PUB-06 (Medium) Git history identity exposure; the "privacy sweep" only looked for secrets
Counts only (`git log --all`, 1,155 commits): 1,005 commits carry an institutional `.edu` author address; 27 + 2 carry machine-local hostnames inside the address (`<machine-host>`, `<machine-host>`); 33 "Cursor Agent" commits on `@cursor.com`; one commit (`5cede51`, 28 files) is authored by a third party's real name on a corporate address; there is no `.mailmap`. `docs/changelogs/2026-10-11-owner-deploy-and-mail-truth.md:146`: "sweep first: no `.env` ever committed, no tokens or private keys in the" covers credentials, not identity.
Failure scenario: scrapers harvest the addresses from a public repo; the third party did not consent to publication.
Direction: decide whether to accept this (note it in SECURITY/README) or run a lock-step `git filter-repo --mailmap` rewrite plus force-push and re-clone notice; ask the third party first. First check: `git shortlog -sne --all` locally.

### PUB-07 (Medium) No security automation; secret scanning is advisory; PEM and key files are not ignored
`ls .github` shows only `workflows/`: no `dependabot.yml`, CodeQL, govulncheck, OSV, gitleaks or CODEOWNERS (grep for those in workflows: 0). `ci.yml` and `ubuntu-install.yml` have no `permissions:` block (L9), actions are tag-pinned. `.gitignore` has no `*.pem`/`*.key`/`*.p12`, though the GitHub App private key is a downloaded PEM (`hosts/*/keys/github/app.yaml` documents the lane). `config/git-hooks/pre-commit` runs the redaction check advisory-only.
Failure scenario: a `git add .` after downloading `<app>.private-key.pem` publishes the App key, and nothing blocks it.
Fix: add `.github/dependabot.yml` (gomod, pub, github-actions, 12 lines), top-level `permissions: contents: read` to both workflows, a `govulncheck` step, a gitleaks step, and `*.pem`, `*.key`, `*.p12` to `.gitignore`. diff_size: ~40 lines.

### PUB-08 (Medium) Five contract guards exist but no workflow runs them
`scripts/tests/` holds `check_go_floor.py`, `check_docs_truth.py`, `check_contribution_surface.py`, `check_smoke_wiring.py`, `check_product_promise_gaps.py`. I ran all five locally: each passes. `git grep` finds no invocation in `.github/`, `config/`, or Go tests. Sibling of L2/L1: the drift they were written to stop (Go floor, 74 broken links, SECURITY.md presence) can return unseen.
Fix: add one `contract-guards` job in `ci.yml` running the five with `python3 -I`. diff_size: ~10 lines.

### PUB-09 (Low) Install skill pulls root-installed toolchains with no checksum; CI check is looser than the floor
`skills/insideout-setup/references/modules.md:88`: `curl -L -o /tmp/go.tgz "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"` followed by `sudo rm -rf /usr/local/go && sudo tar ...` with no sha256 check (Flutter zip at line 171 is the same). `.github/workflows/ubuntu-install.yml:63`: `grep -E 'go1\.(2[5-9]|[3-9][0-9])'` accepts 1.25 while `server/go.mod` floors at 1.26.0.
Fix: pin the published sha256 and `sha256sum -c`; derive the grep from `go.mod`. diff_size: ~6 lines.

### PUB-10 (Low) Personal machine names and an OS username in tracked paths
`hosts/<machine-host>/host.yml:6`: `checkout: /Users/<os-user>/Projects/insideout` (same in the <alias> host.yml, plus `ssh_alias: <alias>`); both machine hostnames are directory names and appear in 9 docs. Manifests also publish which providers and which key lanes each machine uses and fill state. No value is exposed.
Direction: ship `hosts/` as `hosts/example/` plus a gitignored real tree, or keep and accept; either way scrub `checkout:` to a placeholder.

### PUB-11 (Low) SECURITY.md is a door without rules
`SECURITY.md:15`: the preferred channel is a GitHub private advisory; nothing in the tree can confirm that setting is on. No supported-versions line, no response expectation, no safe-harbor or testing limits for the live instance, no `.well-known/security.txt` in `client/web/`. The nightly `smoke.yml` already probes the host, so rules of engagement matter.
Fix: add 4-5 lines (supported version = latest tag, no DoS/automated scans beyond rate limits, test only your own accounts) and a `security.txt`.

### PUB-12 (Low) Host and provider identifiers appear in dated records despite the project rule
14 tracked files name the deployed host (changelogs, plans, review reports, workflows); `docs/changelogs/2026-08-18-flutter-web-host.md:30` records Railway deployment ids and `docs/changelogs/2026-08-21-dogfood-and-cleanup.md:11` a project UUID. The host itself is the shipped default (public by design), so impact is small; the ids add nothing for readers. CI also builds a debug-signed release APK (`client/android/app/build.gradle.kts:32`), harmless while unpublished.

## Lead verification

| Lead | Result | Evidence |
| --- | --- | --- |
| L1 siblings | Extended (none found) | Test dates with `time.Date(2026,...)` all pass a pinned `now`; the one handler-driven fixture (`api/roadmap_validation_test.go:101`, observedAt 2026-10-02) has no staleness window in `roadmap_validation.go:49-51`. |
| L6 | Confirmed, independently | 0 token-shape hits in all added lines of all refs; 2 PEM mentions are prose; 90 DSNs are placeholders. |
| L7 | Confirmed, extended | 31 manifests (plus 2 READMEs, 2 host.yml); no value-like strings; machine names plus `/Users/<name>` in `host.yml` (PUB-10). |
| L8 | Confirmed, extended | 1,005 `.edu`-address commits, 29 machine-hostname-address commits, a third-party identity (PUB-06). |
| L9 | Confirmed | Only `subosito/flutter-action@v2` is third-party; no dependabot (PUB-07). |
| L10 | Confirmed, extended | Ungated tag trigger, `--clobber` (PUB-03); no license in tarball (PUB-04). |
| L11 | Confirmed, extended | CI grep at `ubuntu-install.yml:63`; `check_go_floor.py` passes but is not wired to CI (PUB-08). |
| L13 | Extended | `check_docs_truth.py` exists but no workflow calls it. |

## Recommendations

1. Scrub the five real e-mail addresses from docs and make `check-redaction.sh` reject non-example addresses (PUB-01).
2. Confirm in the Supabase dashboard that the legacy project's Data API exposes nothing, rotate the legacy publishable key, delete `.gitignore:72` (PUB-05).
3. Gate `release.yml` on green CI for a main-reachable SHA, drop `--clobber`, add build-provenance attestation (PUB-03).
4. Put `LICENSE` and generated third-party notices in every tarball (PUB-04).
5. Add `dependabot.yml`, `permissions: contents: read` on all workflows, govulncheck, gitleaks, and `*.pem`/`*.key` ignores (PUB-07).
6. Wire the five `scripts/tests/check_*.py` guards into `ci.yml` (PUB-08).
7. Publish a short privacy notice and add account deletion (PUB-02).
8. Decide on history identity: accept in writing, or rewrite with a mailmap after the third party agrees (PUB-06).

## What this role would do first

Fix PUB-01 today: five doc edits plus a guard rule, because it names a live admin account in a public repo and is the only finding an attacker can use without any other weakness. Right after, spend ten minutes in the Supabase dashboard on PUB-05, since a committed project ref and key against a schema once built for direct client access is the one item whose real impact I could not verify offline. Then lock the release path (PUB-03/04) before the next tag, because `curl | bash` trust rests entirely on that workflow.
