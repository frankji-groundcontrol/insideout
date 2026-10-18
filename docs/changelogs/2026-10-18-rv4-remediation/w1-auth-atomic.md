# Wave 1, atomic auth fixes — review-4 remediation

Branch `rv4/w1-auth-atomic`, `1dba3fe..8e39e72` (six commits: three red/fix
pairs), then this fragment. Findings are cited as `rv4-<ID>` because review
IDs restart per review. Rows 1.9, 1.10 and the cooldown half of 1.13 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md).
The apiclient half of 1.13 (rv4-QUAL-03, escaping every apiclient path segment
and query value) belongs to `w1-apiclient-escape` and is not touched here. No
address, host, secret or one-time code is spelled in this fragment.

## What changed

- **A burst of login-code guesses is capped at five** (rv4-SEC-01, row 1.9,
  red `09d801d`, fix `35f42e8`). `VerifyLoginCode` compared the hash in one
  statement and charged a miss in a second, so every guess already in flight
  when a burst began was compared for free before the first charge landed (the
  review measured 11 to 350 effective guesses instead of 5). It is now one
  conditional `UPDATE ... RETURNING token_hash = $hash`: the statement takes the
  attempt on the user's still-live code, retires the code on a match or on the
  fifth guess, and reports whether the hash matched. One request at a time the
  behaviour is unchanged: five misses retire the code, the right code as the
  fifth guess still verifies, and a miss, an exhausted code, an expired code and
  a code that was never requested all stay the same `ErrNotFound`.
- **Create-invitation says whether the e-mail left** (rv4-USER-07, row 1.10,
  red `0665a07`, fix `86644b9`). The 201 response gains `mailSent`, true only
  when the sender is enabled and the provider accepted the message. Dev mode
  (no mail key) logs the message and reports false; a provider rejection is
  logged as before and reports false. The invitation, the one-time code and the
  signup link come back either way and the status stays 201, so mail stays
  best-effort and the admin's fallback is the link. The admin dialog reads
  "The invitation email was not sent. Copy this signup link and send it to the
  invitee yourself." when the field is an explicit false; a response without
  the field (an older server) keeps the old copy, the same rule the client
  already applies to `authMailEnabled`. One key (`admin.codeNotEmailed`) was
  added to each of the English and Chinese tables. The CLI (`insideout invite`)
  and the MCP tool (`invite_create`) print the response untouched, so they carry
  the field with no code change.
- **The resend cooldown shares the auth throttle's client address** (rv4-QUAL-02,
  row 1.13 cooldown half, red `c1d07c2`, fix `8e39e72`). `cooldownClientIP` was
  a drifted copy of `clientIP`: it took the rightmost `X-Forwarded-For` entry
  even when that entry was empty, then fell back to `RemoteAddr`, which behind
  the proxy is the shared hop. Every client whose header ended in `, ` therefore
  shared one cooldown bucket, and the first of them silenced everyone else's
  sends behind the uniform 200 for 60 seconds. The copy is deleted and the three
  callers (magic link, forgot password, login code) use `clientIP`. The shared
  table test gains the plan's `"9.9.9.9, "` case.

## Design note (row 1.9)

Row 1.9 is not marked `dir`; this records why the obvious shapes lost.

- **One conditional `UPDATE` (chosen).** The row lock serialises concurrent
  guesses, and under READ COMMITTED a waiting `UPDATE` re-evaluates both the
  `WHERE` clause and the `SET` expressions against the latest row version once
  the lock is released. Any concurrent guess past the fifth therefore finds
  `attempts < 5` false and matches no row. The attempt is taken in the same
  statement that compares, so the slot is reserved before the answer exists,
  which is the "reserve before verification" half of the row.
- **`SELECT ... FOR UPDATE` in a transaction, then compare and update.**
  Correct, but two round trips and a transaction to hold a lock across: more
  code for the same guarantee.
- **An advisory lock or an in-process mutex per user.** An advisory lock needs
  a key scheme and a release on every path; a mutex only protects one process,
  while the database is the one place every replica meets.
- **Reserving the slot in `auth_throttle.go`.** Out of this cluster's files and
  the wrong layer: the middleware counts failed `login-code/verify` responses
  after they are written, so it cannot make a statement atomic. It is
  unchanged; the store now enforces the five-guess budget however many requests
  the middleware lets through.

How the test forces the race. Scheduler luck would let a burst test pass on the
broken code and flake on the fixed one, so the test removes the luck. It opens
a second store whose pool holds exactly one connection and holds that
connection itself. The pool queues waiters first-in-first-out, so every guess
parks in launch order, and the test waits (polling goroutine stacks, so no
fixed delay stands in for ordering) until each one is parked before launching
the next. Releasing the connection
then runs the statements strictly in queue order. The old code ran every
comparison before any charge, because each charge re-queued behind all the
other comparisons, so a correct code queued as guess 21 won. The new code
charges each guess as it compares and rejects it.

## The `mailSent` contract

- Additive: no field was renamed or removed and the status is still 201.
- `true` means the sender is enabled **and** the provider accepted the message.
  `false` means nothing left (dev mode) or the provider refused it.
- Only an explicit `false` makes the dialog warn. An absent field keeps the old
  copy, so a client newer than its server never claims an e-mail failed.

## Known ceilings, recorded on purpose

- **A new code resets the budget.** `CreateAuthToken` supersedes the user's
  unused code, so a re-request starts a fresh five guesses. The 60-second
  resend cooldown (per e-mail and per IP) and the 10-minute code lifetime bound
  how often a fresh budget is issued, and the `login-code/verify` throttle
  bounds the failed guesses per client. This change alters none of the three.
- **`mailSent` means the provider accepted the message,** not that it reached
  an inbox. A later bounce is invisible to the server, and the dialog keeps the
  link visible for that reason.
- **The mail test swaps `http.DefaultTransport`.** The sender's HTTP client has
  no transport of its own, so the test answers its call with an in-process fake
  provider and restores the transport in `t.Cleanup`. That is race-free only
  while package `api` runs its tests sequentially, which it does today. If it
  ever adopts `t.Parallel`, give the sender an injectable transport first;
  `internal/mail` was not in this cluster's files.
- **The burst test leans on two pool details:** waiters are served first in,
  first out, and a parked caller shows `semaphore.(*Weighted).Acquire` in its
  goroutine stack. If an upgrade renames that wait path the test fails with
  "only N of M guesses reached the pool queue". If it stopped serving waiters in
  order, the test would stay green on the fixed code but could no longer
  reproduce the old race, which is why the mutation check below is listed.

## Left for the integrator

Documentation, comment and index edits only; no code is waiting.

- `docs/usage/cli-mcp-and-guidance.md` (the signup-policy paragraph, line 105)
  and `SKILL.md` (line 54): one sentence that `insideout invite <email>` and
  `invite_create` now print `mailSent`, and that false means nothing was
  e-mailed, so send `signupUrl` yourself. Both sentences stay true without it.
- `server/internal/apiclient/auth.go` line 146 (a comment, in
  `w1-apiclient-escape`'s file): "(JSON: invitation + code + signupUrl)" becomes
  "(JSON: invitation + code + signupUrl + mailSent)".
- `server/internal/mcpserver/product_tools.go` line 224 (`w1-misc-correctness`'s
  file; a string, not a comment): the `invite_create` tool description may add
  that `mailSent` false means the invitee was not e-mailed.
- `docs/architecture/backend.md` lines 75 and 76 ("a 5-miss attempt budget")
  stay true; no edit.
- The open-findings record and `docs/issues/README.md`: per-ID dispositions for
  rv4-SEC-01, rv4-USER-07 and rv4-QUAL-02 (all `FIXED`); this folder's
  `index.md` wave-1 row; the changelog index files.

**Closed (integrator follow-up, wave 1).** The `mailSent` sentences are in
`docs/usage/cli-mcp-and-guidance.md` and `SKILL.md`, and the comment in
`server/internal/apiclient/auth.go` names `mailSent`. Still open at wave 1: the
`invite_create` tool description in `product_tools.go` (R1-03 in the
[open-findings record](../../issues/2026-10-18-review-4-open-findings.md);
closed in wave 2 by `f9e9894`).

## Verification

Run against a private PostgreSQL 18 cluster on a port of its own, migrated from
this worktree, with a DSN that exists only for the run. A skipped database test
counts as red; none skipped in the scoped runs below.

| Finding | Red, at the red commit | Green, at the fix commit |
| --- | --- | --- |
| rv4-SEC-01 | `TestLoginCodeBurstCappedAtMaxAttempts` fails 10 of 10 runs under `-race`: "correct code as guess 21: err = <nil>; a burst got 21 free evaluations, the cap is 5" and "after the burst attempts = 0 retired = true, want attempts = 5 retired = true" | passes 10 of 10, and 5 of 5 under `-race`. Two guards come with the fix: the correct code as the fifth guess still verifies inside a burst, and eight racing submissions of the correct code yield exactly one success |
| rv4-USER-07 | server: `TestAdminCreateInvitationReportsMailSent` fails all three subtests with "response has no mailSent field"; client: `admin_section_not_emailed_test.dart` finds 0 widgets containing "was not sent", and a scratch probe showed the old "was emailed" copy rendering | all three server subtests pass; the three client widget tests pass, including the two guards (true keeps the old copy, an absent field makes no claim) |
| rv4-QUAL-02 | `TestSendCooldownKeysOnClientIPPastTrailingEmptyXFF` fails on all three endpoints: "second client: status = 200, want 500" | passes on all three; `TestClientIPPrefersRightmostXForwardedFor` gained the `"9.9.9.9, "` case |

Mutation checks a verifier can run; each turns the named test red again and was
undone afterwards:

- rv4-SEC-01: restore the two-statement body of `VerifyLoginCode`. Both burst
  tests go red; the single-use test stays green on either body (it guards the
  rewrite, it does not detect the old race).
- rv4-USER-07, server: `"mailSent": true` fails the dev-mode and the
  provider-rejects subtests; `s.mail.Enabled()` alone fails provider-rejects;
  `sendErr == nil` alone fails dev mode. Client: ignore the field, or read
  `mailSent != true`, and the not-emailed and the old-server tests go red.
- rv4-QUAL-02: switch any one of the three callers back to `r.RemoteAddr`, and
  that endpoint's subtest goes red.

Gates, all green on this branch: `go build ./...` and `go vet ./...`; `gofmt -l`
prints nothing; `go test ./internal/store/... -run 'AuthToken|LoginCode' -count=1
-race` (14 passed, 0 skipped); `go test ./internal/api/... -count=1 -skip
TestProjectQuietFlagged` (120 passed, 0 failed, 0 skipped); the whole server
suite once at the end with `-race -count=1` (26 packages ok);
`flutter pub get --enforce-lockfile`, `flutter analyze --no-fatal-infos`
(exit 0; the 14 infos are the same count as on the base and none are in these
files), the two admin widget tests, the whole client suite (461 passed) and the
same suite under `TZ=Asia/Shanghai`; `git diff --stat -- client/pubspec.lock`
empty; every `scripts/tests/check_*.py`; `sh scripts/check-redaction.sh`;
`sh scripts/check-docs-recorded.sh`; every touched file at most 350 lines.
`TestProjectQuietFlagged` is red on the base and belongs to `w1-clock-and-runes`,
so it is skipped by name. No CI-config row is in this cluster, so nothing is
pending a first CI run.
