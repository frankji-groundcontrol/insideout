-- Emailed one-time auth tokens: magic-link login and password reset.
-- token_hash is sha256 of the opaque token in the email link; single-use
-- (used_at flipped atomically at consume time) and expiring. Same trusted
-- backend-only posture as insideout.sessions: no RLS, no app-facing paths.
CREATE TABLE insideout.auth_tokens (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES insideout.users(id) ON DELETE CASCADE,
  kind text NOT NULL CHECK (kind IN ('magic_link', 'password_reset')),
  token_hash text NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  used_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON insideout.auth_tokens (user_id);
CREATE INDEX ON insideout.auth_tokens (expires_at);
