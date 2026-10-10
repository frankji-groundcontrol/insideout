# 13 · Project Manager

**Score:** **7.5 / 10** · 5 findings (1 High+)

**Bottom line:** An unusually well-governed single-owner repo — CI, records, and honesty culture are top-decile — but as a product it is momentarily a museum: zero releases, a hosted deployment 90 commits stale (verified live: the deployed /api/v1/auth/config lacks the sweep's mailEnabled field), a front-door email funnel whose real state is unresolvable because three authoritative records contradict each other, and one README claim ('Railway production serves this tree') that is false today. Everything blocking is a recorded owner action, nothing is hidden; the gap is execution, not truthfulness — except in the three places I file.

---

## Review 3 — project-manager report (main @ 754c502)

**Verdict.** This repository is governed better than most funded teams' repos: four CI workflows with a race lane and a skip-gate, a dated changelog for every wave, a disposition ledger that closed 187 findings to zero, and a bilingual README that conditions every not-yet-true claim. As a *product*, though, it is currently a museum piece: zero `v*` tags exist (`git tag` — empty, run this session), the hosted deployment is 90 commits behind `main`, and the front-door email funnel's real state cannot be determined from the records because three of them disagree. Every blocker is a recorded owner action — the honesty culture is real — but two documented claims are false today, and I file those.

### What I read and ran

Read: `README.md`, `client/README.md`, `PRODUCT.md`, `docs/HANDOFF.md`, the `docs/plans/README.md` board, `docs/changelogs/README.md` plus the 2026-10-10-close-open-gaps folder, the disposition ledger (all three registers), `.github/workflows/ci.yml`, `docker-compose.yml`, `.env.example` (keys and comments only), `scripts/env.sh` / `env-write.sh` / `env_catalog.py`, and `git log --oneline -60`. Ran (read-only): `git log`/`git tag`/`git branch -a`/`git remote -v`; greps for PRODUCT.md target concepts across `server/` and `client/lib`; and three live probes of the hosted deployment — `curl https://insideout.yalotein.net/healthz` → **200**, `/api/v1/me` → **401**, `/api/v1/auth/config` → **200 `{"signupMode":"invitation_only"}`** (no `mailEnabled` field). The full gates were run by the coordinator (lead 2), not by me; I did not execute an end-to-end clone→run (it needs Docker/Flutter/PostgreSQL and would write files) — that path is assessed by reading, and every artifact it names was verified to exist.

### Lead 14 — which PRODUCT.md promises lack implementation end to end

PRODUCT.md is explicitly a target document ("describes what InsideOut should become, not what every current screen already does", PRODUCT.md:5-6), and the disposition ledger already parks the gaps as **Parked feature** (02-multirole-2.md:13,17,18,22,23,39 — FEAT-08/PM-10/PM-11/PM-16/PM-17/FEAT-11). I re-verified each park still holds at 754c502 — no regression:

- **Product Branch / Merge** — `grep -rln "product_branch\|ProductBranch\|product-branch" server/ client/lib` → **zero matches** (PRODUCT.md:206-210).
- **Decision Log** — only `server/db/migrations/20260822100000_proposal_decisions.sql`; no decision-log surface (PRODUCT.md:218-221).
- **Roadmap Lenses** — only the audience-view comment at `server/internal/audienceview/audienceview.go:16` ("Projection is an audience's lens over the PRD core.") — audience views exist; roadmap Lenses do not (PRODUCT.md:277-285).
- **Scoped feedback invitations / dependency rebaseline** — no surface beyond signup invites (`server/internal/api/admin.go:24-26`); `grep -rn rebaseline` → zero (PRODUCT.md:254-259, 301-303).

What *is* real end to end: idea inbox → coach (offline template fallback without a key) → PRD with audience projections → human Commit → roadmap with deadlines/pressure → GitHub webhook evidence loop → CLI/MCP parity — the README's "what is this" loop (README.md:15-21) describes only implemented behavior. **Answer: the unbuilt promises are exactly the parked set; nothing new is vapor.** One nuance: PRODUCT.md:426-429 singles out only the authority model ("In particular, the running v1 authority model is still workspace-scoped"), which under-represents how much of the version-control layer (Branch/Merge/Decision Log) is target-only — worth one clarifying sentence there, not a finding while the ledger parks it.

### Lead 15 — the real-user funnel

Signup is invitation-only (verified live: `signupMode: invitation_only`). The funnel's email legs — invite delivery, magic link, password reset — depend on hosted mail, and the records cannot agree on its state: board row 42 says "mail key set + sender live 2026-08-25 … all five flows exercised live" *and* in the same row "until then mail runs in log-only dev mode"; row 27 says the Resend key was "rotated 2026-10-05 and synced .env + Railway + store"; HANDOFF (2026-10-10, the resume path) lists setting `INSIDEOUT_MAIL_API_KEY` in Railway as a remaining secret step. The deployed server predates the only probe that could arbitrate (`mailEnabled`, server/internal/api/auth.go:46 — absent from my live response). This is PM-01 below, the worst thing I found: the product's only growth path is in a Schrödinger state, and the sweep's "sign-in form says when mail is off" fix is dark in production either way.

## Findings

| ID | Sev | Dimension | Where | Evidence (verbatim) | Failure → Fix |
| --- | --- | --- | --- | --- | --- |
| PM-01 | High | FIX | docs/HANDOFF.md:33 | "`INSIDEOUT_MAIL_API_KEY` and `INSIDEOUT_MAIL_FROM` in Railway plus a redeploy" vs plans/README.md:42 "mail key set + sender live 2026-08-25" and :27 "Resend key rotated 2026-10-05 and synced .env + Railway + store" | A real user's magic-link/reset email silently never arrives. Fix: reconcile the three records to one truth; deploy so `/auth/config` reports `mailEnabled`; close with one live magic-link send recorded in a changelog. |
| PM-02 | Medium | FIX | client/README.md:5 | "Railway production serves this tree (`client/` Dockerfile + nginx)." vs HANDOFF.md:23 "Nothing from 2026-10-08 on is deployed" | A stranger demos the hosted URL believing it is this tree. Fix: reword to "serves a release, currently behind `main`" or deploy. |
| PM-03 | Medium | FIX | docs/plans/README.md:51 | "**In flight** … Android release build — blocked: SDK verified absent 2026-08-22" vs ci.yml:205 "flutter build apk --release" (green) and plans/README.md:19 "on web, iOS, and Android on `main`" | A resuming agent believes Android is blocked. Fix: close the row; point at the CI android job and the store-signed owner step. |
| PM-04 | Medium | HARDEN | client/lib/api/client_auth.dart:104 | "return data['mailEnabled'] as bool? ?? true;" against a live response with no such field | Forgot-password on the hosted path asserts mail works when the server cannot say. Fix: deploy; treat a missing field as unknown rather than true, or surface backend age in `setup --check`. |
| PM-05 | Low | IMPROVE | README.md:188 | Learn-more table exists; root `ls` shows no CONTRIBUTING.md/SECURITY.md, `.github/` holds only workflows | Outside contributors miss the changelog/English-only conventions; no private security channel for a hosted multi-user instance. Fix: add both, thin, linked from the table. |

## Strengths

- **The clone→run path is real.** Every script README names exists (`scripts/`: env, dev, setup, install, default-api, mcp-config — verified by `ls`); `env.sh init` prompts both setups and warns on placeholders (env-write.sh:195); `env.sh check` makes a placeholder required var a hard failure with the reasoning in-line — scripts/env.sh:150 "A placeholder in a REQUIRED variable is a FAILURE, not a warning." — and CI reproduces the from-zero path on a fresh runner (ci.yml:72-96, migrate→build→vet→race with a skip-gate).
- **The record system is exemplary and load-bearing.** The disposition ledger defines statuses and re-verification (docs/review/2026-10-10-disposition-ledger/README.md:22-31); the wave has a 14-file dated changelog folder; the board defines its status vocabulary (plans/README.md:57-61).
- **The ledger's "fixed" claims are visible in the tree at 754c502** — spot-checked: `readGate` (client/lib/theme/block_state.dart:156), the nginx 8 MiB webhook cap (client/nginx.conf:73 `client_max_body_size 8m;`), the `mailEnabled` contract with tests (server/internal/api/auth.go:33-46, auth_config_test.go:47-61), and the four pending migrations on disk (server/db/migrations: 20261010100000–130000, matching HANDOFF:25-28). They are dark in production, which is PM-02/PM-04, not a false record.
- **CI quality**: race lane with an unexpected-skip tripwire (ci.yml:84-96), analyzer info ratchet that only goes down (ci.yml:154-165), real-nginx re-resolve/header/body checks (ci.yml:127-129), a browser test lane for web-only imports (ci.yml:173).
- **Licensing hygiene**: MPL-2.0 in place (LICENSE:1) with meticulous font provenance including an explicit non-redistribution analysis (README.md:228-229).
- **Honest conditioning**: the prebuilt-binary one-liner is stated as post-first-release (README.md:43-48) — and indeed `git tag` is empty; the invitation-only default documents its own bootstrap commands (README.md:179-184), which I verified live (`signupMode: invitation_only`).

## What an outside contributor hits first

1. The curl|bash install one-liner 404s (no release exists) — but it is explicitly conditioned "Once a `v*` release is tagged", and the ledger's PM-19 row already owns the no-tag/no-CODEOWNERS state; not re-filed. 2. Registering locally returns 403 without an invite code — clearly documented with the `promote-admin` bootstrap (README.md:179-184). 3. No CONTRIBUTING.md — the process rules live in AGENTS.md, an agent contract (PM-05). 4. If they build the client and aim it at the hosted default, they meet the version skew (PM-04).

## Recommendations

1. Reconcile the mail records (PM-01) before anything else — it is one owner decision plus two file edits.
2. Execute HANDOFF item 1 verbatim: `railway up --service server` (four migrations + invite-code rotation caveat), `--service app`, the two BUG-027 requests, a signed-in page load. This single action un-darks all 90 commits and makes PM-02/PM-04 moot.
3. Fix client/README.md:5 and close the stale board row (PM-03) in the same docs pass.
4. Push the first `v*` tag to run release.yml end-to-end and flip the README install section from conditional to real (PM-19 owner row; board row 16 tracks the check).
5. Add CONTRIBUTING.md + SECURITY.md (PM-05).
6. Once mail is verifiably live, run and record one full funnel exercise — invite → register → magic-link → reset — closing lead 15 permanently.

## The release gate

The release gate is: the owner pushes the first `v*` tag (exercising release.yml), runs `railway up --service server` then `--service app` to ship the four migrations with the BUG-027 checks and a signed-in page load, sets and live-verifies the mail key with one delivered magic link — until those owner actions run, the repo's last 90 commits and its entire distribution story exist only on `main`.

## What this role would do first

Reconcile the mail-state records (PM-01): only the owner can read the Railway variables, so decide which of HANDOFF.md:33, plans/README.md:42, and plans/README.md:27 is true, correct the other two, and fold the verification (curl /api/v1/auth/config → mailEnabled, one live magic-link send) into the already-pending deploy.
