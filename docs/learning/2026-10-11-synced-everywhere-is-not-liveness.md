# A key recorded as "synced everywhere" can still be dead everywhere

## What was learned

Three authoritative records said the Resend mail key was set, rotated, and
synced — `.env`, Railway, and two hosts' key stores. All four copies were
**the same dead key**: the provider had revoked it after the last sync, and
no record tracked that because no record *asked the provider*. Key-sync
records prove where a value was copied, never that the value still works.
The only truthful state check is a live exercise against the provider
through a surface that reports the provider's own verdict — for this repo
that is the admin mail probe (`POST /api/v1/admin/mail/probe`), which
answered `resend returned 401 Unauthorized` the moment it existed to ask.

Two corollaries from the same afternoon:

- **A uniform-200 endpoint is not an exercise.** The auth email flows
  answer 200 regardless of delivery (deliberately, so callers can't probe
  addresses), so the 2026-08-24 "all five flows exercised live" record was
  true of the *HTTP contract* and silent about delivery. An exercise proves
  only what its response can refute.
- **`railway run --service server` is the owner-pool channel.** Hosted
  admin operations (migrate confirmation, promote-admin) need
  `DATABASE_OWNER_URL`, and the service env carries it — a one-off command
  under `railway run` runs locally with those values injected (never
  printed) and reaches the hosted database. This is the same sanctioned
  path the smoke script uses with the local owner pool.

## Evidence

The 2026-10-11 owner-gap closure
([changelog](../changelogs/2026-10-11-owner-deploy-and-mail-truth.md)):
after the deploy put the probe on production, it reported the 401; each of
the four stored key copies was then tested directly against Resend's API
and rejected; the replacement key needs the owner's Resend login because no
browser session or credential on either machine could mint one. PM-01's
"three records disagree" dissolved into "all three were stale the same way."

## Scope

Apply whenever a record claims a credential is *live* or a flow was
*exercised*: ask what the response could have refuted, and prefer a probe
that reports the provider's verdict. Sync state and liveness state are
different facts and need different evidence.
