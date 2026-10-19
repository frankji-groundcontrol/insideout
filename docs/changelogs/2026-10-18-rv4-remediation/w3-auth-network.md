# Wave 3, auth and network fixes — review-4 remediation

## Design spike (row 3.3 is `dir`)

Three shapes were settled before any code, each against the obvious
alternative. **Client address: trust a hop count, not a header.**
`INSIDEOUT_TRUSTED_PROXY_HOPS` (default 0, the TCP peer) says how many reverse
proxies in front of the server append to `X-Forwarded-For`; the Nth entry from
the right is the address the outermost trusted proxy saw, and everything to its
left is client-written and ignored. A trusted-CIDR list lost: it would make the
operator track an edge's address ranges, while the number of hops in a topology
is a fact the operator already owns. The default fails closed against an
attacker (a forged header buys nothing) and loud for the operator (one WARN per
process when a header arrives that is being ignored). **Mail: nothing on the
request path may depend on whether the account exists.** The token write and
the provider call move together onto a small bounded queue inside
`internal/mail`, not a goroutine per request (unbounded) and not a `Server`
field (`server.go` is another cluster's file); the price is that a queued mail
dies with the process. **Sessions: a login names a lineage, a replay ends it,
after a grace.** `family_id` rides every rotation, and presenting a token that
was spent more than 10 s ago (database clock) ends every live session of its
family. The grace exists because the wave-2 clients legitimately offer one
token twice (two CLI or MCP processes on one credentials file, two tabs, a
retry). It was settled by running the built CLI rather than by argument: with
10 s, six rounds of twelve simultaneous `whoami` on one login lose nothing;
with 0, the first burst ends the session.

Branch `rv4/w3-auth-network`, base `8c74218`, seven commits `488add8..09f5c01`
(three red/fix pairs and a comment fix), then this fragment. Findings are cited
as `rv4-<ID>` because review IDs restart per review. Row 3.3 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md).
No address, host, secret or token is spelled here; every account in the live
runs below was a throwaway on a private database.

## Dispositions

| ID | Status | Red | Fix | Note |
| --- | --- | --- | --- | --- |
| rv4-SEC-02 | `FIXED` | `488add8` | `e310aaa` | hop count, loopback compose port |
| rv4-SEC-03 | `FIXED` | `bae14ff` | `801e346` | timing oracle closed; register's 409 kept on purpose (below) |
| rv4-SEC-04 | `PARTIAL` | `1cab10c` | `7567835` | family revocation done; `iss`/`aud` and a logout denylist need `auth/jwt.go`, outside this cluster |
| rv4-REL-07 | `FIXED` | `bae14ff` | `801e346` | same pair as SEC-03 |

## What changed

- **A forged `X-Forwarded-For` no longer chooses the throttle bucket**
  (rv4-SEC-02). The auth throttle and the resend cooldown keyed on the
  rightmost entry of a header the client writes itself, so every request with
  a fresh header got a fresh bucket: the 8-failure lockout, the 3-sends window
  and the per-address cooldown could all be walked around by changing one
  header. Both now ask `Server.clientIP` (new `api/client_ip.go`): the TCP peer
  when the hop count is 0, else the Nth entry from the right across every
  `X-Forwarded-For` line (a repeated header is one list), falling back to the
  peer when the header has fewer entries than hops or the entry is not an IP
  address. A shared bucket is the safe answer when the count and the chain
  disagree. `docker-compose.yml` publishes the server port on `127.0.0.1`
  only, so nothing but the host and the bundled nginx can reach it to write the
  header, and passes the variable through with default 0.
  `scripts/tests/check_ops_hardening.py` gains the compose guard.
- **A known address answers like an unknown one, and a slow provider holds no
  request** (rv4-SEC-03, rv4-REL-07). Magic link, forgot password and login
  code did the account-dependent work inline: an unknown address returned at
  once, a known one waited for the token write and the provider (up to the
  client's 10 s). The request path is now the same for both: decode, the
  cooldown gate (keyed on address and client whether or not the account
  exists) and one user lookup. The token, its row and the mail run as one job
  on `mail.Sender.Enqueue`. A failed send is still logged as `<name>: send`
  and still lands in `LastSendError`, which the admin mail probe reads; the
  response never changes shape. `Wait` exists for tests; the server never
  calls it. Invitation mail (`admin.go`) stays synchronous on purpose: it is an
  admin action whose response reports `mailSent`.
- **A replayed refresh token ends the whole session family** (rv4-SEC-04).
  Reuse used to refuse only the replayed request, so whoever had rotated first
  (the owner, or a thief with a copy) kept a live session. Migration
  `20261018170000_sessions_family.sql` adds a nullable `family_id` and a plain
  index, both guarded. `NULL` is a pre-upgrade row and reads as a family of its
  own id, so there is no backfill and no `NOT NULL`. A login makes its session
  the root of a family, `RotateSession` carries it, and `RevokeFamilyOnReuse`
  ends every live session of the family of a token that was revoked more than
  `refreshReuseGrace` ago, comparing on the database clock so every replica
  agrees. The replayer gets the same 401 and body an unknown token gets (a test
  pins it) and the operator gets one WARN with the user id and the session
  count. A family holds one live session at a time, so ending it ends the
  newest; another device is another family and is untouched.

## Tunables

| Name | Where | Starting value | Move it when |
| --- | --- | --- | --- |
| `INSIDEOUT_TRUSTED_PROXY_HOPS` | env, `config.go` | 0 | the server sits behind reverse proxies that append to `X-Forwarded-For`: count them (hosted: 1) |
| `refreshReuseGrace` | `api/auth.go` | 10 s | raise if legitimate users are signed out, lower to tighten; keep it above the client's 500 ms wait (`apiclient` `renewalGrace`) plus request latency |
| `queueWorkers`, `queueDepth`, `jobTimeout` | `mail/mail.go` | 4, 64, 30 s | the provider is slow or busy enough that mail waits: a provider that hangs costs the client's 10 s timeout per job, so 4 workers drain a full queue of 64 in about 160 s at worst; a full queue drops the newest job with a log line. `jobTimeout` covers a token write plus that 10 s |

None is measured. `refreshReuseGrace` and the queue bounds are judgement
calls, written as named constants so a different value is a one-line change.

## The register 409, kept on purpose

`POST /auth/register` still answers an existing address with a distinct 409.
Making it uniform means accepting a signup without proof that the caller owns
the address (an email-verified signup), which is a product change, not a fix
inside this row. Where it is reachable: in `invitation_only` (the default, also
when the setting is missing or unreadable) a caller must already hold an
invitation code bound to that address, or the answer is the same 403 for known
and unknown addresses; in
`closed` it never is; in `open` it is a real oracle, braked only by the
30-per-minute per-IP volume cap on the route. If `open` ever becomes the
default, that is an owner decision: verified signup, or accept the oracle.

## Measured

Mail timing, medians of three requests per endpoint with a mail provider that
takes 200 ms to accept (an in-process stand-in; nothing leaves the machine):

| Endpoint | Before: known / unknown | After: known / unknown, three runs |
| --- | --- | --- |
| magic link | 202.3 ms / 0.83 ms | 0.42, 0.38, 0.39 ms / 0.39, 0.38, 0.33 ms |
| forgot password | 204.8 ms / 1.06 ms | 0.52, 0.35, 0.52 ms / 0.47, 0.35, 0.47 ms |
| login code | 202.7 ms / 0.92 ms | 0.52, 0.45, 0.30 ms / 0.45, 0.39, 0.28 ms |

The largest median gap after the fix is about 0.06 ms (unrounded), the cost of
fetching one row and enqueueing; the test fails at half the provider delay
(100 ms).

Live runs with the built binary against a private PostgreSQL 18, servers on
loopback ports of their own, a base build of `8c74218` beside them:

| Run | Base `8c74218` | This branch |
| --- | --- | --- |
| 12 wrong passwords, a different forged `X-Forwarded-For` each | 401 x12, never locked | hops 0: 401 x8 then 429 x4, plus the one WARN naming the variable |
| 12 wrong passwords, a different forged prefix, one real tail | 401 x8 then 429 x4 | hops 1: 401 x8 then 429 x4 |
| replay of a spent refresh token 11 s after its rotation, then the newest token | replay 401, newest **200** | replay 401 (body equal to an unknown token's), newest **401**, one WARN `refresh token reuse: session family revoked` |
| the same replay at once (the loser of a race), then the winner's pair | loser 401, winner 200 | loser 401, winner 200 |
| forgot-password, a known and an unknown address (dev-mode sender) | not run | both 200 `{"ok":true}` in 1.26 ms and 0.87 ms; the dev-mail log line follows the request log line |

The base locks on a constant tail too (second row): it keyed correctly only
while a proxy wrote the tail, and had no way to ignore a tail the client
rotates.

**The wave-2 hazard, run rather than argued.** The built CLI was logged in once
on a server with a 3 s access TTL, then six rounds of twelve simultaneous
`whoami` found the access token expired, so every process renewed from the same
saved login. With `refreshReuseGrace` at 10 s: 0 failures in 72 runs, six
rotations (one winner per round), no reuse line. With a scratch build whose
grace was 0: round 1 passed but ended the family (one reuse line), and every
process of rounds 2 to 6 was refused. So the losers' replays do reach the
server, moments after the winner, and the grace is what keeps them harmless.
A first harness without a server restart between rounds failed from round 6
on; the cause was the refresh volume cap (30 per minute per address, shared by
every loopback process at hops 0), not revocation: the run added no reuse line
and left one live session in the family. The rounds restart the server now.

## Known ceilings, recorded on purpose

- **Access tokens outlive a family kill.** The thief's access JWT stays valid
  for up to `INSIDEOUT_ACCESS_TTL` (15 minutes by default) after the family
  dies. `iss`/`aud` and a short logout denylist need `auth/jwt.go`, not in this
  cluster: that half of rv4-SEC-04 is the `PARTIAL`.
- **The grace cannot tell a stalled owner from a thief.** A client that
  presents a token spent more than 10 s ago is signed out. The clients re-read
  the stored pair right before posting, so this needs a stall of that length
  between the read and the post. Owner and thief inside one grace window are
  ambiguous: the first rotation wins, the second is refused, nothing is revoked.
- **Pre-upgrade lineages.** Sessions that rotated before the migration cannot
  be linked; the first rotation after it names the family, so a pre-upgrade
  token replayed later still ends what descended from it. Spent rows stay
  detectable until retention purges them (30 days after expiry).
- **The mail queue is in memory.** Jobs not yet sent when the process exits are
  lost (a redeploy can eat a just-requested link; the user asks again, and the
  60 s resend cooldown, also in memory, starts empty after the restart), and a
  full queue drops the new job with a log line. `Wait` is not wired into
  shutdown (`runServe` is `cmd/insideout/main.go`, not owned).
- **A shared bucket without the variable.** At hops 0 behind any proxy, every
  client shares the proxy's address. The WARN says so once per process. Compose
  with the bundled nginx is in that state by design: nginx passes the header
  through untouched (REL-13), so no entry of it can be believed.
- **Browser tabs are unchanged.** `handleRefresh` clears the auth cookies on
  every refused refresh, so a tab that loses a race can clear the cookie the
  winner just set. The grace keeps the session alive on the server, not in that
  tab's jar; this predates the change.
- **The warn-once flag is a package-level map keyed by `*Server`**
  (`forwardedIgnoredWarned`), because `Server` cannot grow a field without
  touching `server.go`. One Server per process in production, so it holds one
  entry.
- **IPv6 clients are keyed by full address**; a /64 prefix would stop a host
  rotating inside its own block. Marked `ponytail:` in `client_ip.go`.
- **Throttle and cooldown stay per process**, as the single-replica contract
  says. Nothing here changes that.

## Operator action

Before this reaches the hosted server, set `INSIDEOUT_TRUSTED_PROXY_HOPS=1`
there (one trusted hop: the Railway edge appends the client address, nginx
passes the chain through). Without it every hosted client shares one throttle
bucket: 8 failed logins from anyone lock everyone, and refresh is capped at 30
a minute for the whole population. The WARN line is the symptom. The compose
stack needs nothing: its default 0 is right for the bundled nginx alone.

## Left for the integrator

Documentation, example and index edits, plus two optional code follow-ups in
files this cluster does not own. Nothing here blocks the code.

INTEGRATOR TODO: `.env.example`: add `#INSIDEOUT_TRUSTED_PROXY_HOPS=0` (with a
one-line comment). `scripts/test_env_catalog.py` requires a skeleton line for
every `${INSIDEOUT_*}` the compose file interpolates; on this branch alone it
fails with `missing: ['INSIDEOUT_TRUSTED_PROXY_HOPS']`, and with the line added
locally it passes (48 passed, 0 failed).

INTEGRATOR TODO: `docs/usage/environment.md`: `INSIDEOUT_TRUSTED_PROXY_HOPS | no
| 0 | how many reverse proxies in front of the server append to
X-Forwarded-For (0 = key on the TCP peer; hosted behind the Railway edge and
the pass-through nginx = 1)`.

INTEGRATOR TODO: `docs/usage/deployment.md` (lines 101 to 108): rewrite the
client-IP trust chain paragraph (REL-13). The server no longer reads "the
rightmost entry" or trusts a locally spoofed header in compose; it takes the
Nth entry from the right, N being the variable, and the hosted value is 1. A
deployment behind a proxy MUST set it, or every client shares one bucket (the
proxy's address). Also say the compose server port is now loopback-only, which
makes the sentence at lines 97 to 99 ("do not publish the server port") true by
default.

INTEGRATOR TODO: `client/nginx.conf` (not owned here; another wave-3 cluster
owns it). The hop defaults assume nginx passes `X-Forwarded-For` through
untouched, as it does on this base (`proxy_set_header X-Forwarded-For
$http_x_forwarded_for;`, lines 79 and 101). At integration, look at that line
again: if the file now appends `$remote_addr`, every count shifts by one (the
compose default becomes 1, the hosted value 2) and the compose comment above
the variable must change with it. The two files change together.

INTEGRATOR TODO: `server/internal/api/auth_email_funnel_test.go` is edited again
by a later cluster (test hardening). Keep the one line added here, `m.mail.Wait()`
in `lastMail`: the emailed requests answer first and send from the queue, so a
test that reads the dev-mail log or counts sends must wait first, or it reads
the log before the job has written to it.

INTEGRATOR TODO: `docs/architecture/backend.md` (the auth paragraph, lines 60
to 100): refresh tokens rotate inside a session family; a spent token replayed
after `refreshReuseGrace` (10 s) ends the family; auth mail goes through the
bounded queue in `internal/mail`, so the "uniform 200" is uniform in time too.

INTEGRATOR TODO: `server/internal/apiclient/refresh.go` (not owned): the
`ponytail:` comment on `renewalGrace` says to serialize renewals with a lock
file "if the server ever revokes a token family on reuse". The server does now,
behind a 10 s grace that covers the 500 ms wait with room to spare, and the
live run above shows no failures. Update the comment to name
`refreshReuseGrace`, and keep the lock file as the upgrade path if that grace is
ever shortened below the client's wait.

INTEGRATOR TODO: indexes: the `w3-auth-network` row in this folder's
`index.md`; `docs/changelogs/README.md` and `docs/changelogs/index/2026-10.md`;
the open-findings record and `docs/issues/README.md` with the dispositions
above (rv4-SEC-04 `PARTIAL`, residue: `iss`/`aud` and a logout denylist in
`auth/jwt.go`).

INTEGRATOR TODO (optional): drain the mail queue on SIGTERM: call
`mail.Sender.Wait()` in `runServe`'s shutdown path (`cmd/insideout/main.go`,
with an accessor on `Server` in `server.go`), so a deploy does not eat a mail
that was accepted a moment earlier.

## Verification

Run against a private PostgreSQL 18 cluster on a port of its own, migrated from
this worktree. A skipped database test counts as red; none skipped.

| Finding | Red, at the red commit | Green, at the fix commit |
| --- | --- | --- |
| rv4-SEC-02 | `TestThrottleIgnoresAForgedForwardedForByDefault` fails 4 of 4 subtests (for example "request 3 from one peer with a fresh X-Forwarded-For: status = 401, want 429 — the forged header minted a new bucket"); `TestResendCooldownIgnoresAForgedForwardedForByDefault` fails 3 of 3 ("second send from the same peer with a fresh X-Forwarded-For: status = 500, want 200"); the compose guard reports the server port published on every interface | both pass; the hop table, repeated-header, fallback and warn-once tests pass; the compose guard passes (5 checks) |
| rv4-SEC-03, rv4-REL-07 | `TestAuthMail_KnownAddressAnswersLikeUnknown` fails on all three endpoints: known 202.3, 204.8, 202.7 ms against unknown 0.83, 1.06, 0.92 ms | passes (table above); the six queue contract tests and `TestAuthMail_ProviderFailureIsLoggedNotSurfaced` pass |
| rv4-SEC-04 | `TestRefreshReuseAfterTheGraceEndsTheWholeFamily` fails ("the newest token still refreshes (status 200) after a spent one was replayed"); `TestReuseOfAPreUpgradeSessionEndsItsSuccessors` fails ("a successor of the pre-upgrade session still refreshes (status 200)"); the three guards (inside the grace, 16 concurrent refreshes, another device) pass at red | all five API tests and the five store tests pass |

Mutation checks a verifier can run; each turned the named tests red and was
undone with `git checkout` afterwards:

- rv4-SEC-02: use one hop whatever the configured count, and 7 tests go red
  (both red-commit tests among them); take the leftmost entry instead of the
  Nth from the right, and 6 go red.
- rv4-SEC-03, rv4-REL-07: run the job inside `Enqueue`, and the timing test
  goes red while the queue tests deadlock (the mail package times out).
- rv4-SEC-04: a grace of 0 turns red the replay-inside-the-grace and the
  16-concurrent-refreshes tests; dropping the call in `handleRefresh` turns red
  the whole-family and pre-upgrade tests; dropping the `COALESCE` in the spent
  token's lookup turns red the pre-upgrade tests at the store and the API;
  starting a new family on every rotation turns red 6 tests across both.

Gates, all green on this branch: `go build ./...`; `go vet` on the api, store,
mail and config packages; `gofmt -l internal cmd` prints nothing; `go test` on
api, store, mail, config, apiclient and clientconfig with `-count=1` (797
passed, 0 failed, 0 skipped, the wave-2 refresh tests among them); the api and
store packages again with `-race`; `insideout migrate` twice (the second a
no-op), and once more with the bookkeeping row forgotten, which re-applied over
the existing column and index and left exactly one of each; every
`scripts/tests/check_*.py`; `sh scripts/check-redaction.sh`;
`sh scripts/check-docs-recorded.sh`; no changed Go file over 350 lines (the
largest is `auth_email_flows.go`, 329). `insideout health` answered 0 against
both live servers with a matching `INSIDEOUT_ADDR`.

Not exercised. The real mail provider: no key was configured, so mail timing
is against an in-process stand-in and the live sender's dev mode, and the
provider's own failure modes were not observed. A real proxy chain: the
`X-Forwarded-For` chains were written by `curl` and by tests, never by the
Railway edge or nginx. `docker compose config`: docker is not installed here,
so the compose properties are proven by the ops-hardening guard (which reads
the file) and a YAML render (server `127.0.0.1:8080:8080`, hop count 0).
Several server replicas: the queue and the throttle are per process, as before.

New test files outside the literal `files_owned` globs, listed for the
verifier: `auth_mail_timing_test.go`, `auth_mail_failure_test.go`,
`auth_refresh_reuse_test.go`, `mail/queue_test.go` and
`store/sessions_family_test.go`.
