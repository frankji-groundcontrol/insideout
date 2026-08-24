# 2026-08-24 — Email auth: magic-link login + password reset + code login

The user asked for the four standard email flows (login, magic link,
password reset, register). Register and email+password login were already
live end-to-end; this change adds the missing two, porting the token
pattern from `~/Projects/docmind` (Rust/Axum reference implementation):
opaque 256-bit tokens with only SHA-256 hashes at rest, single-use and
expiring, Resend HTTP API for delivery, and reset revoking every session.
A same-day follow-up request added the fifth surface — a typed 6-digit
**code login** — covered in the last section.

## Backend

- Migration `20260824120000_auth_email_tokens.sql`: `insideout.auth_tokens`
  (`user_id`, `kind` ∈ magic_link/password_reset, unique `token_hash`,
  `expires_at`, `used_at`; FK cascade). No RLS — the same trusted
  backend-only posture as `insideout.sessions`.
- `internal/store/auth_tokens.go`: `CreateAuthToken` (supersedes the user's
  prior unused token of the same kind — one live link per flow),
  `ConsumeAuthToken` (atomic `UPDATE … WHERE used_at IS NULL AND
  expires_at > now() RETURNING user_id`, zero rows → `ErrNotFound` — the
  same affected-rows race discipline as `RotateSession`),
  `RevokeUserSessions`.
- `internal/mail` (new): Resend `POST /emails` sender, 10 s timeout. Empty
  `INSIDEOUT_MAIL_API_KEY` → dev mode that logs the message (link included)
  instead of sending — the offline-coach degradation posture.
- `internal/api/auth_email_flows.go`: four endpoints, all under the
  existing error contract:
  - `POST /auth/magic-link` — uniform `200 {"ok":true}` (no account
    enumeration; mail failures are logged, not surfaced), 15-minute link to
    `{INSIDEOUT_PUBLIC_URL}/magic-link?token=…`.
  - `POST /auth/magic-link/consume` — session identical to login (user
    fields + tokens + cookies); invalid/expired/replayed → 401.
  - `POST /auth/forgot-password` — same uniform-200 contract; 60-minute
    link to `/reset-password?token=…`.
  - `POST /auth/reset-password` — argon2id re-hash, then revoke all the
    user's sessions so stolen cookies die with the old password.
- Config: `INSIDEOUT_MAIL_API_KEY`, `INSIDEOUT_MAIL_FROM` (default
  `InsideOut <onboarding@resend.dev>`), `INSIDEOUT_PUBLIC_URL` (default
  `http://localhost:5173`), documented in
  [usage/environment.md](../usage/environment.md) and `.env.example`;
  `scripts/env.sh` reports their defaults.

## Flutter client

- `ApiClient`: `requestMagicLink`, `consumeMagicLink`, `forgotPassword`,
  `resetPassword`; `Session.consumeMagicLink` signs in via the login-shaped
  response.
- Login form gains a magic-link sub-mode ("Email me a sign-in link" ↔
  "Use password instead") and a "Forgot password?" link; new public routes
  `/forgot-password`, `/reset-password?token=`, `/magic-link?token=` (the
  landing page consumes its token on load, then goes to `/dashboard`); all
  three reuse the `AuthDoor` ink-seal panel and are public in
  `auth_redirect`. l10n strings added to both catalogs.

## Verification

- TDD: `internal/store/auth_tokens_test.go` written first (build failure),
  7 real-Postgres tests green (round-trip, single-use, expiry, kind
  isolation, replacement, unknown hash, `RevokeUserSessions`), plus mail
  sender tests against httptest and handler validation tests. `go build`,
  `go vet`, `go test ./...` green. Test hashes are uuid-derived: rows
  persist after a run (the mkUser convention) while `token_hash` is
  globally unique, so fixed strings would collide on the second run —
  verified with `-count=2` after a first full-suite run caught exactly
  that.
- Live exercise against a locally booted server on the real database
  (mail dev mode; throwaway user, deleted after — cascade removed its
  tokens/sessions): register 201, password login 200, magic-link request
  200 (uniform for unknown email too), consume 200 with session then 401
  on replay, garbage token 401, forgot 200, reset 200 with old refresh
  token subsequently 401 (revoked), old password 401 / new password 200,
  reset-token replay 401.
- Client: `flutter analyze` (no new issues) and `flutter test` 47/47,
  including a new auth-form magic-mode widget test and updated route /
  redirect expectations.

## Code login (same-day addendum)

The user then asked for login to support email+password, magic link, and
**code login** — the docmind `DOCM-1234` device-code idea applied to login:
an emailed 6-digit code the user types into the form (the typeable
counterpart of the magic link, for clients where following a link is
awkward).

- Migration `20260824130000_auth_tokens_login_code.sql`: `auth_tokens`
  gains kind `login_code` (CHECK widened by ALTER — the 120000 migration
  was already applied) and an `attempts` column.
- `internal/auth/onetimetoken.go`: `GenerateLoginCode` — 6 digits from
  `crypto/rand.Int` (unbiased, unlike `% 1e6`), sha256-hex digest for
  storage.
- `internal/store`: `VerifyLoginCode(userID, codeHash)` — the exact hash
  consumes the code atomically (same single-use/expiry discipline as
  `ConsumeAuthToken`); a miss increments `attempts` on the user's live code
  and retires it at 5 misses, because a 6-digit code (~20 bits) must not be
  brute-forceable inside its 10-minute TTL. Miss / exhausted / expired /
  never-requested all return `ErrNotFound`.
- `POST /auth/login-code` `{email}` — uniform 200 (no enumeration);
  `POST /auth/login-code/verify` `{email, code}` — login-shaped session,
  uniform 401 "invalid or expired code" for anything but the right live
  code; 400 only for malformed input (email format, code not 6 digits).
- Client: `ApiClient.requestLoginCode`/`verifyLoginCode`,
  `Session.loginWithCode`, and a third login-form mode (send code → enter
  digits → verify, with resend); the login sub-modes were refactored from
  a `_magicMode` bool to a `password|magic|code` enum. en/zh strings added.

## Verification (addendum)

- TDD: 5 new real-Postgres tests in `auth_tokens_test.go` (round-trip +
  single-use, miss charges budget not the code, 5 misses retire the code,
  expiry, no active code) plus 5 handler validation cases; `go vet` and
  `go test ./...` green (store gated on `DATABASE_URL`; the remote-pooler
  latency needs `-timeout` above the 10-minute default for the full store
  package). `flutter analyze` no new issues; `flutter test` 48/48 with a
  new code-mode widget test.
- Live exercise against a locally booted server on the real database
  (mail dev mode; throwaway user, deleted after — cascade removed its
  tokens/sessions): request 200 (uniform for unknown email too), malformed
  code 400, five wrong verifies 401 then the right code 401 (budget
  retired it), re-request 200 → fresh code verifies to a login-shaped
  session (all keys, email match modulo server lowercasing), replay 401,
  wrong code on the spent token 401.

## Not done here

- Production mail key: everything else shipped 2026-08-24 — committed to
  `main`, `railway up --service server` + `--service app` (app started
  after the new server, so nginx re-resolved it), `INSIDEOUT_PUBLIC_URL`
  set to the domain, and the new routes verified live
  (`/api/v1/auth/login-code` → 200 `{"ok":true}`, malformed verify → 400,
  `/api/v1/auth/magic-link` → 200, `/api/v1/me` → 401). Remaining operator
  step: `INSIDEOUT_MAIL_API_KEY` + a Resend-verified `INSIDEOUT_MAIL_FROM`
  in the Railway `server` environment, then redeploy `server` — until then
  the sender runs in log-only dev mode (links/codes appear in server logs
  only).
- Failed-attempt lockout / rate limiting on the new endpoints (parity with
  the existing password login, which has none) — opened as
  [a dated issue](../issues/2026-08-24-auth-endpoints-no-rate-limiting.md)
  with a bounded fix prompt. The login code itself carries a per-token
  5-miss budget (it is brute-forceable in principle), so it does not wait
  on that issue. The test-rerun collision caught during verification is
  distilled as
  [a learning record](../learning/2026-08-24-gated-integration-tests-rerun-collisions.md).
- Email verification of new registrations (`email_verified_at` untouched).

Plan: [2026-08-24 email auth](../plans/2026-08-24-email-auth.md).
