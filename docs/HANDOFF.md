# HANDOFF

Resume here. The authoritative multi-task board is
[`docs/plans/README.md`](plans/README.md). The target product experience is
[`PRODUCT.md`](../PRODUCT.md), and the running system is described by
[`docs/architecture/`](architecture/index.md).

## What to do next

1. **Install/onboarding slice closed 2026-10-05** ([plan](plans/2026-10-05-installer-and-readme.md),
   [changelog](changelogs/2026-10-05-installer-and-readme.md)):
   `scripts/install.sh` puts both binaries in `~/.local/bin` (PATH
   appended idempotently), `scripts/mcp-config.sh` emits the
   cross-agent MCP registration, README rewritten for
   what/install/usage/further-reading. Scripts + docs only; nothing
   deployed. No follow-ups owed.
2. **Buddy host key store in flight** ([plan](plans/2026-10-05-buddy-host-keys.md),
   [changelog](changelogs/2026-10-05-buddy-host-keys.md)): this machine
   is now a second host in `hosts/`; 4 of 12 landed (jwt + both Postgres
   DSNs + LLM key; `supabase.yaml` renamed `postgres.yaml` on both
   hosts). Remaining keys land key-by-key with the user — mail + the
   GitHub keys copy next from MBP's agents_home store, then `.env`
   mirrors; audit/fill with
   `INSIDEOUT_KEYS_HOST=buddys-macbook-pro-2.local`.
3. **Both remediation rounds closed — operator actions remain.** The
   review-1 five-wave plan closed 2026-10-04
   ([changelog](changelogs/2026-10-04-remediation-wave-3-4.md)); review 2
   ([report](review/2026-10-04-multirole-2/README.md), 108 findings,
   7 confirmed High) had its waves 1–3 fixed and shipped the same day
   ([plan](plans/2026-10-04-remediation-review-2/2026-10-04-remediation-review-2.md),
   [changelog](changelogs/2026-10-04-remediation-review-2.md)): critic
   empty-messages fix, drop visibility, web SSE 401 refresh, l10n error
   mapper, proposal Accept/Reject loop, per-client XFF throttle keys
   (nginx passes the edge's XFF through), backup runbook, deployment
   evidence, band scaffold. Deploy record: both services shipped CI-green
   `073def0` on 2026-10-04 21:23 +08; domain verified (landing 200,
   API 401, bad-login probes 401). What remains, all human-only: rotate
   the Resend mail key if local session transcripts can leave this
   machine (one was printed into a local transcript 2026-10-04; nothing
   committed or sent); the signed-in hosted page load (PM-02 release
   gate — hosted signup is invitation-only, needs a member login on the
   deployed domain; do not seed the remote database with the local demo
   identity); confirm Supabase automated backups are on (the managed half
   of the backup story); the live critic exercise rides the next
   critique-stage conversation (local env has no LLM key). Wave 4
   feature slices (Decision Log UI, Branch slice 1, per-project
   authority) are the next product work, each behind its own plan. Walk
   record:
   [`design-qa/2026-09-18-cobalt-backend-walk.md`](design-qa/2026-09-18-cobalt-backend-walk.md).
4. The recorded frontier is closed 2026-08-22
   ([frontier](changelogs/2026-08-22-frontier.md), plus
   [follow-ons](changelogs/2026-08-22-product-follow-ons.md) and
   [depth](changelogs/2026-08-22-depth-thread.md)): audience-view web
   rendering, proposal→structure application, and canvas cursors are
   live on top of presence, the time-first roadmap, and the full
   parity loop (50 MCP tools). What remains of PRODUCT.md is genuinely
   new product judgment — the next interview's findings decide it.
5. CLI / MCP parity: Stage 3 — write verbs + agent vocabulary
   (`context`, `focus`, `checkpoint/report`, `propose`, `version`) as
   API routes first, then projected. `insideout-mcp` (50 tools) and the
   CLI are live.
   Plan: [`docs/plans/2026-08-21-cli-mcp-parity.md`](plans/2026-08-21-cli-mcp-parity.md).
6. Restore Ink & Seal on Flutter: slice closed 2026-08-23 — canvas v1
   (bands + minimap), presence hardening (reconnect + a web SSE
   transport that actually streams,
   [changelog](changelogs/2026-08-23-canvas-presence-hardening.md)), and
   the web first-frame font gate
   ([changelog](changelogs/2026-08-23-web-font-flash.md)). Native fonts
   bundled and signed off 2026-08-20.
   Plan: [`docs/plans/2026-08-19-restore-ink-seal.md`](plans/2026-08-19-restore-ink-seal.md).
7. Flutter Android release when an SDK exists
   ([`docs/plans/2026-08-17-flutter-client.md`](plans/2026-08-17-flutter-client.md)).

```bash
# Local Flutter against the hosted API (real data; CORS already allows localhost;
# use your own deployment's domain)
cd client && flutter run -d chrome --web-port=5173 --web-hostname=localhost \
  --dart-define=API_BASE=https://<your-domain>/api/v1
```

Local `.env` `DATABASE_URL` works again (repaired 2026-08-20 with the
known-good app DSN); the Go server can boot from this machine.

The two-role DB model is live end-to-end: shared instance, Railway
(`DATABASE_URL` app + `DATABASE_OWNER_URL` owner), and both `.env`
files ([changelog](changelogs/2026-08-20-owner-app-roles-shared-instance.md)).
Railway autodeploy is off for both services, and `railway redeploy`
re-runs the last image instead of building — after changes land on
`main`, ship them explicitly: any `server/` change with
`railway up --service server` (Go API), any `client/` change with
`railway up --service app` (Flutter web host). **After a `server`
deploy, restart `app`** (`railway redeploy --service app`): its nginx
resolves the server's internal address only at startup, so `/api`
via the domain 504s until then (verify with `/api/v1/me` → 401, not
`/healthz`, which the SPA answers).

The PRODUCT.md version-first slice stays on the board as P2.

## Worktree to preserve

Both machines and origin were synced at the last `[checkpoint]` commit; the
worktree is clean. Do not revert unrelated edits.

Exception (2026-08-24): closed — the email-auth slice (magic link +
password reset + code login) is committed and deployed to Railway
(server + app; new auth routes verified via the domain). Mail delivery
went live 2026-08-25 (Resend key + custom MAIL_FROM in Railway;
[changelog](changelogs/2026-08-25-mail-delivery-live.md)) — the
remaining check is a real-inbox confirmation, and other recipients only
receive once the sender domain is verified in Resend. Record:
[changelog](changelogs/2026-08-24-email-auth.md).

Their exact status, next action, blocker, and plan or record link are on the
[task board](plans/README.md). Review and checkpoint them one task at a time;
do not use `git add .`.

## Non-negotiables

- Never read or expose `.env` or `.env.local`.
- DB-dependent behavior requires a real PostgreSQL check and must stay inside
  the `insideout` schema.
- Commands and environment-safe launch instructions live in
  [`docs/usage/local-development.md`](usage/local-development.md).
