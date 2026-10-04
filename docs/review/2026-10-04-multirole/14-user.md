# 14 · User

**Lens:** first-run happy path from docs alone, auth/session lifecycle, UI strings vs engine behavior, bilingual parity, invitation-only signup vs README
**Score:** **5.5 / 10**
**Bottom line:** Once you are inside, InsideOut mostly keeps its promises — the Coach UI, the review flow, and the bilingual strings all match what the Go engine actually enforces. But the first three minutes documented in the README do not work: step 1 ("Register an account") dead-ends because signup is invitation-only by default and no quick-start doc says so or says how to create the first admin; the advertised `docker compose up -d` fails to parse for the documented default remote-DSN setup; and after 15 minutes the client's expired access token turns every action into a raw English 401 error until a full page reload. The polish inside is real; the doorstep is broken.

Reviewed against main@877ad03; High/Critical findings re-checked by an independent skeptic — verdicts in 01-findings-register.md.

---
## Scope

`README.md` (quick start + core features), `docs/SETENV.md`, `docs/usage/local-development.md`, `docs/usage/environment.md`, `docs/usage/cli-mcp-and-guidance.md`, `client/lib/l10n/{en,zh}.dart`, the auth/landing/workspace/ideas/PRD/coach/dashboard UI under `client/lib/features/`, `client/lib/session/*`, `client/lib/api/client.dart`, registration/mail/session handlers under `server/internal/{api,mail,config,agent,store}`, `server/db/migrations/20260825120000_invitation_signup.sql`, `server/cmd/insideout/seed.go`, `docker-compose.yml`, and recent `docs/design-qa/` entries (2026-09-17/18).

## What is good

- **UI strings and engine behavior are consistent.** The four coach stage labels (`client/lib/l10n/en.dart:267-270`) match the enforced state machine (`server/internal/agent/prompts.go:20` `var stageOrder = []string{StageClarify, StageDraft, StageCritique, StageFinalize}`). The review flow is real: `UpdatePrdStatus` allows `reviewing`/`draft` for the author and `approved`/`rejected` only for workspace admins, so the README's promise holds.
- **Bilingual parity is exact.** A mechanical key diff shows 319 keys in each of `en.dart` and `zh.dart`, zero missing either direction, zero duplicates — including the newer Cobalt-era surfaces.
- **Fail-fast env handling is well documented.** `config.Load` errors map to fixes in `docs/SETENV.md`, `dev.sh` preflights before launch, and rerunning `env.sh init` is documented as safe.
- **Offline degradation is honest in the engine.** Missing `INSIDEOUT_LLM_API_KEY` selects a working template coach (`docs/SETENV.md` Step 3), and mail without a key logs instead of failing (`server/internal/mail/mail.go:48` `mail dev mode — not sent`).
- **Design QA is a living practice.** `docs/design-qa/README.md` records verbatim user feedback with resolutions; the 2026-09-18 backend walk exercised the real API end to end, including a live Coach SSE send.

## Findings

**USER-01 · High · README quick start step 1 dead-ends: invitation-only signup is nowhere on the documented path.**
`README.md:41` — `1. 注册账号（邮箱 + 密码）/ Register an account (email + password)`. A fresh database seeds `('signup_mode', '"invitation_only"')` (`server/db/migrations/20260825120000_invitation_signup.sql:32`), so registration without a code returns 403 `"an invitation is required to register"` (`server/internal/api/auth.go:84`), which the UI renders as the generic `register.error`: `'Could not create account. Please try again.'` (`client/lib/l10n/en.dart:109`). Worse, a fresh install has no admin: the first one is created out-of-band with `insideout promote-admin <email>` (owner DSN), documented only in `docs/usage/cli-mcp-and-guidance.md:67` — never in README/SETENV/local-development.
*Failure scenario:* new operator follows README on a fresh install → fills email+password → "Could not create account. Please try again." → no hint about invitation codes, no mention of `signup-mode open`, and no one who could mint an invite. The team's own walk hit this: "invitation-only hosted signup" blocked the hosted session (`docs/design-qa/2026-09-18-cobalt-flutter-walk.md`).
*Fix:* add a "First user & signup policy" step to README + SETENV + local-development: default is invitation-only; bootstrap with `promote-admin`, then mint invites or set `signup-mode open` for local dev; map the 403 to a distinct UI string. diff_size: ~30 lines across three docs + 2 UI strings.

**USER-02 · High · Expired access token (15-minute default) breaks the live app with no client recovery.**
`server/internal/config/config.go:82` — `parseDuration("INSIDEOUT_ACCESS_TTL", "15m")`. The Dio client has an onRequest interceptor that only attaches the stored bearer (`client/lib/api/client.dart:35-43`); there is no 401 → refresh → retry path anywhere. The only refresh is `Session.hydrate()` at boot (`client/lib/session/session.dart:52-57`). The server prefers the expired bearer over the cookie and answers `invalid or expired session` (`server/internal/api/middleware.go:145`), rendered raw by pages: `Text('${snap.error}')` (`client/lib/features/workspace/ideas_page.dart:100`), and the Coach panel sets `error = e.toString()` (`client/lib/features/prd/coach_panel.dart:92-93`).
*Failure scenario:* user stays on the page 15+ minutes (or suspends the laptop) → every click surfaces a raw English "invalid or expired session" even in zh mode → no redirect to login, no retry; only a full page reload (which re-hydrates and refreshes) recovers.
*Fix:* add a Dio onError interceptor: on 401, call `api.refresh()` once and retry the request; if refresh fails, clear tokens and route to `/login`. Stop rendering raw exception text; map codes to l10n strings. diff_size: moderate.

**USER-03 · High · `docker compose up -d` — the advertised one-command start — fails for the documented default setup, and "whole stack" omits Flutter.**
`README.md:72-77` — `或者直接用 docker-compose 一键启动完整技术栈：/ Or bring up the whole stack with docker-compose: docker compose up -d`. Compose hard-guards `INSIDEOUT_OWNER_PASSWORD: ${POSTGRES_OWNER_PASSWORD:?set POSTGRES_OWNER_PASSWORD in .env}` (`docker-compose.yml:17`) and the app password likewise (line 18). I verified `docker compose config` exits 15 with `required variable POSTGRES_OWNER_PASSWORD is missing a value` even when `DATABASE_URL` points at a remote instance — the documented dev default (`docs/SETENV.md` Step 1(a)). Compose also contains only postgres + server; Flutter is not a service.
*Failure scenario:* new user picks the remote-DSN default, runs the README's one-liner → compose refuses to parse and names a variable the docs told them they did not need; they conclude the project is broken.
*Fix:* relax `:?` to `:-` with a runtime guard inside the postgres init, or move the bundled stack behind a compose profile; fix the README wording to "postgres + server". diff_size: ~10 lines.

**USER-04 · Medium · The login UI promises emails the default local config never delivers.**
The form always renders "More ways to sign in" (`client/lib/features/auth/auth_form.dart:226-245`), and the success strings promise delivery: `login.magicSent`: `'If that email has an account, a sign-in link is on its way. It works once and expires in 15 minutes.'` (`client/lib/l10n/en.dart:53-54`). But with the default empty `INSIDEOUT_MAIL_API_KEY`, `Send` only logs and returns nil (`server/internal/mail/mail.go:48-49`). Mail dev mode is documented solely in `docs/usage/environment.md:100` — absent from SETENV and local-development.
*Failure scenario:* local dev clicks "Email link" → told the link is on its way → nothing arrives, no error anywhere in the UI; the link sits in a server log they were never told to read.
*Fix:* mention mail dev mode in SETENV/local-development next to the offline-coach note, and add `mailEnabled` to `GET /auth/config` so the UI can annotate or hide the email options when false. diff_size: ~20 lines.

**USER-05 · Medium · The "idea inbox" is workspace-wide, contradicting the canonical baseline's private-by-default capture.**
`PRODUCT.md:68` — `Capture remains one-step and private to its author by default.` The shipped list query has no author filter: `WHERE workspace_id = $1` (`server/internal/store/ideas.go` `ListIdeas`), and the README sells "a frictionless idea inbox" to every member (`README.md:24-25`). The UI offers no privacy cue (it even labels the wall `'No tags. No folders.'`, `en.dart:305`).
*Failure scenario:* a member jots a half-formed or sensitive idea expecting a personal inbox; the entire workspace sees it on the shared wall. The gap between the canonical doc and the shipped behavior is not flagged anywhere a user would see.
*Fix:* either filter `ListIdeas` to the author (with an explicit share action) or update PRODUCT.md and the README to state the inbox is shared. diff_size: small to moderate, product decision needed.

**USER-06 · Low · `seed` is not rerun-safe; the docs present it casually.**
`seed` handles a duplicate demo user (`server/cmd/insideout/seed.go` re-fetches on `ErrConflict`) but calls `CreateWorkspace` unconditionally, so a second run creates a second "InsideOut Demo Team" with a duplicate project/idea/PRD. `docs/usage/local-development.md:87` shows `seed      # optional: demo data` with no one-shot warning — while the same doc calls `smoke.sh` "rerun-safe" (line 141).
*Failure scenario:* developer runs `seed` twice during setup poking → dashboard shows duplicated demo workspaces; they cannot tell which is canonical.
*Fix:* skip seeding when the demo workspace already exists, or print a clear "already seeded" message. diff_size: ~5 lines.

**USER-07 · Low · Raw, unlocalized server error text is rendered as page content.**
Beyond USER-02's cases, `client/lib/features/workspace/workspace_page.dart:78` renders `Text('${snap.error}')` and the Coach panel falls back to `e.toString()`. A zh-CN user on any failure path sees English exception text (including `ApiException: ...` wrappers) where the rest of the UI is fully translated.
*Fix:* centralize an `ApiException.code` → l10n-key mapping; fall back to a generic localized message. diff_size: moderate.

**USER-08 · Low · PRODUCT.md's user-visible commitments no longer describe the shipped product.**
`PRODUCT.md:435-437` — `Visual identity: the existing Ink & Seal system—celadon ground, sumi-ink text, a single vermilion accent…` while the shipped theme is Cobalt (`client/lib/theme/cobalt_theme.dart`), a change the user explicitly demanded (`docs/design-qa/2026-09-17-live-language.md`: "I told you to ditch ink and seal theme"). The four-stage coach and admin approval are likewise listed as superseded (`PRODUCT.md:418-422`) yet remain the live experience.
*Failure scenario:* anyone reading the canonical product doc (it declares itself the winner of conflicts) forms expectations about look and workflow that the app contradicts on first screen.
*Fix:* add a "current state vs target" banner to PRODUCT.md pointing at the Cobalt design system and noting which superseded items are still live. diff_size: ~5 lines.

## Lead verification

| Lead | Verdict | Evidence |
| --- | --- | --- |
| L2 (compose fails; README overpromises "whole stack") | Confirmed (user lens) | `docker compose config` exits 15 on a remote DSN with only the two required vars set; `docker-compose.yml:17-18`; README.md:72-77 |
| L3 (README describes superseded coach + admin approval) | Confirmed with nuance | README matches the *live* engine (stageOrder + gated `advance_stage`; admin-only approve/reject in `UpdatePrdStatus`), so users are not misled today — the debt is directional vs PRODUCT.md (USER-08) |
| L4 (PRODUCT.md mandates Ink & Seal; shipped Cobalt) | Confirmed | PRODUCT.md:435-437 vs `client/lib/theme/cobalt_theme.dart`; design-qa/2026-09-17-live-language.md |
| L5 (HANDOFF/plans stale) | Out of lens; not re-verified | — |
| L6 (stale Nuxt references) | Confirmed, not user-facing | `client/lib/api/client.dart` Nuxt comments — code comments only |
| L13 (flutter analyze/test not run) | Confirmed not-run here | Bilingual parity checked mechanically instead (319=319 keys) |
| L15 (secret scan clean) | Out of lens | — |

## Recommendations

1. Rewrite the first three minutes (README + SETENV + local-development): state the invitation-only default, document `promote-admin` → invite (or `signup-mode open`) as the bootstrap step, and give the server's 403 a distinct, actionable UI string.
2. Add a client-side 401 → refresh → retry interceptor with sign-out fallback; eliminate raw `e.toString()` rendering in favor of localized error strings.
3. Make the compose path parse for remote-DSN users (relax `:?` guards or profile the bundled stack) and correct the "whole stack" claim.
4. Document mail dev mode on the quick-start path and expose `mailEnabled` via `/auth/config` so the UI can annotate the email sign-in options.
5. Resolve idea-inbox visibility one way or the other — author-private per PRODUCT.md, or explicitly shared — and make the UI string say which.
6. Make `seed` idempotent or loudly one-shot.
7. Refresh PRODUCT.md's brand and coaching sections to describe the shipped Cobalt, stage-based experience as current state.
8. Keep the mechanical bilingual key-parity check (a two-line script) in the client test suite so parity cannot regress.

## What this role would do first

Rewrite the first three minutes so a newcomer on a fresh clone never hits an undocumented wall. The rewritten quick start (full block below) inserts the missing step 3 — declare the signup policy — between `migrate` and `run`:

```bash
# 1. Configure (two required values: DATABASE_URL + INSIDEOUT_JWT_SECRET)
./scripts/env.sh init

# 2. Database
./scripts/dev.sh -C server go run ./cmd/insideout migrate
#    optional demo: ./scripts/dev.sh -C server go run ./cmd/insideout seed   # one-shot

# 3. Signup policy — registration is INVITATION-ONLY by default.
#    Local dev: open it.
./scripts/dev.sh -C server go run ./cmd/insideout signup-mode open
#    Real deployment: create the first admin (owner DSN), then invite by email.
./scripts/dev.sh -C server go run ./cmd/insideout promote-admin <email>
#    …then use the profile → Admin panel (or `admin invite <email>`) for each member.

# 4. Run
./scripts/dev.sh -C server go run ./cmd/insideout
cd client && flutter run -d chrome --dart-define=API_BASE=http://127.0.0.1:8080/api/v1
```
