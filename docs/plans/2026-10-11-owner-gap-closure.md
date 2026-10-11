# 2026-10-11 — Close the owner-blocked gaps

The owner directed full closure of the six owner-blocked review-3 rows
(SEC-01, USER-02, FEAT-03, PM-01, PM-02, PM-04, USER-05, REL-06, SEC-05 —
the register's owner halves). Everything below is delegated operator work;
each step lands with observable evidence.
*Corrected 2026-10-11: the parenthetical calls all nine register
owner-halves "owner-blocked rows"; at the remediation close the
disposition ledger's Owner dispositions were exactly six (SEC-01, USER-02,
FEAT-03, PM-01, REL-06, SEC-05) plus the owner halves of three code-closed
rows (PM-02, PM-04, USER-05).*

## Checklist

- [x] Deploy: `railway up --service server` (applies the seven pending
      migrations on start — confirm applied list in deploy logs), then
      `railway up --service app`; BUG-027 owner checks (domain `/api/v1/me`
      → 401 without an app restart).
- [x] Hosted probes: `SMOKE_HOSTED=1` read-only suite + `/auth/config`
      reports `mailEnabled` (PM-04 first_check; PM-02 skew gone).
- [x] Admin bootstrap on hosted (sanctioned owner-pool path): mint one
      invitation (code returned inline), register a dated probe user,
      `promote-admin` via the owner pool; record every credential-free step.
- [x] Mail funnel live pass (found the revoked key; re-key is the one owner credential left) (FEAT-03/REL-06/USER-02): admin mail probe
      `status=sent`; invitation emailed to the Resend-verified address and
      delivered (provider-verified); register → magic-link send → token
      redemption; forgot/reset send → reset completion. Provider-side
      delivery evidence via Resend's email API; no secrets printed.
- [x] LLM provider exercise (SEC-05 companion): one real coach turn on the
      hosted box (SSE message_start/delta/message_end).
- [x] PM-01 reconciliation: with Railway read (variable names present: mail
      key, MAIL_FROM, PUBLIC_URL) + live probe result, correct the three
      disagreeing records to one truth.
- [x] Docs truth after deploy (PM-02): update the deployed-vs-tree claims
      (client/README, HANDOFF, docs-truth check expectations) and re-run
      `scripts/tests/check_docs_truth.py`.
- [x] USER-05: `v0.1.0` tagged/published/verified (billing blocked the workflow; assets built locally; anonymous curl needs the repo public — owner visibility call); release workflow builds; verify the
      HTTPS `bootstrap.sh` download and one tarball checksum.
- [x] Records: ledger owner rows closed with evidence, promise register
      updated, dated changelog, board row, HANDOFF resume path; push, CI
      green, MBP + MBP2 synced.

## Status

**Delivered 2026-10-11.** Preflight done: Railway CLI authenticated; mail key,
MAIL_FROM, PUBLIC_URL, LLM quartet all present on the server service
(names only — values never read); no local CLI session, so hosted admin
goes through the owner-pool bootstrap above.
*Superseded 2026-10-11 (same day): the checklist's "re-key is the one
owner credential left" no longer holds — the owner's send-scoped key was
synced to all four homes and every mailed flow ran live through the
provider ([changelog](../changelogs/2026-10-11-owner-deploy-and-mail-truth.md),
[HANDOFF](../HANDOFF.md) item 1).*
