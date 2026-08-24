# 2026-08-24 — Email auth: magic link + password reset + code login

Status: **Completed** — original slice and the code-login addendum
implemented, verified, committed to `main`, and deployed to Railway on
2026-08-24 (`INSIDEOUT_PUBLIC_URL` set; new routes verified via the
domain). The only open item is operator-owned: the Resend key
(`INSIDEOUT_MAIL_API_KEY` + verified `INSIDEOUT_MAIL_FROM`, then redeploy
`server`) — until then mail runs in log-only dev mode.

## Why this exists

Email+password register and login already exist end-to-end (Go
`internal/api/auth.go` + Flutter `AuthForm`), but the two remaining standard
email flows are missing entirely:

1. **Magic link login** — no endpoint, no emailed-token storage, no email
   sender, no landing page.
2. **Password reset** — no forgot/reset endpoints, no token storage, no UI.
   A user who forgets a password has no recovery path at all.

The user asked for all four email flows to work (email login, magic link,
reset, register); two of the four need to be built, and the reference
implementation is `~/Projects/docmind` (Rust/Axum): opaque random tokens with
only SHA-256 hashes at rest, single-use expiring emailed codes, Resend HTTP
API for delivery, and reset → revoke-all-sessions.

## Scope

Backend (Go, stdlib + pgx, pattern-matched to `sessions.go`):

- Migration: `insideout.auth_tokens` — `(user_id, kind IN
  ('magic_link','password_reset'), token_hash UNIQUE, expires_at, used_at)`,
  FK to users with cascade. No RLS, matching `insideout.sessions` (trusted
  backend-only table). Issuing a token deletes the user's prior unused token
  of the same kind (one live link per kind).
- `internal/auth`: one-time-token generator (32 random bytes, URL-safe, plus
  sha256 hex — same shape as the refresh token helpers).
- `internal/store/auth_tokens.go`: `CreateAuthToken` (replace prior unused),
  `ConsumeAuthToken` (atomic `UPDATE ... WHERE used_at IS NULL AND
  expires_at > now() RETURNING user_id`; zero rows → `ErrNotFound`),
  `RevokeUserSessions`. Pre-auth context, no `withUserContext` (same
  precedent as `CreateUser`/`GetUserByEmail`).
- `internal/mail`: Resend `POST /emails` sender (from/to/subject/text/html,
  10s timeout). Empty `INSIDEOUT_MAIL_API_KEY` → dev mode: log the link at
  info level instead of sending (mirrors the offline-coach precedent).
- Config: `INSIDEOUT_MAIL_API_KEY`, `INSIDEOUT_MAIL_FROM`,
  `INSIDEOUT_PUBLIC_URL` (base for emailed links; defaults to
  `http://localhost:5173`, the Flutter dev origin).
- Endpoints (all JSON, same error contract as existing auth routes):
  - `POST /api/v1/auth/magic-link` `{email}` → always `200 {"ok":true}`
    (no account enumeration; sends a 15-minute link to
    `{PublicURL}/magic-link?token=...` when the account exists).
  - `POST /api/v1/auth/magic-link/consume` `{token}` → session (identical
    body to login: user fields + access/refresh tokens + cookies); invalid
    or expired → 401.
  - `POST /api/v1/auth/forgot-password` `{email}` → always `200 {"ok":true}`;
    60-minute link to `{PublicURL}/reset-password?token=...`.
  - `POST /api/v1/auth/reset-password` `{token, password}` → re-hash with
    argon2id, `UpdatePasswordHash`, revoke all the user's sessions, `200`.
- Tests: `DATABASE_URL`-gated store integration tests for the token
  lifecycle (consume/single-use/expiry/kind isolation/replacement +
  RevokeUserSessions), written first; mail sender unit test against an
  httptest server; handler validation tests where no store is needed.

Flutter client:

- `ApiClient`: `requestMagicLink`, `consumeMagicLink`, `forgotPassword`,
  `resetPassword`; `Session.consumeMagicLink` applies the login-shaped
  profile.
- `AuthForm` login mode gains a magic-link sub-mode ("Email me a sign-in
  link") and a "Forgot password?" link.
- New public routes: `/forgot-password`, `/reset-password?token=`,
  `/magic-link?token=` (consumes on load, then signs in).
- l10n strings for both locales; widget tests follow the existing
  `session_test` / `routes_test` shape.

Environment + docs:

- `scripts/env.sh` catalog + propagate; `docs/usage/environment.md`;
  backend architecture auth paragraph; changelog; this board.

## Out of scope

- Failed-attempt lockout / rate limiting on the new endpoints (parity with
  the existing login, which has none; record as a future hardening item).
- Email verification of new registrations (`email_verified_at` stays
  untouched), registration invitation policy, HTML template files (inline
  strings like docmind).
- Railway deployment + setting the Resend key in the Railway environment —
  needs the operator's API key; documented as the ship step.

## Engineering review

Self-review recorded in lieu of a separate pass (scope is additive and
pattern-matched to two existing, reviewed subsystems — sessions and auth
cookies):

- **Enumeration**: both request endpoints return a uniform 200 and send only
  when the account exists; consume/reset reveal token validity only (tokens
  are 256-bit random, not derivable).
- **Single-use + expiry enforced in SQL** (`UPDATE ... WHERE used_at IS NULL
  AND expires_at > now()`), so races can't double-consume — same
  affected-rows discipline as `RotateSession`.
- **Reset revokes sessions** (docmind `set_password` precedent) so a stolen
  refresh cookie dies with the password change.
- **No new trust surface**: `auth_tokens` follows `sessions` (backend-only,
  no RLS); mail failure is logged and never blocks the 200 response.
- Risk noted: emailed links go to the Flutter SPA, which consumes them via
  POST from the page — token appears in a URL query (history/log exposure is
  the standard accepted tradeoff for email links; 15/60-minute TTLs and
  single-use bound it).

## Checklist

- [x] Plan + board row
- [x] Store integration tests (fail first)
- [x] Migration + `auth_tokens` store implementation
- [x] Mail package + config keys
- [x] Magic-link request/consume handlers
- [x] Forgot/reset handlers
- [x] Go build/vet/tests green incl. `DATABASE_URL`-gated
- [x] Client API methods + session + auth-form magic mode
- [x] Client pages/routes/l10n
- [x] Flutter analyze + tests green
- [x] Live server exercise: all four flows end-to-end via curl
- [x] Changelog + env docs + backend.md + HANDOFF touch

## Addendum (same day): code login

The user then asked for the login surface to support **email + password,
magic link, and code login** — the third path (an emailed 6-digit code the
user types in, the docmind `DOCM-1234` device-code idea applied to login)
was not in the original slice. Same patterns, same table:

- Migration: `auth_tokens` gains kind `'login_code'` and an `attempts`
  column (new dated file — the 120000 migration is already applied locally,
  so the CHECK constraint is widened by ALTER, not rewritten).
- `internal/auth`: `GenerateLoginCode` — 6 digits from `crypto/rand`
  (unbiased), sha256-hex digest for storage via the existing helpers.
- `internal/store`: `VerifyLoginCode(userID, codeHash)` — correct code
  consumes atomically (same single-USE/expiry discipline as
  `ConsumeAuthToken`); a miss increments `attempts` on the user's live code
  and retires it at five misses, because a 6-digit code is low-entropy and
  must not be brute-forced inside its 10-minute TTL. Miss and no-active-code
  are indistinguishable (`ErrNotFound`).
- Handlers: `POST /api/v1/auth/login-code` `{email}` → uniform 200 (no
  enumeration, dev-mode mail logs the code); `POST /api/v1/auth/login-code/verify`
  `{email, code}` → login-shaped session; 401 "invalid or expired code" on
  wrong/exhausted/expired; 400 only for malformed input.
- Client: `requestLoginCode`/`verifyLoginCode` API methods, `Session.loginWithCode`,
  a third auth-form mode (email → send → enter code → verify, with resend),
  en/zh strings, widget test of the mode toggle.

Checklist:

- [x] Store integration tests (fail first): round trip, single use, expiry,
      attempt budget exhausts the code
- [x] Migration + `VerifyLoginCode`
- [x] `GenerateLoginCode` + handlers + validation tests
- [x] Go build/vet/tests green incl. gated
- [x] Client API/session/form/l10n + widget test; analyze + tests green
- [x] Live exercise via curl (dev-mode mail log carries the code)
- [x] Changelog + backend.md + HANDOFF/board closure
