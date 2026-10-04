# Auth endpoints have no rate limiting or failed-attempt lockout

**Opened** 2026-08-24, deferred from the
[email-auth slice](../changelogs/2026-08-24-email-auth.md) to keep it
additive and pattern-matched to the existing password login.

**Resolved 2026-10-04** by remediation wave 2 — see
[the changelog](../changelogs/2026-10-04-remediation-wave-2.md): in-memory
per-IP + per-account throttle (`auth_throttle.go`: 8 failures → 15-min
credential lockout, 3 sends / 10 min / email+IP cap, per-IP volume caps,
uniform 429 + `Retry-After`) plus a 60 s per-email + per-IP resend cooldown
on the send endpoints (`auth_cooldown.go`). Locked responses are identical
for known and unknown emails, so no enumeration oracle.

## Problem

None of the auth request endpoints throttle by IP or email:

- `POST /auth/login` (pre-existing) — unlimited password guesses.
- `POST /auth/magic-link`, `POST /auth/forgot-password` (new) — unlimited
  link requests against any address; against a real account this is an
  email-bomb vector and Resend-spend drain, since each request to a real
  address sends mail.
- `POST /auth/magic-link/consume`, `POST /auth/reset-password` — token
  guesses are harmless cryptographically (256-bit, hashed, single-use,
  15/60-minute TTLs) but cost a DB write per probe.

Account enumeration is already closed (uniform 200s, mail failures logged
not surfaced), so this is purely about volume abuse.

## Target design (bounded fix prompt)

Docmind's shape, ported to the existing stack: a `failed_attempts` /
`locked_until` pair on a credentials-adjacent row, plus an in-memory or
`insideout`-schema per-email+IP counter with a short window. Lockout on
login after N failures (docmind: 8 → 15 min); a send cap (e.g. 3 links /
10 min / email) on magic-link and forgot-password; constant responses
retained so the fix doesn't reintroduce enumeration. Add gated store tests
for the lockout transitions and handler tests for the cap, and keep
`login_unconfirmed`-style errors uniform.

## Constraints

- Stay inside the `insideout` schema; no new trust surface.
- Don't make lockout observable as an account-existence oracle — the locked
  response must be indistinguishable for known and unknown emails.
